package main

import (
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	uploadLimit := 1 << 30
	http.MaxBytesReader(w, r.Body, int64(uploadLimit))

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the owner of this video", err)
		return
	}

	multipartFile, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Video not present in request", err)
		return
	}
	defer multipartFile.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing media type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusInternalServerError, "Video must be an mp4", err)
		return
	}

	tempVideoFile, err := createTempVideoFile(multipartFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing video", err)
		return
	}
	defer os.Remove(tempVideoFile.Name())
	defer tempVideoFile.Close()

	aspectRatio, err := getVideoAspectRatio(tempVideoFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error calculating video aspect ratio", err)
		return
	}

	objectKey := filepath.Join(getAspectRatioName(aspectRatio), getAssetPath(mediaType))
	putObjectParams := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &objectKey,
		Body:        tempVideoFile,
		ContentType: &mediaType,
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &putObjectParams)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error putting object in bucket", err)
		return
	}

	videoURL := cfg.getBucketObjectURL(objectKey)
	video.UpdatedAt = time.Now()
	video.VideoURL = &videoURL

	signedVideo, err := cfg.dbVideoToSignedVideo(video)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Couldn't convert video to signed video", err)
		return
	}

	err = cfg.db.UpdateVideo(signedVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, signedVideo)
}

func createTempVideoFile(multipartFile multipart.File) (*os.File, error) {
	tempVideoFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		return nil, fmt.Errorf("Error creating temporary file: %w", err)
	}
	defer os.Remove(tempVideoFile.Name())
	defer tempVideoFile.Close()

	_, err = io.Copy(tempVideoFile, multipartFile)
	if err != nil {
		return nil, fmt.Errorf("Error copying video bytes to new file: %w", err)
	}
	tempVideoFile.Seek(0, io.SeekStart)

	processedVideoPath, err := processVideoForFastStart(tempVideoFile.Name())
	if err != nil {
		return nil, fmt.Errorf("Error pre-processing video for fast start: %w", err)
	}

	processedVideoFile, err := os.Open(processedVideoPath)
	if err != nil {
		return nil, fmt.Errorf("Error creating temporary file for processed video: %w", err)
	}

	return processedVideoFile, nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputFilePath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy",
		"-movflags", "faststart", "-f", "mp4", outputFilePath)

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return outputFilePath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignedClient := s3.NewPresignClient(s3Client)
	params := s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	presignedRequest, err := presignedClient.PresignGetObject(context.Background(), &params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}

	return presignedRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}

	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) < 2 {
		return video, nil
	}

	bucket, key := parts[0], parts[1]
	presignedUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*5)
	if err != nil {
		return database.Video{}, nil
	}

	video.VideoURL = &presignedUrl
	return video, nil
}

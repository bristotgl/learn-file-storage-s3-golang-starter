package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing multipart request body", err)
		return
	}

	multipartFile, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Thumbnail not present in request ", err)
		return
	}

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing media type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusInternalServerError, "Thumbnail must be either a jpeg or a png", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found for thumbnail ", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You are not the owner of this video", err)
		return
	}

	assetPath := getAssetPath(mediaType)
	assetDiskPath := cfg.getAssetDiskPath(assetPath)

	thumbnailFile, err := os.Create(assetDiskPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating file for storing thumbnail", err)
		return
	}
	defer thumbnailFile.Close()

	_, err = io.Copy(thumbnailFile, multipartFile)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error copying thumbnail bytes to new file", err)
		return
	}

	url := cfg.getAssetURL(assetPath)
	video.ThumbnailURL = &url
	video.UpdatedAt = time.Now()
	if err := cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating video with thumbnail", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}

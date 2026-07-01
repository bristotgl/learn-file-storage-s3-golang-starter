package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getExtFromMediaType(mediaType string) string {
	return strings.Split(mediaType, "/")[1]
}

func getBase64Filename() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func getAssetPath(mediaType string) string {
	filename := getBase64Filename()
	ext := getExtFromMediaType(mediaType)
	return fmt.Sprintf("%s.%s", filename, ext)
}

func (cfg *apiConfig) getAssetDiskPath(assetPath string) string {
	return filepath.Join(cfg.assetsRoot, assetPath)
}

func (cfg *apiConfig) getAssetURL(assetPath string) string {
	return fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, assetPath)
}

func (cfg *apiConfig) getBucketObjectURL(key string) string {
	return fmt.Sprintf("%s,%s", cfg.s3Bucket, key)
}

func getAspectRatioName(formattedRatio string) string {
	if formattedRatio == "9:16" {
		return "portrait"
	}

	if formattedRatio == "16:9" {
		return "landscape"
	}

	return "other"
}

func getVideoAspectRatio(filePath string) (string, error) {
	type ffpegResult struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	buffer := new(bytes.Buffer)
	cmd.Stdout = buffer
	if err := cmd.Run(); err != nil {
		return "", err
	}

	cmdResult := ffpegResult{}
	if err := json.Unmarshal(buffer.Bytes(), &cmdResult); err != nil {
		return "", err
	}

	return formatAspectRatio(cmdResult.Streams[0].Width, cmdResult.Streams[0].Height), nil
}

func formatAspectRatio(width, height int) string {
	const tolerance = 0.05
	ratio := float64(width) / float64(height)

	if math.Abs(ratio-9.0/16.0) < tolerance {
		return "9:16"
	}

	if math.Abs(ratio-16.0/9.0) < tolerance {
		return "16:9"
	}

	return "other"
}

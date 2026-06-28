package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
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
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
}

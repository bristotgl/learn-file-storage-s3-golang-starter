package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
)

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

package utils

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"meditour/cdn/config"

	"github.com/google/uuid"
)

// DetectMimeType reads the first 512 bytes of a reader to determine its MIME type
func DetectMimeType(r io.Reader, filename string) (string, error) {
	buffer := make([]byte, 512)
	n, err := r.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	contentType := http.DetectContentType(buffer[:n])

	// Fallback to extension if generic type detected
	if contentType == "application/octet-stream" || contentType == "text/plain; charset=utf-8" {
		ext := filepath.Ext(filename)
		if extType := mime.TypeByExtension(ext); extType != "" {
			// Clean up extra info like charset if needed
			return strings.Split(extType, ";")[0], nil
		}
	}

	return strings.Split(contentType, ";")[0], nil
}

func IsAllowedExtension(filename string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	for _, allowed := range config.AppConfig.AllowedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}

func IsImageExtension(filename string) bool {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), ".")) {
	case "jpg", "jpeg", "png", "gif", "webp":
		return true
	default:
		return false
	}
}

func GenerateUniqueFileName(originalName string) string {
	ext := filepath.Ext(originalName)
	timestamp := time.Now().Format("2006/01/02") // Subfolders for better structure
	newID := uuid.New().String()
	return fmt.Sprintf("%s/%s%s", timestamp, newID, ext)
}

func GetFullUrl(path string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(config.AppConfig.BaseUrl, "/"), strings.TrimPrefix(path, "/"))
}

func GetPathUrl(path string) string {
	normalizedPath := filepath.ToSlash(path)
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(config.AppConfig.UploadPath, "/"), strings.TrimPrefix(normalizedPath, "/"))
}

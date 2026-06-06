package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"meditour/cdn/config"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
)

// ServeFileHandler handles serving both static and processed images
func ServeFileHandler(c *gin.Context) {
	// Extract filepath from URL
	// The route is /uploads/*filepath, so c.Param("filepath") will be like "/2026/03/06/uuid.jpg"
	relPath, err := sanitizeRelativePath(c.Param("filepath"), false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	fullOriginalPath, err := resolvePathWithinRoot(config.AppConfig.UploadDir, relPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}

	// Check if original file exists
	if _, err := os.Stat(fullOriginalPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Get query parameters
	wStr := c.Query("w")
	hStr := c.Query("h")
	fmtStr := strings.ToLower(c.Query("fmt"))

	// If no processing parameters, serve the original file
	if wStr == "" && hStr == "" && fmtStr == "" {
		// Set cache control for static files
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.File(fullOriginalPath)
		return
	}

	// Parse dimensions
	width, _ := strconv.Atoi(wStr)
	height, _ := strconv.Atoi(hStr)

	// Determine output format and extension
	ext := strings.ToLower(filepath.Ext(relPath))
	targetExt := ext
	if fmtStr != "" {
		targetExt = "." + fmtStr
	}

	// Construct cache filename and path
	// Example: cache/2026/03/06/uuid_w200_h200.webp
	fileNameWithoutExt := strings.TrimSuffix(filepath.Base(relPath), ext)
	dirPath := filepath.Dir(relPath)

	cacheFileName := fmt.Sprintf("%s_w%d_h%d%s", fileNameWithoutExt, width, height, targetExt)
	cacheRelativePath := filepath.Join(dirPath, cacheFileName)
	fullCachePath, err := resolvePathWithinRoot(config.AppConfig.CacheDir, cacheRelativePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cache path"})
		return
	}

	// Check if cached version already exists
	if _, err := os.Stat(fullCachePath); err == nil {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("X-Cache", "HIT")
		c.File(fullCachePath)
		return
	}

	// Process image
	img, err := imaging.Open(fullOriginalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open original image: " + err.Error()})
		return
	}

	// Resize if dimensions are provided
	if width > 0 || height > 0 {
		// Resize and maintain aspect ratio if one dimension is 0
		img = imaging.Resize(img, width, height, imaging.Lanczos)
	}

	// Ensure cache subdirectories exist
	if err := os.MkdirAll(filepath.Dir(fullCachePath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cache directory: " + err.Error()})
		return
	}

	// Save processed image to cache
	// Imaging supports saving based on extension
	var saveErr error
	switch targetExt {
	case ".jpg", ".jpeg":
		saveErr = imaging.Save(img, fullCachePath, imaging.JPEGQuality(85))
	case ".png":
		saveErr = imaging.Save(img, fullCachePath)
	case ".webp":
		saveErr = imaging.Save(img, fullCachePath) // Webp support depends on binary usually, but imaging uses golang.org/x/image/webp
	case ".gif":
		saveErr = imaging.Save(img, fullCachePath)
	default:
		// If unknown format requested, just use original extension
		fullCachePath = strings.TrimSuffix(fullCachePath, targetExt) + ext
		saveErr = imaging.Save(img, fullCachePath)
	}

	if saveErr != nil {
		// Fallback to serving original if processing/saving fails
		c.File(fullOriginalPath)
		return
	}

	// Serve processed file
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("X-Cache", "MISS")
	c.File(fullCachePath)
}

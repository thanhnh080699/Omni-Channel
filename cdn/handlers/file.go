package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"meditour/cdn/config"

	"github.com/gin-gonic/gin"
)

type DeleteFileRequest struct {
	Path string `json:"path" query:"path" binding:"required"`
}

func DeleteFileHandler(c *gin.Context) {
	// Try to get path from query or body
	path := c.Query("path")
	if path == "" {
		var req DeleteFileRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			path = req.Path
		}
	}

	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File path is required"})
		return
	}

	cleanPath, err := sanitizeRelativePath(path, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	fullPath, err := resolvePathWithinRoot(config.AppConfig.UploadDir, cleanPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	// Check if exists and is not a directory
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File does not exist"})
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Path is a directory, use folder deletion instead"})
		return
	}

	// Delete file
	if err := os.Remove(fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file: " + err.Error()})
		return
	}

	// Cleanup cache for this specific file
	// Cache files follow pattern: path_w{w}_h{h}.ext
	// We should probably find all files that start with the filename in the cache dir
	cacheDir, err := resolvePathWithinRoot(config.AppConfig.CacheDir, filepath.Dir(cleanPath))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
		return
	}
	if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
		files, _ := os.ReadDir(cacheDir)
		prefix := strings.TrimSuffix(filepath.Base(cleanPath), filepath.Ext(cleanPath))
		ext := filepath.Ext(cleanPath)
		for _, f := range files {
			if strings.HasPrefix(f.Name(), prefix+"_") && strings.HasSuffix(f.Name(), ext) {
				_ = os.Remove(filepath.Join(cacheDir, f.Name()))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

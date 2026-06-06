package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"meditour/cdn/config"

	"github.com/gin-gonic/gin"
)

type CreateFolderRequest struct {
	Path string `json:"path" binding:"required"`
}

func CreateFolderHandler(c *gin.Context) {
	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: path is required"})
		return
	}

	cleanPath, err := sanitizeRelativePath(req.Path, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}

	fullPath, err := resolvePathWithinRoot(config.AppConfig.UploadDir, cleanPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: " + err.Error()})
		return
	}

	// Create the directory (and any parent directories)
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory: " + err.Error()})
		return
	}

	// Also ensure a cache counterpart exists or is clean if needed?
	// Usually not necessary for explicit folder creation.

	c.JSON(http.StatusOK, gin.H{
		"message":   "Folder created successfully",
		"path":      cleanPath,
		"full_path": fullPath,
	})
}

type DeleteFolderRequest struct {
	Path string `json:"path" binding:"required"`
}

func DeleteFolderHandler(c *gin.Context) {
	var req DeleteFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: path is required"})
		return
	}

	cleanPath, err := sanitizeRelativePath(req.Path, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	fullPath, err := resolvePathWithinRoot(config.AppConfig.UploadDir, cleanPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	// Check if exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder does not exist"})
		return
	}

	// Remove All (equivalent to rm -rf)
	if err := os.RemoveAll(fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete folder: " + err.Error()})
		return
	}

	// Also cleanup cache for this path
	fullCachePath, err := resolvePathWithinRoot(config.AppConfig.CacheDir, cleanPath)
	if err == nil {
		_ = os.RemoveAll(fullCachePath)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Folder deleted successfully",
	})
}

type RenameFolderRequest struct {
	OldPath string `json:"old_path" binding:"required"`
	NewPath string `json:"new_path" binding:"required"`
}

func RenameFolderHandler(c *gin.Context) {
	var req RenameFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: old_path and new_path are required"})
		return
	}

	oldPath, err := sanitizeRelativePath(req.OldPath, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	newPath, err := sanitizeRelativePath(req.NewPath, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	fullOldPath, err := resolvePathWithinRoot(config.AppConfig.UploadDir, oldPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	fullNewPath, err := resolvePathWithinRoot(config.AppConfig.UploadDir, newPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path or permission denied"})
		return
	}

	// Check if old folder exists
	if _, err := os.Stat(fullOldPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source folder does not exist"})
		return
	}

	// Check if new folder already exists
	if _, err := os.Stat(fullNewPath); err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Destination folder already exists"})
		return
	}

	// Ensure parent directories of new path exist
	if err := os.MkdirAll(filepath.Dir(fullNewPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination parent directories: " + err.Error()})
		return
	}

	// Perform physical rename
	if err := os.Rename(fullOldPath, fullNewPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rename folder: " + err.Error()})
		return
	}

	// Also attempt to rename cache folder if it exists
	fullOldCachePath, oldCacheErr := resolvePathWithinRoot(config.AppConfig.CacheDir, oldPath)
	fullNewCachePath, newCacheErr := resolvePathWithinRoot(config.AppConfig.CacheDir, newPath)
	if oldCacheErr == nil && newCacheErr == nil {
		if _, err := os.Stat(fullOldCachePath); err == nil {
			_ = os.MkdirAll(filepath.Dir(fullNewCachePath), 0755)
			_ = os.Rename(fullOldCachePath, fullNewCachePath)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Folder renamed successfully",
		"old_path": oldPath,
		"new_path": newPath,
	})
}

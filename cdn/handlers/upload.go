package handlers

import (
	"fmt"
	"image"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"meditour/cdn/config"
	"meditour/cdn/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadResponse struct {
	OriginalName string `json:"original_name"`
	FileName     string `json:"file_name"`
	Path         string `json:"path"`
	Url          string `json:"url"`
	Size         int64  `json:"size"`
	MimeType     string `json:"mime_type"`
}

func SingleUploadHandler(c *gin.Context) {
	// Set max upload size
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, config.AppConfig.MaxUploadSize)

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from request: " + err.Error()})
		return
	}

	// Validate extension (Initial check)
	if !utils.IsAllowedExtension(file.Filename) {
		allowed := strings.Join(config.AppConfig.AllowedExtensions, ", ")
		c.JSON(http.StatusBadRequest, gin.H{"error": "File extension not allowed. Allowed: " + allowed})
		return
	}

	// Deep Inspection: Check actual file content
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file: " + err.Error()})
		return
	}
	defer src.Close()

	// Deep Inspection is optional, but for now we'll allow anything that matches the extension filter
	// If you want to enable strict mime-type check you can do it here.
	// We'll keep the DetectMimeType call just to get the mime type for response, but remove the strict check.
	mimeType, _ := utils.DetectMimeType(src, file.Filename)

	if utils.IsImageExtension(file.Filename) {
		imageFile, openErr := file.Open()
		if openErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to inspect file: " + openErr.Error()})
			return
		}
		defer imageFile.Close()

		if _, _, decodeErr := image.DecodeConfig(imageFile); decodeErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Uploaded file is not a valid image"})
			return
		}
	}

	// Use folder from request or default to root
	folderPath, err := sanitizeRelativePath(c.PostForm("folder"), true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder path"})
		return
	}

	fullUploadDir, err := resolvePathWithinRoot(config.AppConfig.UploadDir, folderPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder path"})
		return
	}

	// Ensure target directory exists
	if _, err := os.Stat(fullUploadDir); os.IsNotExist(err) {
		os.MkdirAll(fullUploadDir, 0755)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	fileName := uuid.New().String() + ext
	savePath := filepath.Join(fullUploadDir, fileName)
	relativePath := filepath.ToSlash(filepath.Join(folderPath, fileName))

	// Save file
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	// Prepare response
	res := UploadResponse{
		OriginalName: file.Filename,
		FileName:     fileName,
		Path:         relativePath,
		Url:          utils.GetPathUrl(relativePath),
		Size:         file.Size,
		MimeType:     mimeType,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"data":    res,
	})
}

// MultiUploadHandler handles multiple files
func MultiUploadHandler(c *gin.Context) {
	form, _ := c.MultipartForm()
	files := form.File["files"]

	var responses []UploadResponse
	folderPath, err := sanitizeRelativePath(c.PostForm("folder"), true)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder path"})
		return
	}

	fullUploadDir, err := resolvePathWithinRoot(config.AppConfig.UploadDir, folderPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder path"})
		return
	}

	if _, err := os.Stat(fullUploadDir); os.IsNotExist(err) {
		os.MkdirAll(fullUploadDir, 0755)
	}

	for _, file := range files {
		if !utils.IsAllowedExtension(file.Filename) {
			continue // Skip invalid extensions in multi-upload
		}

		// Deep Inspection
		src, err := file.Open()
		if err != nil {
			continue
		}
		mimeType, _ := utils.DetectMimeType(src, file.Filename)
		src.Close()

		if utils.IsImageExtension(file.Filename) {
			imageFile, openErr := file.Open()
			if openErr != nil {
				continue
			}
			_, _, decodeErr := image.DecodeConfig(imageFile)
			imageFile.Close()
			if decodeErr != nil {
				continue
			}
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		fileName := uuid.New().String() + ext
		savePath := filepath.Join(fullUploadDir, fileName)
		relativePath := filepath.ToSlash(filepath.Join(folderPath, fileName))

		if err := c.SaveUploadedFile(file, savePath); err != nil {
			continue
		}

		res := UploadResponse{
			OriginalName: file.Filename,
			FileName:     fileName,
			Path:         relativePath,
			Url:          utils.GetPathUrl(relativePath),
			Size:         file.Size,
			MimeType:     mimeType,
		}
		responses = append(responses, res)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Uploaded %d files", len(responses)),
		"data":    responses,
	})
}

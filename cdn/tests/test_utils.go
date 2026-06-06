package tests

import (
	"os"

	"meditour/cdn/config"
	"meditour/cdn/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupTestRouter creates a gin engine with all routes for testing
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	// Mock Config if not loaded
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{
			Port:              "8081",
			UploadDir:         "./test_uploads",
			CacheDir:          "./test_uploads/cache",
			AllowedExtensions: []string{"jpg", "jpeg", "png", "gif", "webp"},
			MaxUploadSize:     10 * 1024 * 1024,
			ApiKey:            "test_key",
			SignatureKey:      "test_sig_key",
			AllowedOrigins:    []string{"*"},
			RateLimitRPS:      1000, // Very high for tests
			RateLimitBurst:    2000,
		}
	}

	// Reset rate limiters for each test
	handlers.ResetLimiters()

	r := gin.Default()

	// Middlewares
	r.Use(handlers.RateLimitMiddleware())

	// CORS Configuration
	corsConfig := cors.DefaultConfig()
	if len(config.AppConfig.AllowedOrigins) == 1 && config.AppConfig.AllowedOrigins[0] == "*" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = config.AppConfig.AllowedOrigins
	}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "X-API-KEY", "Authorization"}
	r.Use(cors.New(corsConfig))

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	// Serving
	r.GET("/uploads/*filepath", handlers.SignatureMiddleware(), handlers.ServeFileHandler)

	// API
	api := r.Group("/api")
	api.Use(handlers.AuthMiddleware())
	{
		api.POST("/upload", handlers.SingleUploadHandler)
		api.POST("/uploads", handlers.MultiUploadHandler)
		api.POST("/folder", handlers.CreateFolderHandler)
		api.PUT("/folder", handlers.RenameFolderHandler)
		api.DELETE("/folder", handlers.DeleteFolderHandler)
	}

	return r
}

// CleanupTestDirs removes directories created during testing
func CleanupTestDirs() {
	os.RemoveAll("./test_uploads")
}

// SetupTestDirs ensures directories exist for testing
func SetupTestDirs() {
	os.MkdirAll("./test_uploads/cache", 0755)
}

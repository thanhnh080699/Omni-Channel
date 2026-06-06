package main

import (
	"log"
	"net/http"

	"meditour/cdn/config"
	"meditour/cdn/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize config
	config.LoadConfig()

	// Default mode for Gin
	if config.AppConfig.AppDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.MaxMultipartMemory = 8 << 20 // 8 MiB

	// Rate Limiting (Global)
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

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "UP",
			"info":   "Meditour CDN Service",
		})
	})

	uploadPath := config.AppConfig.UploadPath

	// File serving with on-the-fly processing and caching
	// Support for Signed URLs
	r.GET(uploadPath+"/*filepath", handlers.SignatureMiddleware(), handlers.ServeFileHandler)

	// API Routes (Authenticated)
	api := r.Group("/api")
	api.Use(handlers.AuthMiddleware())
	{
		api.POST("/upload", handlers.SingleUploadHandler)
		api.POST("/uploads", handlers.MultiUploadHandler)
		// Folder management
		api.POST("/folder", handlers.CreateFolderHandler)
		api.PUT("/folder", handlers.RenameFolderHandler)
		api.DELETE("/folder", handlers.DeleteFolderHandler)
		// File management
		api.DELETE("/file", handlers.DeleteFileHandler)
	}

	// Welcome page
	r.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, "<h1>Meditour CDN</h1><p>Running and ready for uploads.</p>")
	})

	log.Printf("Starting Meditour CDN on port %s", config.AppConfig.Port)
	log.Printf("Files are stored in: %s", config.AppConfig.UploadDir)
	log.Printf("Publicly accessible at: %s:%s%s", config.AppConfig.BaseUrl, config.AppConfig.Port, uploadPath)

	if err := r.Run(":" + config.AppConfig.Port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

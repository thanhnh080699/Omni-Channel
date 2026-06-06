package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	AppDebug          bool
	UploadDir         string
	UploadPath        string
	AllowedExtensions []string
	MaxUploadSize     int64
	ApiKey            string
	BaseUrl           string
	AllowedOrigins    []string
	CacheDir          string
	RateLimitRPS      float64
	RateLimitBurst    int
	RequireSignature  bool
	SignatureKey      string
}

var AppConfig *Config

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, using default environment variables")
	}

	maxSize, _ := strconv.ParseInt(getEnv("MAX_UPLOAD_SIZE", "104857600"), 10, 64)
	apiKey := os.Getenv("API_KEY")
	signatureKey := os.Getenv("SIGNATURE_KEY")

	if strings.TrimSpace(apiKey) == "" {
		log.Fatal("API_KEY is required")
	}

	if strings.TrimSpace(signatureKey) == "" {
		log.Fatal("SIGNATURE_KEY is required")
	}

	AppConfig = &Config{
		Port:              getEnv("PORT", "8081"),
		AppDebug:          getEnvBool("APP_DEBUG", "true"),
		UploadDir:         getEnv("UPLOAD_DIR", "./uploads"),
		UploadPath:        getEnv("UPLOAD_PATH", "/medias/"),
		AllowedExtensions: strings.Split(getEnv("ALLOWED_EXTENSIONS", "jpg,jpeg,png,gif,webp,svg,mp4,mov,avi,webm,pdf,doc,docx,xls,xlsx,ppt,pptx,txt"), ","),
		MaxUploadSize:     maxSize,
		ApiKey:            apiKey,
		BaseUrl:           getEnv("BASE_URL", "http://localhost:8081"),
		AllowedOrigins:    strings.Split(getEnv("ALLOW_ORIGINS", "*"), ","),
		CacheDir:          getEnv("CACHE_DIR", "./uploads/cache"),
		RateLimitRPS:      getEnvFloat("RATE_LIMIT_RPS", 5.0),
		RateLimitBurst:    getEnvInt("RATE_LIMIT_BURST", 10),
		RequireSignature:  getEnvBool("REQUIRE_SIGNATURE", "false"),
		SignatureKey:      signatureKey,
	}

	// Ensure upload dir exists
	ensureDir(AppConfig.UploadDir)
	// Ensure cache dir exists
	ensureDir(AppConfig.CacheDir)
}

func ensureDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		if err != nil {
			log.Fatalf("Failed to create directory %s: %v", path, err)
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		f, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return f
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key, fallback string) bool {
	value := getEnv(key, fallback)
	b, _ := strconv.ParseBool(value)
	return b
}

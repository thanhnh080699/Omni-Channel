package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env           string
	Port          string
	MongoURI      string
	MongoDatabase string
	JWTSecret     string
	JWTTTL        time.Duration
	AdminEmail    string
	AdminPassword string
	CDNBaseURL    string
	CDNAPIKey     string
}

func Load() Config {
	_ = godotenv.Load("backend/.env")
	_ = godotenv.Load(".env")

	ttlHours, err := strconv.Atoi(getenv("JWT_TTL_HOURS", "24"))
	if err != nil || ttlHours <= 0 {
		ttlHours = 24
	}

	return Config{
		Env:           getenv("APP_ENV", "local"),
		Port:          getenv("API_PORT", "8080"),
		MongoURI:      getenv("MONGO_URI", "mongodb://root:root@localhost:27017/omni_channel?authSource=admin"),
		MongoDatabase: getenv("MONGO_DATABASE", "omni_channel"),
		JWTSecret:     getenv("JWT_SECRET", "local-dev-change-me"),
		JWTTTL:        time.Duration(ttlHours) * time.Hour,
		AdminEmail:    getenv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword: getenv("ADMIN_PASSWORD", "admin123456"),
		CDNBaseURL:    getenv("CDN_BASE_URL", "http://localhost:8081"),
		CDNAPIKey:     getenv("CDN_API_KEY", ""),
	}
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

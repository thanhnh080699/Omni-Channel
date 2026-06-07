package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                        string
	Port                       string
	MongoURI                   string
	MongoDatabase              string
	JWTSecret                  string
	JWTTTL                     time.Duration
	AdminEmail                 string
	AdminPassword              string
	CDNBaseURL                 string
	CDNAPIKey                  string
	RabbitMQURL                string
	RedisAddr                  string
	RedisPassword              string
	RedisDB                    int
	QueuePartitions            int
	IdempotencyTTL             time.Duration
	OutboundTTL                time.Duration
	WhatsAppAdapterURL         string
	WhatsAppAdapterAutostart   bool
	WhatsAppAdapterAutoInstall bool
	WhatsAppAdapterDir         string
	WhatsAppAdapterCommand     string
	WhatsAppAdapterArgs        []string
	WebhookSharedSecret        string
}

func Load() Config {
	_ = godotenv.Load("backend/.env")
	_ = godotenv.Load(".env")

	ttlHours, err := strconv.Atoi(getenv("JWT_TTL_HOURS", "24"))
	if err != nil || ttlHours <= 0 {
		ttlHours = 24
	}
	redisDB, err := strconv.Atoi(getenv("REDIS_DB", "0"))
	if err != nil || redisDB < 0 {
		redisDB = 0
	}
	partitions, err := strconv.Atoi(getenv("QUEUE_PARTITIONS", "8"))
	if err != nil || partitions <= 0 {
		partitions = 8
	}
	idempotencyHours, err := strconv.Atoi(getenv("IDEMPOTENCY_TTL_HOURS", "72"))
	if err != nil || idempotencyHours <= 0 {
		idempotencyHours = 72
	}
	outboundTTLMinutes, err := strconv.Atoi(getenv("OUTBOUND_TTL_MINUTES", "60"))
	if err != nil || outboundTTLMinutes <= 0 {
		outboundTTLMinutes = 60
	}
	adapterArgs := splitCSV(getenv("WHATSAPP_ADAPTER_ARGS", "run,dev"))

	return Config{
		Env:                        getenv("APP_ENV", "local"),
		Port:                       getenv("API_PORT", "8080"),
		MongoURI:                   getenv("MONGO_URI", "mongodb://root:root@localhost:27017/omni_channel?authSource=admin"),
		MongoDatabase:              getenv("MONGO_DATABASE", "omni_channel"),
		JWTSecret:                  getenv("JWT_SECRET", "local-dev-change-me"),
		JWTTTL:                     time.Duration(ttlHours) * time.Hour,
		AdminEmail:                 getenv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:              getenv("ADMIN_PASSWORD", "admin123456"),
		CDNBaseURL:                 getenv("CDN_BASE_URL", "http://localhost:8081"),
		CDNAPIKey:                  getenv("CDN_API_KEY", ""),
		RabbitMQURL:                getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RedisAddr:                  getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:              getenv("REDIS_PASSWORD", "redis"),
		RedisDB:                    redisDB,
		QueuePartitions:            partitions,
		IdempotencyTTL:             time.Duration(idempotencyHours) * time.Hour,
		OutboundTTL:                time.Duration(outboundTTLMinutes) * time.Minute,
		WhatsAppAdapterURL:         getenv("WHATSAPP_ADAPTER_URL", "http://localhost:19090"),
		WhatsAppAdapterAutostart:   getenv("WHATSAPP_ADAPTER_AUTOSTART", "true") == "true",
		WhatsAppAdapterAutoInstall: getenv("WHATSAPP_ADAPTER_AUTO_INSTALL", "true") == "true",
		WhatsAppAdapterDir:         getenv("WHATSAPP_ADAPTER_DIR", "./whatsapp-adapter"),
		WhatsAppAdapterCommand:     getenv("WHATSAPP_ADAPTER_COMMAND", "npm"),
		WhatsAppAdapterArgs:        adapterArgs,
		WebhookSharedSecret:        getenv("WEBHOOK_SHARED_SECRET", ""),
	}
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

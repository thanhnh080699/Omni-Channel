package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"omni-channel/backend/internal/adapterprocess"
	"omni-channel/backend/internal/auth"
	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/handlers"
	"omni-channel/backend/internal/middleware"
	"omni-channel/backend/internal/queue"
	"omni-channel/backend/internal/store"
	"omni-channel/backend/internal/workers"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	cfg := config.Load()

	// If no command is provided, default to running the API server.
	command := "serve"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch command {
	case "serve", "dev":
		runAPIServer(cfg)
	case "worker":
		runWorkers(ctx, cfg)

	case "build":
		log.Println("Building API server...")
		cmd := exec.Command("go", "build", "-o", "bin/api", "main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Fatalf("Error building API: %v", err)
		}
		log.Println("Build completed successfully. Output path: bin/api")

	case "migrate":
		db, err := database.Connect(ctx, cfg)
		if err != nil {
			log.Fatalf("Error connecting to MongoDB: %v", err)
		}
		defer func() {
			if err := db.Client.Disconnect(ctx); err != nil {
				log.Printf("Error disconnecting from MongoDB: %v", err)
			}
		}()

		log.Println("Running database migrations (indexes)...")
		if err := database.EnsureIndexes(ctx, db); err != nil {
			log.Fatalf("Error ensuring indexes: %v", err)
		}
		log.Println("Database migrations (indexes) completed successfully.")

	case "db:seed":
		db, err := database.Connect(ctx, cfg)
		if err != nil {
			log.Fatalf("Error connecting to MongoDB: %v", err)
		}
		defer func() {
			if err := db.Client.Disconnect(ctx); err != nil {
				log.Printf("Error disconnecting from MongoDB: %v", err)
			}
		}()

		log.Println("Running database seeders...")
		if err := database.SeedDefaults(ctx, db, cfg); err != nil {
			log.Fatalf("Error seeding database: %v", err)
		}
		log.Println("Database seeders completed successfully.")

	case "migrate:fresh":
		db, err := database.Connect(ctx, cfg)
		if err != nil {
			log.Fatalf("Error connecting to MongoDB: %v", err)
		}
		defer func() {
			if err := db.Client.Disconnect(ctx); err != nil {
				log.Printf("Error disconnecting from MongoDB: %v", err)
			}
		}()

		log.Printf("Dropping database: %s...\n", cfg.MongoDatabase)
		if err := db.DB.Drop(ctx); err != nil {
			log.Fatalf("Error dropping database: %v", err)
		}
		log.Println("Database dropped successfully.")

		log.Println("Running database migrations (indexes)...")
		if err := database.EnsureIndexes(ctx, db); err != nil {
			log.Fatalf("Error ensuring indexes: %v", err)
		}
		log.Println("Database migrations (indexes) completed successfully.")

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func runAPIServer(cfg config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := db.Client.Disconnect(shutdownCtx); err != nil {
			log.Printf("disconnect mongo: %v", err)
		}
	}()

	if err := database.EnsureIndexes(ctx, db); err != nil {
		log.Fatalf("ensure indexes: %v", err)
	}
	if err := database.SeedDefaults(ctx, db, cfg); err != nil {
		log.Fatalf("seed defaults: %v", err)
	}

	adapterProcess, err := adapterprocess.StartWhatsAppAdapter(context.Background(), cfg)
	if err != nil {
		log.Fatalf("start whatsapp adapter: %v", err)
	}
	defer adapterProcess.Stop()
	bootstrapWhatsAppAutoConnect(context.Background(), cfg, db)

	tokenService := auth.NewTokenService(cfg.JWTSecret, cfg.JWTTTL)
	publisher, err := queue.NewRabbitPublisher(ctx, cfg.RabbitMQURL, cfg.QueuePartitions)
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Printf("close rabbitmq: %v", err)
		}
	}()

	stopWorkers := startEmbeddedWorkers(cfg, db)
	defer stopWorkers()

	router := handlers.NewRouter(cfg, db, tokenService, publisher)
	router.Use(middleware.RequestID())

	log.Printf("omni-channel api listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("run api: %v", err)
	}
}

func startEmbeddedWorkers(cfg config.Config, db *database.Mongo) func() {
	ctx, cancel := context.WithCancel(context.Background())
	publisher, err := queue.NewRabbitPublisher(ctx, cfg.RabbitMQURL, cfg.QueuePartitions)
	if err != nil {
		cancel()
		log.Fatalf("connect embedded worker rabbitmq: %v", err)
	}
	redisClient := store.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	runner := workers.NewRunner(cfg, db, publisher, store.NewIdempotencyGate(redisClient, cfg.IdempotencyTTL))

	go func() {
		log.Printf("embedded queue workers starting with %d conversation partitions", cfg.QueuePartitions)
		if err := runner.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("embedded queue workers stopped: %v", err)
		}
	}()

	return func() {
		cancel()
		if err := redisClient.Close(); err != nil {
			log.Printf("close embedded worker redis: %v", err)
		}
		if err := publisher.Close(); err != nil {
			log.Printf("close embedded worker rabbitmq: %v", err)
		}
	}
}

func bootstrapWhatsAppAutoConnect(ctx context.Context, cfg config.Config, db *database.Mongo) {
	go func() {
		bootCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		var whatsappChannel struct {
			ID string `bson:"_id"`
		}
		if err := db.C("channels").FindOne(bootCtx, bson.M{"code": "whatsapp", "status": "enabled"}).Decode(&whatsappChannel); err != nil {
			log.Printf("whatsapp auto-connect skipped: channel not available: %v", err)
			return
		}

		cursor, err := db.C("channel_accounts").Find(bootCtx, bson.M{
			"channel_id": whatsappChannel.ID,
			"enabled":    true,
		})
		if err != nil {
			log.Printf("whatsapp auto-connect list accounts: %v", err)
			return
		}
		defer cursor.Close(bootCtx)

		var accounts []struct {
			ID       string                 `bson:"_id"`
			Name     string                 `bson:"name"`
			Metadata map[string]interface{} `bson:"metadata"`
		}
		if err := cursor.All(bootCtx, &accounts); err != nil {
			log.Printf("whatsapp auto-connect decode accounts: %v", err)
			return
		}

		for _, account := range accounts {
			if metadataBool(account.Metadata, "autoConnect", true) {
				go connectWhatsAppAccount(ctx, cfg, db, account.ID, account.Name)
			}
		}
	}()
}

func connectWhatsAppAccount(ctx context.Context, cfg config.Config, db *database.Mongo, accountID string, accountName string) {
	connectCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	now := time.Now().UTC()
	_, _ = db.C("channel_accounts").UpdateByID(connectCtx, accountID, bson.M{"$set": bson.M{
		"session_status": "connecting",
		"updated_at":     now,
	}})

	req, err := http.NewRequestWithContext(connectCtx, http.MethodPost, strings.TrimRight(cfg.WhatsAppAdapterURL, "/")+"/connect/"+accountID, nil)
	if err != nil {
		log.Printf("whatsapp auto-connect %s: build request: %v", accountID, err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("whatsapp auto-connect %s: adapter unavailable: %v", accountID, err)
		_, _ = db.C("channel_accounts").UpdateByID(context.Background(), accountID, bson.M{"$set": bson.M{
			"session_status": "error",
			"last_error":     "whatsapp adapter unavailable during API startup",
			"updated_at":     time.Now().UTC(),
		}})
		return
	}
	defer resp.Body.Close()

	var payload struct {
		Status    string `json:"status"`
		LastError string `json:"lastError"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&payload)
	if payload.Status == "" {
		payload.Status = "connecting"
	}

	set := bson.M{
		"session_status": payload.Status,
		"updated_at":     time.Now().UTC(),
	}
	if resp.StatusCode >= 300 {
		set["session_status"] = "error"
		set["last_error"] = fmt.Sprintf("adapter returned %s during API startup", resp.Status)
	} else if payload.LastError != "" {
		set["last_error"] = payload.LastError
	} else {
		set["last_error"] = ""
	}
	_, _ = db.C("channel_accounts").UpdateByID(context.Background(), accountID, bson.M{"$set": set})
	log.Printf("whatsapp auto-connect %s (%s): %s", accountID, accountName, set["session_status"])
}

func metadataBool(metadata map[string]interface{}, key string, fallback bool) bool {
	if metadata == nil {
		return fallback
	}
	value, ok := metadata[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1"
	default:
		return fallback
	}
}

func runWorkers(ctx context.Context, cfg config.Config) {
	db, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("connect mongo: %v", err)
	}
	defer func() {
		if err := db.Client.Disconnect(context.Background()); err != nil {
			log.Printf("disconnect mongo: %v", err)
		}
	}()
	publisher, err := queue.NewRabbitPublisher(ctx, cfg.RabbitMQURL, cfg.QueuePartitions)
	if err != nil {
		log.Fatalf("connect rabbitmq: %v", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Printf("close rabbitmq: %v", err)
		}
	}()
	redisClient := store.NewRedisClient(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("close redis: %v", err)
		}
	}()

	runner := workers.NewRunner(cfg, db, publisher, store.NewIdempotencyGate(redisClient, cfg.IdempotencyTTL))
	if err := runner.Run(context.Background()); err != nil {
		log.Fatalf("run workers: %v", err)
	}
}

func printUsage() {
	fmt.Println("Artisan Database & API Runner Tool")
	fmt.Println("Usage:")
	fmt.Println("  go run main.go [command]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  serve / dev     Start the API server, adapters, and queue workers (default)")
	fmt.Println("  build           Build the API server executable")
	fmt.Println("  migrate         Create/ensure all MongoDB indexes")
	fmt.Println("  db:seed         Seed default permissions, roles, channels, and admin user")
	fmt.Println("  migrate:fresh   Drop the database and re-run migrations (create indexes)")
	fmt.Println("  worker          Start only RabbitMQ workers (advanced/debug)")
	fmt.Println("  help            Show this usage information")
}

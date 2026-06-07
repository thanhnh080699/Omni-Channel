package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	router := handlers.NewRouter(cfg, db, tokenService, publisher)
	router.Use(middleware.RequestID())

	log.Printf("omni-channel api listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("run api: %v", err)
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
	fmt.Println("  serve / dev     Start the API server (default)")
	fmt.Println("  build           Build the API server executable")
	fmt.Println("  migrate         Create/ensure all MongoDB indexes")
	fmt.Println("  db:seed         Seed default permissions, roles, channels, and admin user")
	fmt.Println("  migrate:fresh   Drop the database and re-run migrations (create indexes)")
	fmt.Println("  worker          Start RabbitMQ dispatcher, conversation, outbound, and resync workers")
	fmt.Println("  help            Show this usage information")
}

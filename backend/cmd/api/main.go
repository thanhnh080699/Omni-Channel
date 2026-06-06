package main

import (
	"context"
	"log"
	"time"

	"omni-channel/backend/internal/auth"
	"omni-channel/backend/internal/config"
	"omni-channel/backend/internal/database"
	"omni-channel/backend/internal/handlers"
	"omni-channel/backend/internal/middleware"
)

func main() {
	cfg := config.Load()

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

	tokenService := auth.NewTokenService(cfg.JWTSecret, cfg.JWTTTL)
	router := handlers.NewRouter(cfg, db, tokenService)
	router.Use(middleware.RequestID())

	log.Printf("omni-channel api listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("run api: %v", err)
	}
}

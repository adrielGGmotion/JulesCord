package main

import (
	"log/slog"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"julescord/internal/api"
	"julescord/internal/bot"
	"julescord/internal/config"
	"julescord/internal/db"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	// Initialize default JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 1. Load config
	slog.Info("Starting JulesCord...")
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Initialize the Database & Migrations
	if cfg.DatabaseURL != "" {
		err = db.RunMigrations(cfg.DatabaseURL)
		if err != nil {
			slog.Error("Failed to run migrations", "error", err)
			os.Exit(1)
		}
	}

	var database *db.DB
	if cfg.DatabaseURL != "" {
		database, err = db.New(cfg.DatabaseURL)
		if err != nil {
			slog.Error("Failed to initialize database connection", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Warn("DATABASE_URL not set. Running without database.")
	}

	// 3. Initialize the Discord Bot
	discordBot, err := bot.New(cfg, database)
	if err != nil {
		slog.Error("Failed to initialize Discord bot", "error", err)
		os.Exit(1)
	}

	// 4. Initialize the API Server
	apiServer := api.New(cfg, database, discordBot.Session)

	// 5. Start Bot concurrently
	go func() {
		if err := discordBot.Start(); err != nil {
			slog.Error("Discord bot failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// 6. Start API concurrently
	go func() {
		if err := apiServer.Start(); err != nil {
			slog.Error("API server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// 7. Wait for termination signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("JulesCord is now running. Press Ctrl+C to exit.")

	<-stop

	slog.Info("Received termination signal. Shutting down...")

	// 8. Gracefully shutdown everything
	if err := discordBot.Stop(); err != nil {
		slog.Error("Error stopping Discord bot", "error", err)
	}

	if err := apiServer.Stop(); err != nil {
		slog.Error("Error stopping API server", "error", err)
	}

	if database != nil {
		database.Close()
	}

	slog.Info("Shutdown complete. Exiting.")
}

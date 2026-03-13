package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"julescord/internal/api"
	"julescord/internal/bot"
	"julescord/internal/config"
	"julescord/internal/db"
)

func main() {
	// 1. Load config
	log.Println("Starting JulesCord...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize the Database & Migrations
	if cfg.DatabaseURL != "" {
		err = db.RunMigrations(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
	}

	var database *db.DB
	if cfg.DatabaseURL != "" {
		database, err = db.New(cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize database connection: %v", err)
		}
	} else {
		log.Println("Warning: DATABASE_URL not set. Running without database.")
	}

	// 3. Initialize the Discord Bot
	discordBot, err := bot.New(cfg, database)
	if err != nil {
		log.Fatalf("Failed to initialize Discord bot: %v", err)
	}

	// 4. Initialize the API Server
	apiServer := api.New(cfg)

	// 5. Start Bot concurrently
	go func() {
		if err := discordBot.Start(); err != nil {
			log.Fatalf("Discord bot failed to start: %v", err)
		}
	}()

	// 6. Start API concurrently
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("API server failed to start: %v", err)
		}
	}()

	// 7. Wait for termination signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Println("JulesCord is now running. Press Ctrl+C to exit.")

	<-stop

	log.Println("\nReceived termination signal. Shutting down...")

	// 8. Gracefully shutdown everything
	if err := discordBot.Stop(); err != nil {
		log.Printf("Error stopping Discord bot: %v", err)
	}

	if err := apiServer.Stop(); err != nil {
		log.Printf("Error stopping API server: %v", err)
	}

	if database != nil {
		database.Close()
	}

	log.Println("Shutdown complete. Exiting.")
}

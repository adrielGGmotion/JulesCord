package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the configuration values for the application.
type Config struct {
	DiscordToken    string
	DiscordClientID string
	DatabaseURL     string
	APIPort         string
}

// Load reads the configuration from environment variables.
// It optionally loads a .env file if it exists.
func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		// Log that .env is not found, but we don't return an error because
		// environment variables might be set via Docker, system env, etc.
		log.Println("No .env file found or error reading it, falling back to environment variables")
	}

	config := &Config{
		DiscordToken:    os.Getenv("DISCORD_TOKEN"),
		DiscordClientID: os.Getenv("DISCORD_CLIENT_ID"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		APIPort:         os.Getenv("API_PORT"),
	}

	if config.APIPort == "" {
		config.APIPort = "8080" // Default port
	}

	return config, nil
}

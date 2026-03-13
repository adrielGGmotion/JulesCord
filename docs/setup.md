# JulesCord Setup Guide

This document outlines how to set up and run the JulesCord bot and its associated web dashboard locally or via Docker.

## Prerequisites

- **Go** (1.21+ recommended)
- **Node.js** (18+ recommended) and `npm`
- **PostgreSQL** database (if running locally without Docker)
- **Docker** and **docker-compose** (optional, for containerized setup)
- A **Discord Bot Token** and **Client ID** (create one at the [Discord Developer Portal](https://discord.com/developers/applications))

## Environment Variables

The application is configured using environment variables. You can create a `.env` file in the root directory based on `.env.example`.

| Variable | Description |
|---|---|
| `BOT_TOKEN` | Your Discord bot token. **Note:** Must be strictly named `BOT_TOKEN`, not `DISCORD_TOKEN`. |
| `DISCORD_CLIENT_ID` | Your Discord application client ID (used for registering slash commands). |
| `DATABASE_URL` | PostgreSQL connection string (e.g., `postgres://user:password@localhost:5432/julescord?sslmode=disable`). |
| `API_PORT` | Port for the Gin HTTP server. Defaults to `8080`. |

## Running Locally

To run the full stack locally for development:

### 1. Database Setup

Ensure you have a PostgreSQL instance running. The backend uses `golang-migrate` to automatically apply migrations from the `migrations/` directory on startup, so you only need to create an empty database and set the `DATABASE_URL` accordingly.

### 2. Backend (Go Bot and API)

1. Navigate to the root of the repository.
2. Install Go dependencies:
   ```bash
   go mod tidy
   ```
3. Run the bot and API:
   ```bash
   go run ./cmd/bot/
   ```
   *The bot and Gin API server will start concurrently.*

### 3. Frontend (React Dashboard)

1. Navigate to the `web/` directory:
   ```bash
   cd web
   ```
2. Install Node.js dependencies:
   ```bash
   npm install
   ```
3. Start the Vite development server:
   ```bash
   npm run dev
   ```
   *The frontend dashboard will be available at `http://localhost:5173`.*

## Running via Docker

For a production-like environment or simpler setup, you can use Docker Compose, which will spin up the Go backend, PostgreSQL database, and build the React frontend.

1. Ensure your `.env` file is properly configured.
2. Build and start the containers:
   ```bash
   docker-compose up -build -d
   ```
3. The bot will connect to Discord, the Gin API will be exposed on port `8080`, and the PostgreSQL database will be available on port `5432`.

## Troubleshooting

- **Slash Commands Not Registering:** Ensure `DISCORD_CLIENT_ID` is correctly set. Registration happens via Discord's REST API on bot startup.
- **Address Already in Use:** If restarting the local Go backend, ensure any previous instances are killed (e.g., `kill $(lsof -t -i :8080)`).
- **Database Connection Issues:** Check that your PostgreSQL service is running and `DATABASE_URL` is correctly formatted. If the DB fails, the bot will gracefully skip DB initialization but many features will be disabled.

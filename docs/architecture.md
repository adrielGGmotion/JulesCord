# JulesCord Architecture

The architecture of JulesCord is separated into distinct backend and frontend layers, ensuring scalability, ease of development, and maintainability.

## Backend Architecture

The backend is built in **Go (1.21+)** and serves two primary roles simultaneously: acting as the Discord bot and exposing a RESTful HTTP API.

### 1. Concurrency Model
The main entry point (`cmd/bot/main.go`) initializes and runs both the Discord bot and the Gin HTTP API server concurrently as separate goroutines. A signal listener intercepts interrupts (SIGINT/SIGTERM) to trigger graceful shutdown processes.

### 2. Discord Bot Engine
- Uses `github.com/bwmarrin/discordgo` to interface with the Discord API.
- All slash commands are modularized within `internal/bot/commands/` and registered centrally via `internal/bot/commands/registry.go`. Registration occurs automatically with Discord on bot startup.
- Background goroutines manage persistent tasks, such as rotating bot status text and polling for scheduled announcements.
- Event handlers hook into lifecycle events like `messageCreate` and `guildMemberAdd` directly in `internal/bot/bot.go`.

### 3. API & WebSockets
- Uses the `gin-gonic/gin` framework located at `internal/api/server.go`.
- Exposes standard REST endpoints (e.g., `/api/stats`, `/api/users`, `/api/guilds`) for the frontend dashboard to fetch data.
- Exposes a WebSocket route at `/ws` using `gorilla/websocket`. The backend pushes real-time updates (like bot uptime and command counts) to connected frontend clients every 5 seconds.
- Exposes Prometheus metrics via the `/metrics` endpoint.

### 4. Database Layer
- Backed by **PostgreSQL**, connected using the connection pool library `github.com/jackc/pgx/v5`.
- Database operations are centralized in `internal/db/db.go`, abstracting queries into reusable methods (e.g., `GetStats()`, `GetUserEconomy()`).
- Automated schema management is performed via `golang-migrate` during application startup, applying SQL files from the `migrations/` directory.

## Frontend Architecture

The web dashboard provides a graphical interface for server administrators to view metrics and manage their bot settings.

- **Stack:** Built using **React 18** and **Vite**, styled entirely with **Tailwind CSS v3**.
- **Location:** The entire source code lives in the `web/` directory.
- **Routing:** Handled via `react-router-dom` to support features like Home, Guilds, Users, Metrics, and Moderation Logs pages.
- **API Integration:** All data fetching uses `axios`, defaulting to the `VITE_API_URL` environment variable or `http://localhost:8080`.
- **Charts:** Uses the `recharts` library for interactive components, such as the command usage popularity bar chart and latency histograms on the Metrics page.

## Metrics & Observability

- Uses standard library `log/slog` for structured JSON logging.
- Custom Prometheus metrics (Counters and Histograms) monitor API request counts, slash command latency, and database query durations. These are defined in `internal/metrics/metrics.go` and utilized across the application.

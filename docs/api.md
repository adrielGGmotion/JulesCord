# JulesCord API Documentation

The JulesCord backend exposes a Gin-based RESTful API for the web dashboard to consume, as well as a WebSocket connection for real-time data streaming.

## REST API Endpoints

All endpoints are hosted on the configured `API_PORT` (default: 8080).

### Core & Observability

- `GET /health`
  - Returns the health status of the API, checking both the database connection ping and the Discord WebSocket heartbeat latency.
  - Returns `200 OK` on success or `503 Service Unavailable` on failure.

- `GET /api/status`
  - Returns basic JSON status of the bot ("status: ok").

- `GET /metrics`
  - Exposes Prometheus metrics text output tracking command executions, command latency histograms, and database query durations.

- `GET /api/dashboard-metrics`
  - Exposes structured metrics for the dashboard's "Metrics" page.

### Dashboard Data Fetching

- `GET /api/stats`
  - Returns general bot statistics: total guilds, total users, total commands run, and uptime.

- `GET /api/stats/commands`
  - Returns the top 10 most used commands from the command log. Useful for plotting charts (e.g., via `recharts`).

- `GET /api/guilds`
  - Returns a list of all Discord guilds (servers) the bot has joined.

- `GET /api/users`
  - Returns a list of all registered users with their total XP and max level.

- `GET /api/mod-actions`
  - Returns a list of all moderation actions logged by the bot (e.g., warnings, kicks, bans, purges).

### Guild Configuration Management

- `GET /api/guilds/:id/config`
  - Fetches the current configuration settings for the specified guild `id`.

- `PATCH /api/guilds/:id/config`
  - Updates the configuration settings for the specified guild `id`. Used by the dashboard to toggle features or set specific channels and roles.

## WebSocket Streaming

- `GET /ws`
  - Upgrades an HTTP request to a continuous WebSocket connection.
  - The backend routinely pushes real-time bot statistics every 5 seconds to connected clients.
  - Used prominently by the Dashboard Home component to update the interface automatically without polling.

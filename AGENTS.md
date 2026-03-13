# JulesCord — Agent State File

> This file is your memory and your constitution. Read it completely at the start of every task. Update it completely before opening your PR. Never skip this step.

---

## Project Goal

Build **JulesCord** — a production-grade, complex Discord bot written in **Go**, with a **React + Tailwind web dashboard**, backed by **PostgreSQL**, with a REST API, real-time WebSocket features, and a clean command architecture. This is not a toy bot. It is a serious, modern, scalable application that improves every single iteration.

You are the sole developer. No human writes code. Every 15 minutes you receive this prompt, read this file, and implement the next set of tasks. The PR is auto-merged. You iterate forever.

---

## Stack (LOCKED — never change these)

| Layer | Technology |
|---|---|
| Bot runtime | Go (latest stable) |
| Discord library | `github.com/bwmarrin/discordgo` |
| Web framework | `github.com/gin-gonic/gin` |
| Database | PostgreSQL via `github.com/jackc/pgx/v5` |
| Migrations | `github.com/golang-migrate/migrate/v4` |
| Config | `.env` via `github.com/joho/godotenv` |
| Frontend | React 18 + Vite + Tailwind CSS v3 |
| Frontend HTTP | axios |
| Containerization | Docker + docker-compose |

---

## Repository Structure (build toward this)

```
JulesCord/
├── cmd/
│   └── bot/
│       └── main.go          # Entry point
├── internal/
│   ├── bot/                 # Discord bot logic
│   │   ├── bot.go
│   │   ├── commands/        # One file per command
│   │   └── events/          # Event handlers
│   ├── api/                 # REST API (Gin)
│   │   ├── server.go
│   │   └── handlers/
│   ├── db/                  # Database layer
│   │   ├── db.go
│   │   └── queries/
│   └── config/
│       └── config.go
├── migrations/              # SQL migration files
├── web/                     # React frontend
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   └── main.jsx
│   ├── package.json
│   └── vite.config.js
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── .env.example
├── .gitignore
└── AGENTS.md
```

---

## Current Status

**Phase: 1 — Foundation**
Starting from scratch in Go. All old Node.js files must be removed first.

---

## Completed Work
- Implemented Phase 2 Database Foundation: created `internal/db/db.go` with connection pooling via `pgxpool`, and configured automated migration runs using `golang-migrate`.
- Authored initial migration `migrations/001_init.sql` defining `guilds`, `users`, and `command_log` tables.
- Integrated database with the Discord bot in `internal/bot/bot.go` to handle `guildCreate` (upserting guilds) and interaction creates (upserting users and logging commands).
- Updated `cmd/bot/main.go` to run DB migrations at startup and establish a DB connection (with graceful fallback if no `DATABASE_URL` is set).
- Added `GetStats` method to `internal/db/db.go` to retrieve counts for guilds, users, and command logs.
- Added `/about` command to describe JulesCord and its autonomous build loop.
- Added `/stats` command to display guild count, user count, total commands run, and bot uptime.
- Added `/help` command to dynamically list all registered commands by iterating over the command registry.


- Removed old Node.js files (`index.js`, `deploy-commands.js`, `package.json`, `package-lock.json`, and `commands/` directory).
- Initialized Go module (`go.mod` and `go.sum`) with all required dependencies.
- Updated `.env.example` with `DISCORD_TOKEN`, `DISCORD_CLIENT_ID`, `DATABASE_URL`, `API_PORT`.
- Updated `.gitignore` to include Go binaries, `.env`, and `node_modules`.
- Created `internal/config/config.go` to load and parse environment variables.
- Created `internal/bot/bot.go` to connect to Discord and handle basic startup/shutdown.
- Created `internal/api/server.go` to serve Gin HTTP REST API endpoints like `/health`.
- Created `cmd/bot/main.go` to act as the primary entry point, running the Bot and API concurrently and shutting them down gracefully.

---

## Task Checklist

### Phase 1 — Foundation
- [x] Remove all old Node.js files (`index.js`, `deploy-commands.js`, `commands/`, `package.json`, `package-lock.json`)
- [x] `go.mod` and `go.sum` with all required dependencies
- [x] `.env.example` with `BOT_TOKEN`, `DISCORD_CLIENT_ID`, `DATABASE_URL`, `API_PORT`
- [x] `.gitignore` (Go binaries, node_modules, .env)
- [x] `cmd/bot/main.go` — entry point, loads config, starts bot + API server concurrently
- [x] `internal/config/config.go` — loads env vars into a typed Config struct
- [x] `internal/bot/bot.go` — connects to Discord, registers handlers, graceful shutdown on SIGINT
- [x] `internal/bot/commands/ping.go` — `/ping` slash command reporting latency
- [x] `internal/bot/commands/registry.go` — central command registration and dispatch system
- [x] `internal/api/server.go` — Gin HTTP server with `/health` and `/api/status` endpoints
- [x] `docker-compose.yml` — services: bot, postgres
- [x] `Dockerfile` — multi-stage Go build, final image is minimal

### Phase 2 — Database & Core Features
- [x] `internal/db/db.go` — PostgreSQL connection pool via pgx
- [x] `migrations/001_init.sql` — guilds, users, command_log tables
- [x] Guild auto-registration when bot joins a server
- [x] User tracking — upsert Discord users in DB on every interaction
- [x] `/about` command — describes itself and the autonomous build loop
- [x] `/stats` command — guild count, user count, uptime, commands run
- [x] `/help` command — dynamically lists all registered commands with descriptions

### Phase 3 — Moderation System
- [ ] `/warn @user reason` — stores warning in DB with timestamp and moderator ID
- [ ] `/warnings @user` — lists all warnings for a user
- [ ] `/kick @user reason` — kicks with audit log reason
- [ ] `/ban @user reason` — bans with audit log reason
- [ ] `/purge [count]` — bulk delete up to 100 messages
- [ ] Mod action log channel — all mod actions posted as embeds to configurable channel
- [ ] `migrations/002_moderation.sql` — warnings, mod_actions tables

### Phase 4 — Leveling & Economy
- [ ] XP award on message (cooldown: 1 min per user per channel)
- [ ] Level calculation from XP, level-up announcement in channel
- [ ] `/rank` — user's XP, level, server rank
- [ ] `/leaderboard` — top 10 users by XP as an embed
- [ ] `/daily` — daily coin reward, tracked per user per day
- [ ] `/coins` — check coin balance
- [ ] `migrations/003_economy.sql` — xp, levels, coins tables

### Phase 5 — Web Dashboard
- [ ] `web/` scaffold — Vite + React 18 + Tailwind CSS v3
- [ ] Dashboard home — bot status card, guild count, uptime, commands run
- [ ] Guilds page — table of all servers the bot is in
- [ ] Users page — searchable user list with XP and level
- [ ] Moderation log page — filterable table of all mod actions
- [ ] Real-time stats via WebSocket — Go backend pushes updates every 5 seconds
- [ ] Command usage bar chart (recharts)
- [ ] Dark theme, clean design — NOT generic Bootstrap

### Phase 6 — Per-Guild Config
- [ ] Guild config table in DB — log channel, mod roles, welcome channel, feature flags
- [ ] `/config` subcommands — admins can view and update guild settings
- [ ] Config API — `GET /api/guilds/:id/config` and `PATCH /api/guilds/:id/config`
- [ ] Welcome messages — customizable per guild on member join
- [ ] `migrations/004_config.sql`

### Phase 7 — Advanced Features
- [ ] Reaction roles system — add/remove roles via emoji reactions
- [ ] Auto-role on join — assign configurable role automatically
- [ ] Scheduled announcements — guild admins schedule messages at a time
- [ ] Bot status rotation — cycling presence messages about building itself
- [ ] `/changelog` — reads recent git commits from GitHub API and summarizes changes

### Phase 8 — Observability
- [ ] Structured JSON logging via `log/slog` (stdlib, Go 1.21+)
- [ ] Prometheus metrics at `/metrics` — command latency, errors, DB query time
- [ ] Dashboard metrics page — error rates, latency histogram, command popularity
- [ ] Improved `/health` — reports DB connectivity and Discord WS heartbeat status

---

## Architecture Rules — NEVER violate

1. **Never touch `.github/workflows/`** — automation handles itself
2. **Never hardcode secrets** — always environment variables
3. **Never push directly to main** — always open a PR
4. **Always update AGENTS.md** before opening a PR
5. **Never regress** — don't remove or break working features
6. **Write real, compiling Go code** — no pseudocode, no empty stubs
7. **One migration per phase** — never modify existing migration files, only add new ones
8. **PR title must start with `[Jules]`**
9. **Max 4 checklist items per iteration** — do them well, don't rush
10. **Frontend lives in `/web` only** — never mix frontend and backend

---

## Architecture Notes

- The Go binary runs two goroutines concurrently: Discord bot and Gin HTTP server
- React frontend is a separate Vite app in `/web`, served as static files in prod
- All Discord interactions go through the bot goroutine
- The REST API at `/api` is for the dashboard only
- WebSocket at `/ws` broadcasts real-time events to connected dashboard clients
- Use pgx connection pool — never open raw individual DB connections
- Slash commands are registered with Discord's REST API on every bot startup
- Use embeds for all Discord responses — never plain text

## Notes for Next Iteration

- Start Phase 1 completely: delete Node.js files, set up Go module, get bot connecting to Discord
- Get `/health` returning 200 from the API before anything else
- DB is optional in Phase 1 — skip if it blocks getting the bot online

---

## Environment Variables

| Variable | Description |
|---|---|
| `BOT_TOKEN` | Discord bot token — this is the exact secret name, use it everywhere |
| `DISCORD_CLIENT_ID` | Discord application client ID for slash command registration |
| `DATABASE_URL` | PostgreSQL connection string |
| `API_PORT` | Port for the Gin HTTP server (default 8080) |

**Important:** The token secret is named `BOT_TOKEN` — not `DISCORD_TOKEN`. Always use `BOT_TOKEN` in code and `.env.example`.

---

## Testing Requirements — MANDATORY

After every iteration that touches bot code, Jules MUST test before opening a PR.

### Steps

1. Build: `go build ./cmd/bot/` — must compile with zero errors
2. Run: `BOT_TOKEN=$BOT_TOKEN DISCORD_CLIENT_ID=$DISCORD_CLIENT_ID go run ./cmd/bot/` for 10 seconds
3. Verify the log contains a "Ready" / connected message
4. Verify slash commands were submitted to Discord REST API without error
5. Kill the process cleanly

The GitHub Actions runner has `BOT_TOKEN` and `DISCORD_CLIENT_ID` available as environment variables from secrets. Jules can reference them directly when running the test.

**If the bot fails to start or slash commands fail to register — fix it before opening the PR.**

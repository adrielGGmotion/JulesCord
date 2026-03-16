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

- Implemented Phase 3 Moderation System first 4 items: created `migrations/002_moderation.sql` with tables `warnings` and `mod_actions`.
- Added database methods `AddWarning`, `GetWarnings`, and `LogModAction` to `internal/db/db.go`.
- Added `/warn` command to issue warnings and log them in the database.
- Added `/warnings` command to list all warnings for a user.
- Added `/kick` command to kick users and log the moderation action.
- Added `/ban` command to ban users and log the moderation action.
- Added `/purge` command to bulk delete messages and log the moderation action.

- Implemented Phase 4 Leveling & Economy first 4 items: created `migrations/003_economy.sql` with table `user_economy` for xp, levels, and coins.
- Added database methods `GetUserEconomy`, `AddXP`, `SetLevel`, and `GetRank` to `internal/db/db.go`.
- Added `messageCreateHandler` to `internal/bot/bot.go` to award XP on message with a 1-minute cooldown per user per channel, and announce level-ups.
- Added `/rank` command to display a user's XP, level, and server rank.
- Added database methods `GetTopUsersByXP` and `ClaimDaily` to `internal/db/db.go`.
- Added `/leaderboard` command to display the top 10 users by XP in the server.
- Added `/daily` command to claim a 24-hour coin reward.
- Added `/coins` command to display a user's coin balance.

- Implemented Phase 5 Web Dashboard foundation: scaffolded React 18 frontend with Vite and Tailwind CSS.
- Added `/api/stats` and `/api/guilds` endpoints to Gin backend `internal/api/server.go` and implemented CORS.
- Added `GetGuilds` method to `internal/db/db.go`.
- Created Dashboard Home component at `web/src/pages/Home.jsx` showing bot status, total guilds, users, command run count and uptime.
- Created Dashboard Guilds component at `web/src/pages/Guilds.jsx` listing all servers the bot is in.
- Built Layout component in `web/src/components/Layout.jsx` featuring a clean, dark theme and dynamic sidebar navigation.
- Configured frontend routing via `react-router-dom` in `web/src/App.jsx`.
- Added database methods `GetUsersWithEconomy` and `GetModActions` to `internal/db/db.go`.
- Added `/api/users` and `/api/mod-actions` endpoints to Gin backend `internal/api/server.go`.
- Created Dashboard Users component at `web/src/pages/Users.jsx` listing all users with their total XP and max level.
- Created Dashboard Moderation component at `web/src/pages/Moderation.jsx` listing all moderation actions with a filter and search bar.
- Updated frontend routing in `web/src/App.jsx` to include the new Users and Moderation pages.

- Implemented remaining Phase 3 Moderation, Phase 5 Web Dashboard, and Phase 6 Guild Config items:
  - Created `migrations/004_config.sql` defining `guild_config` table.
  - Added `/config set-log-channel` command to `internal/bot/commands/config.go` allowing admins to set a log channel.
  - Added DB methods `SetGuildLogChannel` and `GetGuildLogChannel` to `internal/db/db.go`.
  - Updated all moderation commands (`warn`, `kick`, `ban`, `purge`) to post action embeds to the configured mod log channel.
  - Implemented WebSocket server in `internal/api/server.go` on `/ws` to stream real-time bot stats every 5 seconds.
  - Added DB method `GetCommandUsageStats` returning the top 10 most used commands from the command log.
  - Created `/api/stats/commands` endpoint in `internal/api/server.go`.
  - Updated React Dashboard Home component `web/src/pages/Home.jsx` to connect to WebSocket for real-time updates and added a `recharts` BarChart displaying command usage statistics.

- Removed old Node.js files (`index.js`, `deploy-commands.js`, `package.json`, `package-lock.json`, and `commands/` directory).
- Implemented Phase 6 Guild Config remaining features: added `/config view`, `/config set-welcome-channel`, and `/config set-mod-role` subcommands.
- Created Config API `GET /api/guilds/:id/config` and `PATCH /api/guilds/:id/config` endpoints in `internal/api/server.go`.
- Implemented `GuildMemberAdd` event handler in `internal/bot/bot.go` to send automated welcome messages to configured channels.
- Implemented Phase 7 Advanced Features first 4 items:
  - Created `migrations/005_advanced.sql` adding `auto_role_id` to `guild_config` and creating `reaction_roles` and `scheduled_announcements` tables.
  - Added `set-auto-role` subcommand to `/config` and updated `guildMemberAddHandler` in `internal/bot/bot.go` to assign the role.
  - Added a background goroutine `rotateStatus` in `internal/bot/bot.go` to cycle bot custom presence every 5 minutes.
  - Added `/reactionrole add` and `/reactionrole remove` commands in `internal/bot/commands/reactionrole.go`.
  - Added `messageReactionAddHandler` and `messageReactionRemoveHandler` in `internal/bot/bot.go` to assign/remove roles based on reaction emojis.
  - Added `/schedule add` command in `internal/bot/commands/schedule.go` to schedule future announcements.
  - Added a background goroutine `checkScheduledAnnouncements` in `internal/bot/bot.go` to dispatch pending messages every minute.
- Implemented remaining Phase 7 Advanced Features and Phase 8 Observability first 3 items:
  - Created `/changelog` slash command in `internal/bot/commands/changelog.go` to fetch recent GitHub commits.
  - Replaced standard logging with structured JSON logging via `log/slog` throughout the codebase.
  - Added Prometheus metrics track command executions, command latency, and DB query latency in `internal/metrics/metrics.go`.
  - Instrumented `internal/bot/commands/registry.go` and `internal/db/db.go` with Prometheus latency trackers.
  - Exposed Prometheus `/metrics` endpoint on the Gin API server.
  - Improved `/health` endpoint to check DB connection ping and Discord Heartbeat latency.
- Implemented Phase 8 Observability item "Dashboard metrics page":
  - Created `/api/dashboard-metrics` endpoint in `internal/api/server.go` to expose Prometheus metrics (command execution count, command latency, and DB query latency) in a JSON structure.
  - Added React Dashboard Metrics component at `web/src/pages/Metrics.jsx` utilizing `recharts` to display the command stats and DB query latency.
  - Updated frontend routing in `web/src/App.jsx` to include the new Metrics page.
- Initialized Go module (`go.mod` and `go.sum`) with all required dependencies.
- Updated `.env.example` with `DISCORD_TOKEN`, `DISCORD_CLIENT_ID`, `DATABASE_URL`, `API_PORT`.
- Updated `.gitignore` to include Go binaries, `.env`, and `node_modules`.
- Created `internal/config/config.go` to load and parse environment variables.
- Created `internal/bot/bot.go` to connect to Discord and handle basic startup/shutdown.
- Created `internal/api/server.go` to serve Gin HTTP REST API endpoints like `/health`.
- Created `cmd/bot/main.go` to act as the primary entry point, running the Bot and API concurrently and shutting them down gracefully.
- Implemented Phase 10 Ticketing System features: added migrations `007_tickets.sql` with table `tickets`.
- Added database operations `CreateTicket`, `GetTicketByChannel`, and `CloseTicket` in `internal/db/db.go`.
- Added `/ticket create` command in `internal/bot/commands/ticket.go` to create private ticket channels.
- Added `/ticket close` command to close tickets and delete the respective channels.
- Implemented Phase 11 Tags System features: added migrations `008_tags.sql` with table `tags`.
- Added database operations `CreateTag`, `GetTag`, `DeleteTag`, and `ListTags` in `internal/db/db.go`.
- Added `/tag` command structure in `internal/bot/commands/tag.go` with four subcommands (`create`, `list`, `delete`, `view`).
- Implemented Phase 12 Auto-Responder System features: added migrations `009_auto_responders.sql` with table `auto_responders`.
- Implemented Phase 13 Starboard System features: added migrations `010_starboard.sql` with tables `starboard_config` and `starboard_messages`.
- Added DB operations `SetStarboardConfig`, `GetStarboardConfig`, `GetStarboardMessage`, and `UpsertStarboardMessage` in `internal/db/db.go`.
- Added `/starboard setup` command in `internal/bot/commands/starboard.go` to configure the starboard channel and star threshold.
- Added starboard reaction handler in `internal/bot/bot.go` to track `⭐` reactions, counting them and posting/updating embeds on the configured starboard channel.
- Added database operations `AddAutoResponder`, `RemoveAutoResponder`, `ListAutoResponders`, and `ListAllAutoResponders` in `internal/db/db.go`.
- Added `/autoresponder` command in `internal/bot/commands/autoresponder.go` with subcommands `add`, `remove`, and `list`.
- Updated message handler in `internal/bot/bot.go` to use an in-memory cache to check incoming messages and reply if a trigger word matches without querying the database each time.
- Implemented Phase 14 Giveaways System features: added migrations `011_giveaways.sql` with tables `giveaways` and `giveaway_entrants`.
- Added DB operations `CreateGiveaway`, `GetActiveGiveaways`, `EndGiveaway`, `GetGiveawayByMessage`, `AddGiveawayEntrant`, and `GetGiveawayEntrants` in `internal/db/db.go`.
- Added `/giveaway create` and `/giveaway end` commands in `internal/bot/commands/giveaway.go`.
- Added message reaction handler in `internal/bot/bot.go` for the `🎉` emoji to allow users to enter giveaways.
- Added a background goroutine `checkGiveaways` in `internal/bot/bot.go` that picks winners for ended giveaways every minute and announces them.

- Implemented Phase 15 AFK System features: added migrations `012_afk.sql` with table `afk_users`.
- Added DB operations `SetAFK`, `RemoveAFK`, and `GetAFK` in `internal/db/db.go`.
- Added `/afk` command in `internal/bot/commands/afk.go` allowing users to set an AFK reason.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to remove AFK status when a user types and notify the channel when an AFK user is mentioned.

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
- [x] `/warn @user reason` — stores warning in DB with timestamp and moderator ID
- [x] `/warnings @user` — lists all warnings for a user
- [x] `/kick @user reason` — kicks with audit log reason
- [x] `/ban @user reason` — bans with audit log reason
- [x] `/purge [count]` — bulk delete up to 100 messages
- [x] Mod action log channel — all mod actions posted as embeds to configurable channel
- [x] `migrations/002_moderation.sql` — warnings, mod_actions tables

### Phase 4 — Leveling & Economy
- [x] XP award on message (cooldown: 1 min per user per channel)
- [x] Level calculation from XP, level-up announcement in channel
- [x] `/rank` — user's XP, level, server rank
- [x] `/leaderboard` — top 10 users by XP as an embed
- [x] `/daily` — daily coin reward, tracked per user per day
- [x] `/coins` — check coin balance
- [x] `migrations/003_economy.sql` — xp, levels, coins tables

### Phase 5 — Web Dashboard
- [x] `web/` scaffold — Vite + React 18 + Tailwind CSS v3
- [x] Dashboard home — bot status card, guild count, uptime, commands run
- [x] Guilds page — table of all servers the bot is in
- [x] Users page — searchable user list with XP and level
- [x] Moderation log page — filterable table of all mod actions
- [x] Real-time stats via WebSocket — Go backend pushes updates every 5 seconds
- [x] Command usage bar chart (recharts)
- [x] Dark theme, clean design — NOT generic Bootstrap

### Phase 6 — Per-Guild Config
- [x] Guild config table in DB — log channel, mod roles, welcome channel, feature flags
- [x] `/config` subcommands — admins can view and update guild settings
- [x] Config API — `GET /api/guilds/:id/config` and `PATCH /api/guilds/:id/config`
- [x] Welcome messages — customizable per guild on member join
- [x] `migrations/004_config.sql`

### Phase 7 — Advanced Features
- [x] Reaction roles system — add/remove roles via emoji reactions
- [x] Auto-role on join — assign configurable role automatically
- [x] Scheduled announcements — guild admins schedule messages at a time
- [x] Bot status rotation — cycling presence messages about building itself
- [x] `/changelog` — reads recent git commits from GitHub API and summarizes changes

### Phase 8 — Observability
- [x] Structured JSON logging via `log/slog` (stdlib, Go 1.21+)
- [x] Prometheus metrics at `/metrics` — command latency, errors, DB query time
- [x] Dashboard metrics page — error rates, latency histogram, command popularity
- [x] Improved `/health` — reports DB connectivity and Discord WS heartbeat status

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

### Phase 9 — Reminders System
- [x] `migrations/006_reminders.sql` — reminders table
- [x] DB operations — AddReminder, GetPendingReminders, GetDueReminders, DeleteReminder, MarkReminderSent
- [x] `/remind` subcommands — add, list, delete
- [x] Background goroutine for delivery

### Phase 10 — Ticketing System
- [x] `migrations/007_tickets.sql` — tickets table
- [x] DB operations — CreateTicket, CloseTicket, GetTicketByChannel
- [x] `/ticket create` — creates a new private text channel for the ticket and sends a welcome message
- [x] `/ticket close` — marks the ticket as closed in DB and deletes the ticket channel

### Phase 11 — Tags System
- [x] `migrations/008_tags.sql` — tags table
- [x] DB operations — CreateTag, GetTag, DeleteTag, ListTags
- [x] `/tag` subcommands — create, list, delete, view

### Phase 12 — Auto-Responder System
- [x] `migrations/009_auto_responders.sql` — auto_responders table
- [x] DB operations — AddAutoResponder, RemoveAutoResponder, ListAutoResponders
- [x] `/autoresponder` subcommands — add, remove, list
- [x] Message handler to check for triggers and respond automatically

### Phase 13 — Starboard System
- [x] `migrations/010_starboard.sql` — `starboard_config` and `starboard_messages` tables
- [x] DB operations — `SetStarboardConfig`, `GetStarboardConfig`, `GetStarboardMessage`, `UpsertStarboardMessage`
- [x] `/starboard setup` command — configures the starboard channel and threshold
- [x] Message reaction handler for ⭐ — posts/updates messages on the starboard

### Phase 14 — Giveaways System
- [x] `migrations/011_giveaways.sql` — `giveaways` and `giveaway_entrants` tables
- [x] DB operations — `CreateGiveaway`, `GetActiveGiveaways`, `EndGiveaway`, `AddGiveawayEntrant`, `GetGiveawayEntrants`
- [x] `/giveaway create` and `/giveaway end` commands
- [x] Message reaction handler for 🎉 to enter giveaways, and background goroutine to pick winners

- Implemented Phase 16 Sticky Messages System features: added migrations 013_sticky_messages.sql with table sticky_messages.
- Added DB operations SetSticky, RemoveSticky, GetSticky, and UpdateStickyMessageID in internal/db/db.go.
- Added /sticky command with set and remove subcommands in internal/bot/commands/sticky.go.
- Updated message handler in internal/bot/bot.go to maintain sticky messages at the bottom of channels.
- Implemented Phase 18 Suggestions System features: added migrations `015_suggestions.sql` with tables `suggestion_config` and `suggestions`.
- Added DB operations `SetSuggestionChannel`, `GetSuggestionChannel`, `CreateSuggestion`, `GetSuggestionByID`, and `UpdateSuggestionStatus` in `internal/db/db.go`.
- Added `/suggest` command with `setup`, `submit`, `accept`, and `reject` subcommands in `internal/bot/commands/suggest.go`.
- Updated `internal/bot/bot.go` to register the `suggest` command.

- Implemented Phase 17 Polls System features: added migrations `014_polls.sql` with table `polls`.
- Added DB operations `CreatePoll`, `GetPoll`, and `ClosePoll` in `internal/db/db.go`.
- Added `/poll create` and `/poll close` commands in `internal/bot/commands/poll.go`.
- Added logic to handle adding number reactions to poll options and dynamically counting reactions to display poll results when closing a poll.

- Implemented Phase 19 Server Logs System features: added migrations `016_server_logs.sql` with table `server_log_config`.
- Added DB operations `SetServerLogChannel` and `GetServerLogChannel` in `internal/db/db.go`.
- Added `/serverlog setup` command in `internal/bot/commands/serverlog.go`.
- Added `messageUpdateHandler` and `messageDeleteHandler` in `internal/bot/bot.go` to track and log edited and deleted messages.

- Implemented Phase 20 Auto-Moderation System features: added migrations `017_automod.sql` with tables `automod_config` and `automod_words`.
- Added DB operations `SetAutomodConfig`, `GetAutomodConfig`, `AddAutomodWord`, `RemoveAutomodWord`, `GetAutomodWords` in `internal/db/db.go`.
- Added `/automod` command with `setup`, `word add`, `word remove`, and `word list` subcommands in `internal/bot/commands/automod.go`.
- Added `checkAutomod` message handler check in `internal/bot/bot.go` to intercept and delete messages with links, invites, or bad words and send embedded logs to the configured log channel.

- Implemented Phase 21 Verification System features: added migrations `018_verification.sql` with table `verification_config`.
- Added DB operations `SetVerificationConfig` and `GetVerificationConfig` in `internal/db/db.go`.
- Added `/verify setup` command in `internal/bot/commands/verification.go` that posts an interactive verification panel.
- Added a component interaction handler in `internal/bot/bot.go` to listen for clicks on the `verify_button` and assign the designated role.

- Implemented Phase 22 User Notes System features: added migrations `019_notes.sql` with table `user_notes`.
- Added DB operations `AddNote`, `GetNotes`, and `RemoveNote` in `internal/db/db.go`.
- Added `/note` command with `add`, `list`, and `remove` subcommands in `internal/bot/commands/note.go`.

- Implemented Phase 24 Voice Logging System features: added migrations `021_voice_logs.sql` with table `voice_log_config`.
- Added DB operations `SetVoiceLogChannel` and `GetVoiceLogChannel` in `internal/db/db.go`.
- Added `/voicelog` command with `setup` subcommand in `internal/bot/commands/voicelog.go`.
- Added `voiceStateUpdateHandler` in `internal/bot/bot.go` to track and log voice joins, leaves, and moves and send embed logs to the configured channel.
- Implemented Phase 25 Reputation System features: added migrations `022_reputation.sql` with tables `reputation` and `reputation_log`.
- Added DB operations `GetReputation`, `AddReputation`, and `CanGiveReputation` in `internal/db/db.go`.
- Added `/rep` command with `give` and `check` subcommands in `internal/bot/commands/rep.go`.
- Updated `internal/bot/bot.go` to register the `rep` command.

- Implemented Phase 26 User Profiles System features: added migrations `023_profiles.sql` with table `user_profiles`.
- Added DB operations `SetProfileBio`, `SetProfileColor`, and `GetProfile` in `internal/db/db.go`.
- Added `/profile` command with `view`, `set-bio`, and `set-color` subcommands in `internal/bot/commands/profile.go`.
- Updated `internal/bot/bot.go` to register the `profile` command.
- Implemented Phase 23 Level Roles System features: added migrations `020_level_roles.sql` with table `level_roles`.
- Added DB operations `SetLevelRole`, `RemoveLevelRole`, `GetLevelRoles`, and `GetLevelRole` in `internal/db/db.go`.
- Added `/levelrole` command with `add`, `list`, and `remove` subcommands in `internal/bot/commands/levelrole.go`.
- Updated message handler in `internal/bot/bot.go` to assign level roles when users reach specific levels via the XP system.
- Implemented Phase 27 Economy Shop System features: added migrations `024_shop.sql` with tables `shop_items` and `user_inventory`.
- Added DB operations `AddShopItem`, `RemoveShopItem`, `GetShopItems`, `GetShopItem`, `BuyItem`, and `GetUserInventory` in `internal/db/db.go`.
- Added `/shop` command with `add`, `remove`, `list`, and `buy` subcommands in `internal/bot/commands/shop.go`.
- Added `/inventory` command in `internal/bot/commands/inventory.go` to view purchased items.
- Updated `internal/bot/bot.go` to register the `shop` and `inventory` commands.

### Phase 15 — AFK System
- [x] `migrations/012_afk.sql` — `afk_users` table
- [x] DB operations — `SetAFK`, `RemoveAFK`, `GetAFK`
- [x] `/afk` command
- [x] Message handler checks for mentions to notify channel and removes AFK status when an AFK user types

### Phase 16 — Sticky Messages System
- [x] `migrations/013_sticky_messages.sql` — `sticky_messages` table
- [x] DB operations — `SetSticky`, `RemoveSticky`, `GetSticky`, `UpdateStickyMessageID`
- [x] `/sticky` command with `set` and `remove` subcommands
- [x] Message handler to maintain the sticky message at the bottom of the channel

### Phase 17 — Polls System
- [x] `migrations/014_polls.sql` — `polls` table
- [x] DB operations — `CreatePoll`, `GetPoll`, `ClosePoll`
- [x] `/poll create` command — creates a poll with up to 10 options, posts embed, adds number reactions
- [x] `/poll close` command — closes a poll, tallies reactions, and displays the final results

### Phase 18 — Suggestions System
- [x] `migrations/015_suggestions.sql` — `suggestion_config` and `suggestions` tables
- [x] DB operations — `SetSuggestionChannel`, `GetSuggestionChannel`, `CreateSuggestion`, `GetSuggestionByID`, `UpdateSuggestionStatus`
- [x] `/suggest` command with `setup`, `submit`, `accept`, and `reject` subcommands

### Phase 19 — Server Logs System
- [x] `migrations/016_server_logs.sql` — `server_log_config` table
- [x] DB operations — `SetServerLogChannel`, `GetServerLogChannel`
- [x] `/serverlog` command with `setup` subcommand
- [x] Message handlers for tracking edited and deleted messages

### Phase 20 — Auto-Moderation System
- [x] `migrations/017_automod.sql` — `automod_config` and `automod_words` tables
- [x] DB operations — `SetAutomodConfig`, `GetAutomodConfig`, `AddAutomodWord`, `RemoveAutomodWord`, `GetAutomodWords`
- [x] `/automod` command with `setup`, `word add`, `word remove`, `word list` subcommands
- [x] Message handlers check for links, invites, and bad words, delete the message, and send an embed to the log channel

### Phase 21 — Verification System
- [x] `migrations/018_verification.sql` — `verification_config` table
- [x] DB operations — `SetVerificationConfig`, `GetVerificationConfig`
- [x] `/verify setup` command — creates an interactive verification panel with a button
- [x] Message component handler — assigns the verification role when the button is clicked

### Phase 22 — User Notes System
- [x] `migrations/019_notes.sql` — `user_notes` table
- [x] DB operations — `AddNote`, `GetNotes`, `RemoveNote`
- [x] `/note` command with `add`, `list`, and `remove` subcommands

### Phase 23 — Level Roles System
- [x] `migrations/020_level_roles.sql` — `level_roles` table
- [x] DB operations — `SetLevelRole`, `RemoveLevelRole`, `GetLevelRoles`, `GetLevelRole`
- [x] `/levelrole` command with `add`, `remove`, `list` subcommands
- [x] Assign role on level up in XP system

### Phase 24 — Voice Logging System
- [x] `migrations/021_voice_logs.sql` — `voice_log_config` table
- [x] DB operations — `SetVoiceLogChannel`, `GetVoiceLogChannel`
- [x] `/voicelog` command with `setup` subcommand
- [x] `voiceStateUpdateHandler` to track and log voice joins, leaves, and moves

### Phase 25 — Reputation System
- [x] `migrations/022_reputation.sql` — `reputation` and `reputation_log` tables
- [x] DB operations — `GetReputation`, `AddReputation`, `CanGiveReputation`
- [x] `/rep` command with `give` and `check` subcommands

### Phase 26 — User Profiles System
- [x] `migrations/023_profiles.sql` — `user_profiles` table
- [x] DB operations — `SetProfileBio`, `SetProfileColor`, `GetProfile`
- [x] `/profile` command with `view`, `set-bio`, and `set-color` subcommands
- [x] Profile embed displays bio, color, and integrates economy (XP, coins) and reputation stats

### Phase 27 — Economy Shop System
- [x] `migrations/024_shop.sql` — `shop_items` and `user_inventory` tables
- [x] DB operations — `AddShopItem`, `RemoveShopItem`, `GetShopItems`, `GetShopItem`, `BuyItem`, `GetUserInventory`
- [x] `/shop` command with `add`, `remove`, `list`, and `buy` subcommands
- [x] `/inventory` command to view purchased items

### Phase 28 — Birthday System
- [x] `migrations/025_birthdays.sql` — `birthday_config` and `birthdays` tables
- [x] DB operations — `SetBirthdayChannel`, `GetBirthdayChannel`, `SetBirthday`, `RemoveBirthday`, `GetBirthdays`, `GetDueBirthdays`, `MarkBirthdayAnnounced`
- [x] `/birthday` command with `setup`, `set`, `remove`, and `list` subcommands
- [x] Background goroutine for daily birthday announcements

### Phase 29 — Temporary Voice Channels
- [x] `migrations/026_temp_voice.sql` — `temp_voice_config` and `temp_voice_channels` tables
- [x] DB operations — `SetTempVoiceConfig`, `GetTempVoiceConfig`, `CreateTempVoiceChannel`, `GetTempVoiceChannel`, `DeleteTempVoiceChannel`
- [x] `/tempvoice` command with `setup` subcommand
- [x] `voiceStateUpdateHandler` to create/delete temporary voice channels

### Phase 30 — Ticket Panels
- [x] `migrations/027_ticket_panels.sql` — `ticket_panels` table
- [x] DB operations — `SetTicketPanel`, `GetTicketPanel`
- [x] `/ticket panel` command to create the panel with a button
- [x] Interaction handler for `ticket_panel_button` to automatically open a ticket

- Implemented Phase 28 Birthday System features: added migrations `025_birthdays.sql` with tables `birthday_config` and `birthdays`.
- Added DB operations `SetBirthdayChannel`, `GetBirthdayChannel`, `SetBirthday`, `RemoveBirthday`, `GetBirthdays`, `GetDueBirthdays`, and `MarkBirthdayAnnounced` in `internal/db/db.go`.
- Added `/birthday` command with `setup`, `set`, `remove`, and `list` subcommands in `internal/bot/commands/birthday.go`.
- Added background goroutine `checkBirthdays` in `internal/bot/bot.go` to announce birthdays daily.

- Implemented Phase 29 Temporary Voice Channels foundation: added migrations `026_temp_voice.sql` with tables `temp_voice_config` and `temp_voice_channels`.
- Added DB operations `SetTempVoiceConfig`, `GetTempVoiceConfig`, `CreateTempVoiceChannel`, `GetTempVoiceChannel`, and `DeleteTempVoiceChannel` in `internal/db/db.go`.
- Added `/tempvoice setup` command in `internal/bot/commands/tempvoice.go` to configure category and trigger channel.
- Added `voiceStateUpdateHandler` logic to dynamically create and delete temporary voice channels based on user state.
- Implemented Phase 30 Ticket Panels features: added migrations `027_ticket_panels.sql` with table `ticket_panels`.
- Added DB operations `SetTicketPanel` and `GetTicketPanel` in `internal/db/db.go`.
- Added `/ticket panel` command in `internal/bot/commands/ticket.go` to create interactive ticket creation buttons.
- Updated `interactionCreateHandler` in `internal/bot/bot.go` to handle button interactions and automatically create tickets.

- Implemented Phase 31 Marriage System features: added migrations `028_marriages.sql` with table `marriages`.
- Added DB operations `ProposeMarriage`, `AcceptMarriage`, `Divorce`, and `GetMarriage` in `internal/db/db.go`.
- Added `/marry` command with `propose`, `accept`, and `divorce` subcommands in `internal/bot/commands/marry.go`.
- Updated `/profile` command in `internal/bot/commands/profile.go` to display a user's marriage status in their profile embed.
- Implemented Phase 32 Counting Channel System features: added migrations `029_counting.sql` with table `counting_config`.
- Added DB operations `SetCountingChannel`, `GetCountingChannel`, `UpdateCountingNumber`, and `ResetCountingNumber` in `internal/db/db.go`.
- Added `/counting setup` command in `internal/bot/commands/counting.go` to configure the channel.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to validate and increment numbers in the configured counting channel.

- Implemented Phase 33 Trivia System features: added migrations `030_trivia.sql` with table `trivia_scores`.
- Added DB operations `AddTriviaScore`, `GetTriviaLeaderboard`, and `AddCoins` in `internal/db/db.go`.
- Added `/trivia` command with `start` and `leaderboard` subcommands in `internal/bot/commands/trivia.go` fetching questions from OpenTDB.
- Updated `interactionCreateHandler` in `internal/bot/bot.go` to process trivia answer buttons, award coins, and update the leaderboard.

- Implemented Phase 34 Custom Commands features: added migrations `031_custom_commands.sql` with table `custom_commands`.
- Added DB operations `AddCustomCommand`, `RemoveCustomCommand`, `ListCustomCommands`, and `GetCustomCommand` in `internal/db/db.go`.
- Added `/customcommand` command with `add`, `remove`, and `list` subcommands in `internal/bot/commands/customcommand.go`.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to parse messages for custom prefixes and respond with the configured custom command response.

- Implemented Phase 35 Snipe System features: added migrations `032_snipe.sql` with tables `snipes` and `edit_snipes`.
- Added DB operations `AddSnipe`, `GetSnipe`, `AddEditSnipe`, and `GetEditSnipe` in `internal/db/db.go`.
- Added `/snipe` and `/editsnipe` commands in `internal/bot/commands/snipe.go`.
- Updated `messageDeleteHandler` and `messageUpdateHandler` in `internal/bot/bot.go` to save deleted and edited messages to the database.

- Implemented Phase 36 Gambling System features: added migrations `033_gambling.sql` with table `gambling_stats`.
- Added DB operations `RemoveCoins`, `UpdateGamblingStats`, and `GetGamblingStats` in `internal/db/db.go`.
- Added `/gamble` command with `coinflip`, `slots`, and `stats` subcommands in `internal/bot/commands/gamble.go`.
- Updated `internal/bot/bot.go` to register the `gamble` command.

- Implemented Phase 37 Confessions System features: added migrations `034_confessions.sql` with table `confession_config`.
- Added DB operations `SetConfessionChannel` and `GetConfessionChannel` in `internal/db/db.go`.
- Added `/confession` command with `setup` subcommand in `internal/bot/commands/confession.go` to set the channel.
- Added `/confess` command in `internal/bot/commands/confess.go` to anonymously post to the configured channel.
- Updated `internal/bot/bot.go` to register the `confession` and `confess` commands.

- Implemented Phase 38 To-Do List System features: added migrations `035_todos.sql` with table `todos`.
- Added DB operations `AddTodo`, `GetTodos`, `CompleteTodo`, and `RemoveTodo` in `internal/db/db.go`.
- Added `/todo` command with `add`, `list`, `complete`, and `remove` subcommands in `internal/bot/commands/todo.go`.
- Updated `internal/bot/bot.go` to register the `todo` command.

- Implemented Phase 39 Role Menu System features: added migrations `036_role_menus.sql` with tables `role_menus` and `role_menu_options`.
- Added DB operations `CreateRoleMenu`, `AddRoleMenuOption`, and `GetRoleMenu` in `internal/db/db.go`.
- Added `/rolemenu` command with `setup` and `add_role` subcommands in `internal/bot/commands/rolemenu.go`.
- Updated `internal/bot/bot.go` to register the `rolemenu` command and handle the `role_menu_select` dropdown interaction to assign/remove roles dynamically.

### Phase 31 — Marriage System
- [x] `migrations/028_marriages.sql` — `marriages` table
- [x] DB operations — `ProposeMarriage`, `AcceptMarriage`, `Divorce`, `GetMarriage`
- [x] `/marry` command with `propose`, `accept`, `divorce` subcommands
- [x] Display marriage status in `/profile` embed

### Phase 32 — Counting Channel System
- [x] `migrations/029_counting.sql` — `counting_config` table
- [x] DB operations — `SetCountingChannel`, `GetCountingChannel`, `UpdateCountingNumber`, `ResetCountingNumber`
- [x] `/counting setup` command to configure the channel
- [x] Message handler to validate and increment numbers

### Phase 33 — Trivia System
- [x] `migrations/030_trivia.sql` — `trivia_scores` table
- [x] DB operations — `AddTriviaScore`, `GetTriviaLeaderboard`
- [x] `/trivia start` command to fetch a random question from OpenTDB and display it with interactive buttons
- [x] Message component handler to process trivia answers, award coins to winners, and update the leaderboard

### Phase 34 — Custom Commands
- [x] `migrations/031_custom_commands.sql` — `custom_commands` table
- [x] DB operations — `AddCustomCommand`, `RemoveCustomCommand`, `ListCustomCommands`, `GetCustomCommand`
- [x] `/customcommand` command with `add`, `remove`, and `list` subcommands
- [x] Message handler to listen for custom commands (e.g., matching exact prefix-less trigger or slash command emulation) and respond

### Phase 35 — Snipe System
- [x] `migrations/032_snipe.sql` — `snipes` and `edit_snipes` tables
- [x] DB operations — `AddSnipe`, `GetSnipe`, `AddEditSnipe`, `GetEditSnipe`
- [x] `/snipe` command — fetches the last deleted message in the channel
- [x] `/editsnipe` command — fetches the last edited message in the channel

### Phase 36 — Gambling System
- [x] `migrations/033_gambling.sql` — `gambling_stats` table
- [x] DB operations — `RemoveCoins`, `UpdateGamblingStats`, `GetGamblingStats`
- [x] `/gamble coinflip` command — bet coins on a coin flip
- [x] `/gamble slots` command — bet coins on a slot machine

### Phase 37 — Confessions System
- [x] `migrations/034_confessions.sql` — `confession_config` table
- [x] DB operations — `SetConfessionChannel`, `GetConfessionChannel`
- [x] `/confession setup` command to configure the channel
- [x] `/confess` command to anonymously post confessions

### Phase 38 — To-Do List System
- [x] `migrations/035_todos.sql` — `todos` table
- [x] DB operations — `AddTodo`, `GetTodos`, `CompleteTodo`, `RemoveTodo`
- [x] `/todo` command with `add`, `list`, `complete`, and `remove` subcommands

### Phase 39 — Role Menu System
- [x] `migrations/036_role_menus.sql` — `role_menus` and `role_menu_options` tables
- [x] DB operations — `CreateRoleMenu`, `AddRoleMenuOption`, `GetRoleMenu`
- [x] `/rolemenu` command with `setup` and `add_role` subcommands
- [x] Interaction handler for `role_menu_select` drop-down to assign/remove roles

- Implemented Phase 40 Quotes System features: added migrations `037_quotes.sql` with table `quotes`.
- Added DB operations `AddQuote`, `GetQuote`, `GetRandomQuote`, and `DeleteQuote` in `internal/db/db.go`.
- Added `/quote` command with `add`, `get`, `random`, and `delete` subcommands in `internal/bot/commands/quote.go`.
- Updated `internal/bot/bot.go` to register the `quote` command.

- Implemented Phase 41 Music System Foundation features: added migrations `038_music.sql` with table `music_config`.
- Added DB operations `SetMusicChannel` and `GetMusicChannel` in `internal/db/db.go`.
- Added `/music setup` command in `internal/bot/commands/music.go`.
- Added `/play` command placeholder in `internal/bot/commands/play.go`.
- Updated `internal/bot/bot.go` to register the `music` and `play` commands.

- Implemented Phase 42 Report System features: added migrations `039_reports.sql` with tables `report_config` and `reports`.
- Added DB operations `SetReportChannel`, `GetReportChannel`, and `CreateReport` in `internal/db/db.go`.
- Added `/report` command with `setup` and `user` subcommands in `internal/bot/commands/report.go`.
- Updated `internal/bot/bot.go` to register the `report` command.

- Implemented Phase 43 Welcome System Extension features: added migrations `040_welcome_images.sql` with table `welcome_images`.
- Added DB operations `SetWelcomeImage` and `GetWelcomeImage` in `internal/db/db.go`.
- Added `/welcome` command with `setup-image` and `test` subcommands in `internal/bot/commands/welcome.go`.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to support image embedding.
- Updated `internal/bot/bot.go` to register the `welcome` command.

- Implemented Phase 44 Reminder System Extension features:
- Added DB operations `GetPendingRemindersForGuild`, `DeleteAllRemindersForUser`, and `SnoozeReminder` in `internal/db/db.go`.
- Added `/remind list-all` and `/remind delete-all` subcommands to `internal/bot/commands/remind.go`.
- Updated `checkReminders` in `internal/bot/bot.go` to send a "Snooze 10m" button with reminders.
- Added interaction handler for the `snooze_` custom ID to parse and execute reminder snoozing.

### Phase 40 — Quotes System
- [x] `migrations/037_quotes.sql` — `quotes` table
- [x] DB operations — `AddQuote`, `GetQuote`, `GetRandomQuote`, `DeleteQuote`
- [x] `/quote` command with `add`, `get`, `random`, and `delete` subcommands

### Phase 41 — Music System Foundation
- [x] `migrations/038_music.sql` — `music_config` table
- [x] DB operations — `SetMusicChannel`, `GetMusicChannel`
- [x] `/music setup` command to configure the music channel
- [x] `/play` command placeholder (just replies with "Coming soon")

### Phase 42 — Report System
- [x] `migrations/039_reports.sql` — `report_config` and `reports` tables
- [x] DB operations — `SetReportChannel`, `GetReportChannel`, `CreateReport`
- [x] `/report` command with `setup` and `user` subcommands

### Phase 43 — Welcome System Extension
- [x] `migrations/040_welcome_images.sql` — `welcome_images` table
- [x] DB operations — `SetWelcomeImage`, `GetWelcomeImage`
- [x] `/welcome` command with `setup-image` and `test` subcommands
- [x] Update `GuildMemberAdd` handler to support image embedding

### Phase 44 — Reminder System Extension
- [x] Add `/remind list-all` subcommand to view all reminders for a server (Admin only)
- [x] Add `/remind delete-all` subcommand to clear all reminders for a user
- [x] Add `/remind snooze` interaction for reminder messages
- [x] Update reminder delivery logic to include a snooze button component

### Phase 45 — Leveling System Extension
- [x] Add `/rank role-rewards` subcommand to view all level roles in the server
- [x] Add `/rank set-background` subcommand to set a custom profile background image URL
- [x] `migrations/041_level_backgrounds.sql` — add `background_url` to `user_economy` table
- [x] Update `/rank` embed to use the custom background image if set

- Implemented Phase 45 Leveling System Extension features: added migrations `041_level_backgrounds.sql` with table `user_economy` update.
- Added DB operation `SetBackgroundURL` in `internal/db/db.go`.
- Added `/rank view`, `/rank set-background`, and `/rank role-rewards` subcommands to `internal/bot/commands/rank.go`.
- Updated `/rank view` embed to use the custom background image if set.

### Phase 46 — Utility Commands
- [x] Add `/userinfo` command to display detailed information about a user (joined date, created date, roles, etc.)
- [x] Add `/serverinfo` command to display detailed information about the server (member count, creation date, boost level, etc.)
- [x] Add `/avatar` command to view a user's avatar in high resolution

- Implemented Phase 46 Utility Commands features:
- Added `/userinfo` command in `internal/bot/commands/userinfo.go` to display detailed user information.
- Added `/serverinfo` command in `internal/bot/commands/serverinfo.go` to display detailed server information.
- Added `/avatar` command in `internal/bot/commands/avatar.go` to display a user's avatar in high resolution.
- Updated `internal/bot/bot.go` to register the `userinfo`, `serverinfo`, and `avatar` commands.

### Phase 47 — Auto-Roles System
- [x] `migrations/042_autoroles.sql` — `autorole_config` table
- [x] DB operations — `SetAutoRole`, `GetAutoRole`
- [x] `/autorole setup` command to configure a role to be assigned automatically
- [x] Update `GuildMemberAdd` handler to assign the configured auto-role

- Implemented Phase 47 Auto-Roles System features: added migrations `042_autoroles.up.sql` and `042_autoroles.down.sql` with table `autorole_config`.
- Added DB operations `SetAutoRole` and `GetAutoRole` in `internal/db/db.go`.
- Added `/autorole setup` command in `internal/bot/commands/autorole.go`.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to assign the configured auto-role and registered the `autorole` command.

### Phase 48 — Media Only Channels
- [x] `migrations/043_media_channels.sql` — `media_channels` table
- [x] DB operations — `AddMediaChannel`, `RemoveMediaChannel`, `ListMediaChannels`, `IsMediaChannel`
- [x] `/mediachannel` command with `add`, `remove`, and `list` subcommands
- [x] Update `messageCreateHandler` to delete messages without attachments or URLs in media channels

- Implemented Phase 48 Media Only Channels features: added migrations `043_media_channels.up.sql` and `043_media_channels.down.sql` with table `media_channels`.
- Added DB operations `AddMediaChannel`, `RemoveMediaChannel`, `ListMediaChannels`, and `IsMediaChannel` in `internal/db/db.go`.
- Added `/mediachannel` command in `internal/bot/commands/mediachannel.go`.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to delete messages without attachments or URLs in configured media channels.

### Phase 49 — React Roles Enhancement
- [x] `migrations/044_reaction_roles_update.sql` — add `emoji_id` and `is_custom` columns to support custom emojis
- [x] DB operations — update `AddReactionRole`, `RemoveReactionRole`, and `GetReactionRoles` to handle custom emojis
- [x] `/reactionrole` command — update to parse and store custom emojis
- [x] Update `MessageReactionAdd` / `MessageReactionRemove` handlers to check for custom emojis

- Implemented Phase 49 React Roles Enhancement features: added migrations `044_reaction_roles_update.up.sql` and `044_reaction_roles_update.down.sql` with columns `emoji_id` and `is_custom`.
- Updated DB operations `AddReactionRole`, `RemoveReactionRole`, and `GetReactionRole` in `internal/db/db.go` to support custom emojis.
- Updated `/reactionrole` command in `internal/bot/commands/reactionrole.go` to parse custom emojis properly and store the emoji name and ID separately.
- Updated `messageReactionAddHandler` and `messageReactionRemoveHandler` in `internal/bot/bot.go` to handle reaction interactions with custom emojis appropriately.

### Phase 50 — User Badges System
- [x] `migrations/045_user_badges.sql` — `user_badges` and `available_badges` tables
- [x] DB operations — `CreateBadge`, `AwardBadge`, `RemoveBadge`, `GetUserBadges`, `GetAllBadges`
- [x] `/badge` command with `create`, `award`, `remove`, and `list` subcommands
- [x] Update `/profile` command embed to display a user's earned badges

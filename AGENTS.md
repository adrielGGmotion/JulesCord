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

- Implemented Phase 109 Leveling Multipliers: added migrations `094_leveling_multipliers.up.sql` and `094_leveling_multipliers.down.sql` with table `leveling_multipliers`.
- Added DB operations `AddLevelMultiplier`, `RemoveLevelMultiplier`, and `GetLevelMultipliers` in `internal/db/db.go`.
- Added `/levelmultiplier` command in `internal/bot/commands/levelmultiplier.go` with `add`, `remove`, and `list` subcommands. Registered it in `internal/bot/bot.go`.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to calculate and apply the highest applicable role multiplier to earned XP.

- Implemented Phase 108 Message Forwarding: added migrations `093_message_forwarding.up.sql` and `093_message_forwarding.down.sql` with table `forwarding_config`.
- Added DB operations `AddForwardingRule`, `RemoveForwardingRule`, and `GetForwardingRules` in `internal/db/db.go`.
- Added `/forward` command in `internal/bot/commands/forward.go` with `add`, `remove`, and `list` subcommands. Registered it in `internal/bot/bot.go`.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to forward messages according to the configured rules.













- Implemented Phase 105 Message Snippets features: added migrations `090_message_snippets.up.sql` and `090_message_snippets.down.sql` with table `message_snippets`.
- Added DB operations `AddSnippet`, `RemoveSnippet`, `GetSnippet`, and `ListSnippets` in `internal/db/db.go`.
- Added `/snippet` command in `internal/bot/commands/snippet.go` with `add`, `remove`, `list`, and `send` subcommands. Registered it in `internal/bot/bot.go`.

- Implemented Phase 101 Welcome DMs System: added migrations `086_welcome_dms.up.sql` and `086_welcome_dms.down.sql` with table `welcome_dm_config`.
- Added DB operations `SetWelcomeDM`, `GetWelcomeDM`, and `ToggleWelcomeDM` in `internal/db/db.go`.
- Added `/welcomedm` command in `internal/bot/commands/welcomedm.go` with `set`, `enable`, and `disable` subcommands.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to send the configured welcome DM when a user joins the server. Registered the `welcomedm` command.

- Implemented Phase 93 Auto-Publish (Crosspost) Messages features: added migrations `079_auto_publish.up.sql` and `079_auto_publish.down.sql` with table `auto_publish_config`.
- Added DB operations `AddAutoPublishChannel`, `IsAutoPublishChannel`, and `RemoveAutoPublishChannel` in `internal/db/db.go`.
- Added `/autopublish` command in `internal/bot/commands/autopublish.go` with `add` and `remove` subcommands.
- Updated `internal/bot/bot.go` to register the `autopublish` command.

- Implemented Phase 91 Thread Management features: added migrations `077_thread_management.up.sql` and `077_thread_management.down.sql` with table `thread_config`.
- Added DB operations `SetThreadConfig` and `GetThreadConfig` in `internal/db/db.go`.
- Added `/thread` command with `setup`, `lock`, and `unlock` subcommands in `internal/bot/commands/thread.go`.
- Updated `internal/bot/bot.go` to register the `thread` command.


- Implemented Phase 90 Ticket Transcripts features: added migrations `076_ticket_transcripts.up.sql` and `076_ticket_transcripts.down.sql` with table `ticket_transcripts`.
- Added DB operations `SaveTicketTranscript` and `GetTicketTranscripts` in `internal/db/db.go`.
- Enhanced `/ticket close` command in `internal/bot/commands/ticket.go` to generate and DM channel transcripts to the user before deleting the channel.
- Added `/ticket transcripts` command to allow users to view their saved transcripts.


- Implemented Phase 87 Reaction Roles Logging features:
- Updated `messageReactionAddHandler` in `internal/bot/bot.go` to log role assignments if `advanced_log_config` enables role logging.
- Updated `messageReactionRemoveHandler` in `internal/bot/bot.go` to log role removals if `advanced_log_config` enables role logging.

- Implemented Phase 86 Advanced Logging System features: added migrations `073_advanced_logging.up.sql` and `073_advanced_logging.down.sql` with table `advanced_log_config`.
- Added DB operations `SetAdvancedLogConfig` and `GetAdvancedLogConfig` in `internal/db/db.go`.
- Added `/advancedlog` command in `internal/bot/commands/advancedlog.go` to configure detailed event logging per channel.
- Updated `internal/bot/bot.go` to register the `advancedlog` command and implemented enhanced event tracking and routing for channel and role events.

- Implemented Phase 85 Advanced Anti-Spam features: added migrations `072_anti_spam.up.sql` and `072_anti_spam.down.sql` with table `anti_spam_config`.
- Added DB operations `SetAntiSpamConfig` and `GetAntiSpamConfig` in `internal/db/db.go`.
- Added `/antispam` command in `internal/bot/commands/antispam.go` to configure message limits and mute durations.
- Updated `internal/bot/bot.go` to register the `antispam` command.


- Implemented Phase 81 Channel Moderation Commands features:
- Added `/lock` command in `internal/bot/commands/lock.go` to deny SendMessages for the `@everyone` role in the current channel.
- Added `/unlock` command in `internal/bot/commands/unlock.go` to remove the SendMessages deny overwrite for the `@everyone` role.
- Added `/slowmode` command in `internal/bot/commands/slowmode.go` to set the channel slowmode duration.
- Updated `internal/bot/bot.go` to register the `lock`, `unlock`, and `slowmode` commands.
- Implemented Phase 80 Moderation Unban and Clear Warnings features:
- Added DB operations `RemoveWarning` and `ClearWarnings` in `internal/db/db.go`.
- Added `/unban` command in `internal/bot/commands/unban.go` to unban a user by ID and mark active temp bans as resolved.
- Added `/clearwarnings` command in `internal/bot/commands/clearwarnings.go` to clear a user's warning history.
- Updated `internal/bot/bot.go` to register the `unban` and `clearwarnings` commands.

- Implemented Phase 78 Nickname Automation features: added migrations `068_nicknames.up.sql` and `068_nicknames.down.sql` with table `nickname_config`.
- Added DB operations `SetNicknameTemplate` and `GetNicknameTemplate` in `internal/db/db.go`.
- Added `/nicktemplate` command with `set` and `view` subcommands in `internal/bot/commands/nicktemplate.go`.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to automatically apply the configured nickname template to new members.
- Updated `internal/bot/bot.go` to register the `nicktemplate` command.

- Implemented Phase 77 Role Rewards Extension features: added migrations `067_level_role_rewards.up.sql` and `067_level_role_rewards.down.sql` to add `coins_reward` to `level_roles` table.
- Added `coins_reward` support to `SetLevelRole`, `GetLevelRole`, and `GetLevelRoles` in `internal/db/db.go`.
- Updated `/levelrole add` command to accept a `coins` reward amount, and `/levelrole list` to display it in `internal/bot/commands/levelrole.go`.

- Implemented Phase 76 Advanced User Configuration features: added migrations `066_user_config.up.sql` and `066_user_config.down.sql` with table `user_config`.
- Added DB operations `SetUserConfig` and `GetUserConfig` in `internal/db/db.go`.
- Added `/settings` command with `dnd` and `dm-notifications` subcommands in `internal/bot/commands/settings.go`.
- Updated `internal/bot/bot.go` to register the `settings` command.

- Implemented Phase 75 Server Highlights features: added migrations `065_highlights.up.sql` and `065_highlights.down.sql` with table `highlights`.
- Added DB operations `AddHighlight`, `GetHighlights`, and `RemoveHighlight` in `internal/db/db.go`.
- Added `/highlight` command with `add`, `list`, and `remove` subcommands in `internal/bot/commands/highlight.go`.
- Updated `internal/bot/bot.go` to register the `highlight` command.

- Implemented Phase 74 Profile Links System features: added migrations `064_profile_links.up.sql` and `064_profile_links.down.sql` to add `website`, `github`, and `twitter` columns to `user_profiles`.
- Added DB operation `SetProfileLinks` and updated `GetProfile` in `internal/db/db.go`.
- Added `set-links` subcommand to `/profile` command in `internal/bot/commands/profile.go` and updated `view` embed to display social links.
- Implemented Phase 73 Custom Roles System features: added migrations `063_custom_roles.up.sql` and `063_custom_roles.down.sql` with table `custom_roles`.
- Added DB operations `CreateCustomRole`, `GetCustomRole`, `UpdateCustomRole`, and `DeleteCustomRole` in `internal/db/db.go`.
- Added `/myrole` command with `create`, `name`, `color`, `icon`, and `delete` subcommands in `internal/bot/commands/myrole.go`.
- Updated `internal/bot/bot.go` to register the `myrole` command.
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

---

- Implemented Phase 83 Role Information Commands features:
- Added `/roleinfo` command in `internal/bot/commands/roleinfo.go` to display detailed information about a role.
- Updated `/userinfo` command in `internal/bot/commands/userinfo.go` to include the highest role.
- Updated `internal/bot/bot.go` to register the `roleinfo` command.


- Implemented Phase 88 Dynamic Voice Channels features: added migrations `074_dynamic_voice.up.sql` and `074_dynamic_voice.down.sql` with table `dynamic_voice_config`.
- Added DB operations `SetDynamicVoiceConfig` and `GetDynamicVoiceConfig` in `internal/db/db.go`.
- Added `/dynamicvoice` command with `setup` subcommand in `internal/bot/commands/dynamicvoice.go`.
- Updated `voiceStateUpdateHandler` in `internal/bot/bot.go` to support dynamic voice creation, moving users into them, and deleting them when empty.
- Updated `internal/bot/bot.go` to register the `dynamicvoice` command.


- Implemented Phase 89 Reaction Role Groups features: added migrations `075_reaction_role_groups.up.sql` and `075_reaction_role_groups.down.sql` with table `reaction_role_groups` and altering `reaction_roles` table to add `group_id`.
- Added DB operations `CreateReactionRoleGroup`, `GetReactionRoleGroups`, `AssignRoleToGroup`, `GetGroupRoles`, and `GetReactionRoleGroup` in `internal/db/db.go`.
- Added `/reactiongroup` command with `create`, `list`, and `addrole` subcommands in `internal/bot/commands/reactiongroup.go`.
- Updated `messageReactionAddHandler` in `internal/bot/bot.go` to enforce exclusivity by removing other roles in the group when a role is assigned.
- Updated `internal/bot/bot.go` to register the `reactiongroup` command.

## Task Checklist

### Phase 109 — Leveling Multipliers
- [x] `migrations/094_leveling_multipliers.up.sql` — `leveling_multipliers` table (guild_id, role_id, multiplier)
- [x] DB operations — `AddLevelMultiplier`, `RemoveLevelMultiplier`, `GetLevelMultipliers`
- [x] `/levelmultiplier` command with `add`, `remove`, and `list` subcommands
- [x] Update `messageCreateHandler` to apply the highest multiplier to earned XP based on user's roles

### Phase 108 — Message Forwarding
- [x] `migrations/093_message_forwarding.up.sql` — `forwarding_config` table (guild_id, source_channel_id, target_channel_id)
- [x] DB operations — `AddForwardingRule`, `RemoveForwardingRule`, `GetForwardingRules`
- [x] `/forward` command with `add`, `remove`, and `list` subcommands
- [x] Update `messageCreateHandler` to copy and forward messages according to the rules

#
### Phase 107 — Thread Automation Config
- [x] `migrations/092_thread_automation.up.sql` — `thread_automation_config` table (guild_id, channel_id, auto_join)
- [x] DB operations — `SetThreadAutomation`, `GetThreadAutomation`, `RemoveThreadAutomation`
- [x] `/threadauto` command with `setup` and `remove` subcommands
- [x] Update `threadCreateHandler` in `internal/bot/bot.go` to automatically join threads in configured channels

## Phase 106 — Message Translation
- [x] `migrations/091_translation_config.sql` — `translation_config` table (guild_id, default_language)
- [x] DB operations — `SetTranslationConfig`, `GetTranslationConfig`
- [x] `/translate` command with `text` subcommand (source, target, text options)
- [x] Update `bot.go` to register the `translate` command

### Phase 105 — Message Snippets / Macros
- [x] `migrations/090_message_snippets.up.sql` — `message_snippets` table (id, guild_id, name, content)
- [x] DB operations — `AddSnippet`, `RemoveSnippet`, `GetSnippet`, `ListSnippets`
- [x] `/snippet` command with `add`, `remove`, `list`, and `send` subcommands
- [x] Update `internal/bot/bot.go` to register the `snippet` command


### Phase 104 — User Warn Level Automation
- [x] `migrations/089_warn_automation.sql` — `warn_automation_config` table (guild_id, warning_threshold, action, duration)
- [x] DB operations — `AddWarnAutomationRule`, `RemoveWarnAutomationRule`, `GetWarnAutomationRules`
- [x] `/warnautomod` command with `add` and `remove` subcommands
- [x] Update `/warn` command logic to evaluate warning count and automatically apply configured punishments (mute, kick, ban)

### Phase 103 — Leveling Channel Blacklist System
- [x] `migrations/088_leveling_channel_blacklist.sql` — `leveling_channel_blacklist` table (id, guild_id, channel_id)
- [x] DB operations — `AddLevelingChannelBlacklist`, `RemoveLevelingChannelBlacklist`, `GetLevelingChannelBlacklists`
- [x] `/levelchannelblacklist` command with `add`, `remove`, and `list` subcommands
- [x] Update `messageCreateHandler` to skip awarding XP if the channel is blacklisted

- Implemented Phase 65 Reputation Leaderboard features:
- Added DB operation `GetTopReputationUsers` in `internal/db/db.go`.
- Added `/replb` command in `internal/bot/commands/replb.go`.
- Updated `internal/bot/bot.go` to register the `replb` command.

- Implemented Phase 66 Level Leaderboard features:
- Added DB operation `GetTopLevelUsers` in `internal/db/db.go`.
- Added `/levellb` command in `internal/bot/commands/levellb.go`.
- Updated `internal/bot/bot.go` to register the `levellb` command.

- Implemented Phase 67 User Roles Sync features: added migrations `059_user_roles.up.sql` and `059_user_roles.down.sql` with table `user_roles`.
- Added DB operations `SaveUserRoles` and `GetUserRoles` in `internal/db/db.go`.
- Added `guildMemberRemoveHandler` in `internal/bot/bot.go` to save user roles when a user leaves a server.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to restore previously saved roles when a user rejoins a server.

- Implemented Phase 68 Mod Logs Extension features: added migrations `060_mod_log_updates.up.sql` and `060_mod_log_updates.down.sql` to add `duration`, `resolved`, and `evidence_url` columns to `mod_actions`.
- Added DB operations `GetActiveBans`, `GetActiveMutes`, and `MarkModActionResolved` in `internal/db/db.go`.
- Enhanced `/warn`, `/kick`, `/ban`, and `/mute` commands to accept evidence attachments (images/logs).
- Updated the Mod Action Log embed to display the attached evidence.

- Implemented Phase 69 Web Dashboard Guild Settings features:
- Backend: Updated `GET /api/guilds/:id/config` to return all settings (prefix, log channel, welcome channel, auto-role, counting channel, suggestion channel).
- Backend: Updated `PATCH /api/guilds/:id/config` to allow updating all these settings.
- Frontend: Created GuildSettings component at `web/src/pages/GuildSettings.jsx`.
- Frontend: Updated routing in `web/src/App.jsx` to include the new Guild Settings page.
- Frontend: Updated Guilds page `web/src/pages/Guilds.jsx` to include an Action column with links to the settings page.


- Implemented Phase 70 Music System Enhancements features: added migrations `061_music_queue.up.sql` and `061_music_queue.down.sql` with table `music_queue`.
- Added DB operations `PlayMusic`, `SkipMusic`, `StopMusic`, and `GetQueue` in `internal/db/db.go`.
- Updated `/play` command in `internal/bot/commands/play.go` to insert songs into the DB queue instead of showing a placeholder.
- Added `/skip` command to remove the currently playing song from the queue in `internal/bot/commands/skip.go`.
- Added `/stop` command to clear the music queue in `internal/bot/commands/stop.go`.
- Updated `internal/bot/bot.go` to register the `skip` and `stop` commands and pass the DB connection to the `play` command.


- Implemented Phase 71 Fun Commands features:
- Added `/8ball` command in `internal/bot/commands/8ball.go`.
- Added `/roll` command in `internal/bot/commands/roll.go`.
- Added `/rps` command in `internal/bot/commands/rps.go`.
- Updated `internal/bot/bot.go` to register the `8ball`, `roll`, and `rps` commands.

### Phase 72 — Fact System
- [x] `migrations/062_facts.sql` — `facts` table (id, guild_id, text, author_id, created_at)
- [x] DB operations — `AddFact`, `GetRandomFact`, `DeleteFact`
- [x] `/fact` command with `add`, `random`, and `delete` subcommands in `internal/bot/commands/fact.go`
- [x] Update `internal/bot/bot.go` to register the `fact` command

### Phase 73 — Custom Roles System
- [x] `migrations/063_custom_roles.sql` — `custom_roles` table (id, guild_id, user_id, role_id, name, color, icon_url)
- [x] DB operations — `CreateCustomRole`, `GetCustomRole`, `UpdateCustomRole`, `DeleteCustomRole`
- [x] `/myrole` command with `create`, `name`, `color`, `icon`, and `delete` subcommands in `internal/bot/commands/myrole.go`
- [x] Update `internal/bot/bot.go` to register the `myrole` command

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



### Phase 90 — Ticket Transcripts
- [x] `migrations/076_ticket_transcripts.sql` — `ticket_transcripts` table (id, ticket_id, channel_id, guild_id, user_id, transcript_url, created_at)
- [x] DB operations — `SaveTicketTranscript`, `GetTicketTranscripts`
- [x] Enhance `/ticket close` command to automatically generate a transcript of the channel messages and save it before deleting the channel.
- [x] Add `/ticket transcripts` command to view saved transcripts for a user.

### Phase 89 — Reaction Role Groups (Exclusive Roles)
- [x] `migrations/075_reaction_role_groups.sql` — `reaction_role_groups` table (id, guild_id, name, is_exclusive, max_roles)
- [x] DB operations — `CreateReactionRoleGroup`, `GetReactionRoleGroups`, `AssignRoleToGroup`
- [x] `/reactiongroup` command — `create`, `list`, and `addrole` subcommands
- [x] Update `messageReactionAddHandler` to enforce exclusivity (remove other roles in group when one is selected)

### Phase 91 — Thread Management
- [x] `migrations/077_thread_management.sql` — `thread_config` table (guild_id, auto_archive_duration)
- [x] DB operations — `SetThreadConfig`, `GetThreadConfig`
- [x] `/thread` command — `setup` to configure auto-archive duration, `lock`, `unlock`
- [x] Update `messageCreateHandler` to enforce thread auto-archive durations

- Implemented Phase 92 Voice Channel Generator features: added migrations `078_voice_generator.up.sql` and `078_voice_generator.down.sql` with table `voice_generator_config`.
- Added DB operations `SetVoiceGeneratorConfig` and `GetVoiceGeneratorConfig` in `internal/db/db.go`.
- Added `/voicegen` command in `internal/bot/commands/voicegen.go` with a `setup` subcommand.
- Updated `internal/bot/bot.go` to register the `voicegen` command and implemented dynamic voice channel generation logic in `voiceStateUpdateHandler` alongside automatic empty channel deletion.

### Phase 92 — Voice Channel Generator
- [x] `migrations/078_voice_generator.sql` — `voice_generator_config` table (guild_id, base_channel_id, max_channels)
- [x] DB operations — `SetVoiceGeneratorConfig`, `GetVoiceGeneratorConfig`
- [x] `/voicegen` command — `setup`
- [x] Update `voiceStateUpdateHandler` to generate voice channels when base channel joined

### Phase 88 — Dynamic Voice Channels
- [x] `migrations/074_dynamic_voice.sql` — `dynamic_voice_config` table (guild_id, category_id, trigger_channel_id)
- [x] DB operations — `SetDynamicVoiceConfig`, `GetDynamicVoiceConfig`
- [x] `/dynamicvoice` command — `setup` subcommand
- [x] Update `voiceStateUpdateHandler` to support multiple dynamic voice instances per user


### Phase 94 — Role Commands
- [x] DB operations — `AddRole`, `RemoveRole`, `HasRole` (Optional, as this mainly uses Discord API)
- [x] `/role` command — `add`, `remove`, `info` subcommands
- [x] Update `bot.go` to register the `role` command

### Phase 95 — Sticky Roles System
- [x] `migrations/080_sticky_roles.sql` — `sticky_roles` table (guild_id, user_id, role_id)
- [x] DB operations — `SaveStickyRole`, `GetStickyRoles`, `RemoveStickyRole`
- [x] `/stickyrole` command — `add`, `remove`, `list`
- [x] Update `guildMemberAddHandler` to restore sticky roles when a user leaves and rejoins

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

- Implemented Phase 33 Trivia System features: added migrations `030_trivia.sql` with table `trivia_scores`.
- Added DB operations `AddTriviaScore`, `GetTriviaLeaderboard`, and `AddCoins` in `internal/db/db.go`.
- Added `/trivia` command with `start` and `leaderboard` subcommands in `internal/bot/commands/trivia.go` fetching questions from OpenTDB.
- Updated `interactionCreateHandler` in `internal/bot/bot.go` to process trivia answer buttons, award coins, and update the leaderboard.

- Implemented Phase 34 Custom Commands features: added migrations `031_custom_commands.sql` with table `custom_commands`.
- Added DB operations `AddCustomCommand`, `RemoveCustomCommand`, `ListCustomCommands`, and `GetCustomCommand` in `internal/db/db.go`.
- Added `/customcommand` command with `add`, `remove`, and `list` subcommands in `internal/bot/commands/customcommand.go`.

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

### Phase 51 — Economy Transfer System
- [x] `migrations/046_economy_transfers.sql` — `transfers` table to track coin transfers
- [x] DB operation — `TransferCoins` inside a transaction
- [x] `/transfer` command to send coins to another user
- [x] `/transfers` command to view a user's transfer history

- Implemented Phase 51 Economy Transfer System features: added migrations `046_economy_transfers.up.sql` and `046_economy_transfers.down.sql` with table `transfers`.
- Added DB operations `TransferCoins` and `GetTransfers` in `internal/db/db.go`.
- Added `/transfer` command with user and amount options in `internal/bot/commands/transfer.go`.
- Added `/transfers` command to view recent transfer history in `internal/bot/commands/transfers.go`.
- Updated `internal/bot/bot.go` to register the `transfer` and `transfers` commands.

### Phase 52 — Economy Coinflip Bet
- [x] Update `/gamble coinflip` to allow betting all coins by specifying "all" as an amount string.
- [x] Add `/baltop` command to view the top 10 users by coins.

- Implemented Phase 52 Economy Coinflip Bet features:
- Updated `/gamble coinflip` and `/gamble slots` commands in `internal/bot/commands/gamble.go` to accept an `amount` string, which can be a number or "all".
- Added DB operation `GetTopUsersByCoins` in `internal/db/db.go`.
- Added `/baltop` command in `internal/bot/commands/baltop.go` to display the top 10 users by coins.
- Updated `internal/bot/bot.go` to register the `baltop` command.

### Phase 53 — Advanced Moderation
- [x] `migrations/047_advanced_moderation.sql` — `mutes` table to track temporary timeouts
- [x] DB operations — `AddMute`, `GetMute`, `RemoveMute`, `GetActiveMutes`
- [x] `/mute` command to time out a user for a specific duration (e.g., 1h, 1d)
- [x] `/unmute` command to remove a timeout
- [x] Background goroutine to periodically check and remove expired mutes

### Phase 54 — Economy Activities
- [x] `migrations/048_economy_activities.sql` — add `last_work_at` and `last_crime_at` to `user_economy` table
- [x] DB operations — `UpdateWorkActivity`, `UpdateCrimeActivity`
- [x] `/work` command to earn 50-200 coins with a 1-hour cooldown
- [x] `/crime` command to attempt earning 200-500 coins with a 50% success rate (or lose 50-200) with a 2-hour cooldown

- Implemented Phase 53 Advanced Moderation features: added migrations `047_advanced_moderation.up.sql` and `047_advanced_moderation.down.sql` with table `mutes`.
- Added DB operations `AddMute`, `GetMute`, `RemoveMute`, and `GetExpiredMutes` in `internal/db/db.go`.
- Added `/mute` and `/unmute` commands in `internal/bot/commands/mute.go` and `internal/bot/commands/unmute.go`.
- Added background goroutine `checkExpiredMutes` in `internal/bot/bot.go` to remove expired mutes.

- Implemented Phase 54 Economy Activities features: added migrations `048_economy_activities.up.sql` and `048_economy_activities.down.sql` to add `last_work_at` and `last_crime_at` to `user_economy` table.
- Added DB operations `UpdateWorkActivity` and `UpdateCrimeActivity` in `internal/db/db.go`.
- Added `/work` command to earn 50-200 coins with a 1-hour cooldown in `internal/bot/commands/work.go`.
- Added `/crime` command to attempt earning 200-500 coins with a 50% success rate (or lose 50-200) with a 2-hour cooldown in `internal/bot/commands/crime.go`.
- Updated `internal/bot/bot.go` to register the `work` and `crime` commands.

### Phase 55 — Economy Robbing
- [x] Add `last_rob_at` column to `user_economy` table via migration.
- [x] Update DB operations to handle `last_rob_at` and a `RobCoins` transaction (transferring coins directly between users).
- [x] Add `/rob` command allowing users to try stealing from others, with success rate based on target's balance vs robber's balance, and a cooldown.
- [x] Implement a system where failed robberies fine the robber and give it to the victim.

- Implemented Phase 55 Economy Robbing features: added migrations `049_economy_robbing.up.sql` and `049_economy_robbing.down.sql` to add `last_rob_at` to `user_economy` table.
- Added DB operations `UpdateRobActivity` and `RobCoins` in `internal/db/db.go`.
- Added `/rob` command in `internal/bot/commands/rob.go` handling success rates, fines, and transaction execution.
- Updated `internal/bot/bot.go` to register the `rob` command.

### Phase 56 — Economy Items & Use
- [x] `migrations/050_economy_items.sql` — `user_items` table mapping `user_id`, `item_id`, `quantity`.
- [x] DB operations — `AddItem`, `RemoveItem`, `GetUserItems`
- [x] Update `/shop` to give items to `user_items` instead of a flat log.
- [x] `/use` command to use items from inventory (with predefined effects).

- Implemented Phase 56 Economy Items & Use features: added migrations `050_economy_items.up.sql` and `050_economy_items.down.sql` with table `user_items`.
- Added DB operations `AddUserItem`, `RemoveUserItem`, and `GetUserItems` in `internal/db/db.go`, and updated `BuyItem` to correctly track quantity.
- Updated `/inventory` command to accurately group and display item quantities.
- Added `/use` command in `internal/bot/commands/use.go` with successful deduction logic and user feedback.
- Updated `internal/bot/bot.go` to register the `use` command, and retroactively registered the missed `rob` command.

- Implemented Phase 57 Bank & Interest System features: added migrations `051_bank.up.sql` and `051_bank.down.sql` to add `bank` and `last_interest_at` to `user_economy` table.
- Added DB operations `DepositCoins`, `WithdrawCoins`, and `ApplyInterest` in `internal/db/db.go`.
- Added `/bank` command with `deposit`, `withdraw`, and `balance` subcommands in `internal/bot/commands/bank.go`.
- Added background goroutine `applyInterestLoop` in `internal/bot/bot.go` to apply 1% interest daily to eligible bank balances.
- Updated `internal/bot/bot.go` to register the `bank` command.

### Phase 57 — Bank & Interest System
- [x] `migrations/051_bank.sql` — add `bank` and `last_interest_at` to `user_economy` table
- [x] DB operations — `DepositCoins`, `WithdrawCoins`, `ApplyInterest`
- [x] `/bank` commands — `deposit`, `withdraw`, `balance`
- [x] Background goroutine to apply 1% interest daily to bank balances

- Implemented Phase 58 Pet System features: added migrations `052_pets.up.sql` and `052_pets.down.sql` with table `user_pets`.
- Added DB operations `AdoptPet`, `FeedPet`, `PlayPet`, `GetPet`, and `UpdateAllPetStats` in `internal/db/db.go`.
- Added `/pet` command with `adopt`, `view`, `feed`, and `play` subcommands in `internal/bot/commands/pet.go`.
- Added background goroutine `petStatsLoop` in `internal/bot/bot.go` to slowly increase pet hunger and decrease happiness over time.
- Updated `internal/bot/bot.go` to register the `pet` command.

### Phase 58 — Pet System
- [x] `migrations/052_pets.sql` — `user_pets` table (pet name, type, hunger, happiness)
- [x] DB operations — `AdoptPet`, `FeedPet`, `PlayPet`, `GetPet`
- [x] `/pet adopt` and `/pet view` commands
- [x] Background goroutine to slowly increase pet hunger over time

- Implemented Phase 59 Jobs System features: added migrations `053_jobs.up.sql` and `053_jobs.down.sql` with table `available_jobs` and `job_id` to `user_economy`.
- Added DB operations `CreateJob`, `GetJobs`, `GetJob`, `SetUserJob`, and `RemoveUserJob` in `internal/db/db.go`.
- Added `/job` command with `create`, `list`, `apply`, `quit`, and `info` subcommands in `internal/bot/commands/job.go`.
- Updated `/work` command in `internal/bot/commands/work.go` to grant coins based on user's job salary instead of a random amount if they have a job.
- Updated `internal/bot/bot.go` to register the `job` command.

### Phase 59 — Jobs System
- [x] `migrations/053_jobs.sql` — add `job_id` to `user_economy` table and create `available_jobs` table (name, description, salary, required_level)
- [x] DB operations — `CreateJob`, `GetJobs`, `GetJob`, `SetUserJob`, `RemoveUserJob`
- [x] `/job` commands — `list`, `apply`, `quit`, `info`
- [x] Update `/work` to grant coins based on user's job salary instead of a random amount if they have a job

### Phase 60 — Custom Prefixes
- [x] `migrations/054_custom_prefixes.sql` — add `prefix` column to `guilds` table
- [x] DB operations — `SetGuildPrefix`, `GetGuildPrefix`
- [x] `/prefix` command — allows admins to set a custom text prefix (e.g., `!`, `?`)
- [x] Update `messageCreateHandler` to check for custom prefix before checking for text commands

- Implemented Phase 60 Custom Prefixes features: added migrations `054_custom_prefixes.up.sql` and `054_custom_prefixes.down.sql` with `prefix` column to `guilds` table.
- Added DB operations `SetGuildPrefix` and `GetGuildPrefix` in `internal/db/db.go`.
- Added `/prefix` command allowing admins to view and update the server's custom prefix in `internal/bot/commands/prefix.go`.
- Updated `internal/bot/bot.go` to register the `prefix` command.

### Phase 61 — Auto-Threads
- [x] `migrations/055_auto_threads.sql` — add `auto_threads_config` table (channel_id, thread_name_template)
- [x] DB operations — `SetAutoThread`, `GetAutoThread`, `RemoveAutoThread`
- [x] `/autothread` command with `setup` and `remove` subcommands
- [x] Update `messageCreateHandler` to automatically create a thread for new messages in configured channels

- Implemented Phase 61 Auto-Threads System features: added migrations `055_auto_threads.up.sql` and `055_auto_threads.down.sql` with table `auto_threads_config`.
- Added DB operations `SetAutoThread`, `GetAutoThread`, and `RemoveAutoThread` in `internal/db/db.go`.
- Added `/autothread` command with `setup` and `remove` subcommands in `internal/bot/commands/autothread.go`.
- Updated `internal/bot/bot.go` to register the `autothread` command.

### Phase 62 — Voice XP System
- [x] `migrations/056_voice_xp.sql` — `voice_xp` table (user_id, guild_id, join_time)
- [x] DB operations — `SetVoiceJoinTime`, `GetVoiceJoinTime`, `RemoveVoiceJoinTime`
- [x] Update `voiceStateUpdateHandler` to track join time and calculate XP based on duration spent in VC when leaving
- [x] Award standard economy XP and coins based on VC duration

- Implemented Phase 62 Voice XP System features: added migrations `056_voice_xp.up.sql` and `056_voice_xp.down.sql` with table `voice_xp`.
- Added DB operations `SetVoiceJoinTime`, `GetVoiceJoinTime`, and `RemoveVoiceJoinTime` in `internal/db/db.go`.
- Updated `voiceStateUpdateHandler` in `internal/bot/bot.go` to track join time and dynamically award economy XP and coins based on VC duration upon leaving.

- Implemented Phase 63 Message Bookmarks features: added migrations `057_bookmarks.up.sql` and `057_bookmarks.down.sql` with table `bookmarks`.
- Added DB operations `AddBookmark`, `RemoveBookmark`, and `GetBookmarks` in `internal/db/db.go`.
- Added `Bookmark` message context command to save a message to the database.
- Added `/bookmarks` slash command with `list` and `remove` subcommands to view and manage saved bookmarks in `internal/bot/commands/bookmark.go`.
- Updated `internal/bot/bot.go` to register the `BookmarkContext` and `BookmarksSlash` commands.

- Implemented Phase 64 User Timezones features: added migrations `058_user_timezones.up.sql` and `058_user_timezones.down.sql` with table `user_timezones`.
- Added DB operations `SetUserTimezone` and `GetUserTimezone` in `internal/db/db.go`.
- Added `/timezone` command with `set` and `get` subcommands in `internal/bot/commands/timezone.go`.
- Updated `internal/bot/bot.go` to register the `timezone` command.

### Phase 63 — Message Bookmarks
- [x] `migrations/057_bookmarks.sql` — `bookmarks` table (user_id, message_id, channel_id, guild_id, note)
- [x] DB operations — `AddBookmark`, `RemoveBookmark`, `GetBookmarks`
- [x] `Bookmark` message context command — allows users to right click a message and save it to their DMs/DB
- [x] `/bookmarks` slash command — view, list and manage saved messages

### Phase 64 — User Timezones
- [x] `migrations/058_user_timezones.sql` — `user_timezones` table (`user_id`, `timezone`)
- [x] DB operations — `SetUserTimezone`, `GetUserTimezone`
- [x] `/timezone` command with `set` and `get` subcommands to allow users to set and view local times

### Phase 65 — Reputation Leaderboard
- [x] DB operation — `GetTopReputationUsers`
- [x] `/replb` command — displays the top 10 users with the highest reputation

### Phase 66 — Level Leaderboard
- [x] DB operation — `GetTopLevelUsers`
- [x] `/levellb` command — displays the top 10 users with the highest level


### Phase 67 — User Roles Sync
- [x] `migrations/059_user_roles.sql` — `user_roles` table (`user_id`, `guild_id`, `role_ids`)
- [x] DB operations — `SaveUserRoles`, `GetUserRoles`
- [x] Add `guildMemberRemoveHandler` to save user roles to the DB when a user leaves
- [x] Add logic to `guildMemberAddHandler` to restore previously saved roles when a user rejoins a server

### Phase 68 — Mod Logs Extension
- [x] `migrations/060_mod_log_updates.sql` — update `mod_actions` to include `duration` and `resolved` boolean for temp mutes/bans
- [x] DB operations — `GetActiveBans`, `GetActiveMutes`, `MarkModActionResolved`
- [x] Enhance `/warn`, `/kick`, `/ban`, and `/mute` to accept evidence attachments (images/logs)
- [x] Display attached evidence in the Mod Action Log embed

### Phase 69 — Web Dashboard Guild Settings
- [x] Backend: Update `GET /api/guilds/:id/config` to return all settings (prefix, log channel, welcome channel, auto-role, counting channel, suggestion channel)
- [x] Backend: Update `PATCH /api/guilds/:id/config` to allow updating all these settings
- [x] Frontend: Create GuildSettings component at `web/src/pages/GuildSettings.jsx`
- [x] Frontend: Update routing in `web/src/App.jsx` to include the new Guild Settings page

### Phase 70 — Music System Enhancements
- [x] Backend: Add `PlayMusic`, `SkipMusic`, `StopMusic`, and `GetQueue` DB operations in `internal/db/db.go` to mock/store currently playing music state.
- [x] Implement actual logic for `/play` command in `internal/bot/commands/play.go` using a mock URL player or text confirmation.
- [x] Add `/skip` command to skip the currently playing track in `internal/bot/commands/skip.go`.
- [x] Add `/stop` command to stop music and clear the queue in `internal/bot/commands/stop.go`.

### Phase 71 — Fun Commands
- [x] Add `/8ball` command in `internal/bot/commands/8ball.go` to answer yes/no questions
- [x] Add `/roll` command in `internal/bot/commands/roll.go` to roll virtual dice
- [x] Add `/rps` command in `internal/bot/commands/rps.go` to play Rock, Paper, Scissors against the bot
- [x] Update `internal/bot/bot.go` to register the `8ball`, `roll`, and `rps` commands

### Phase 74 — Profile Links System
- [x] `migrations/064_profile_links.sql` — add `website`, `github`, and `twitter` columns to `user_profiles` table
- [x] DB operations — update `GetProfile` to fetch the new links, and add `SetProfileLinks` to update them
- [x] `/profile set-links` command in `internal/bot/commands/profile.go` to allow users to set their social links
- [x] Update `/profile view` embed to display the user's social links

### Phase 75 — Server Highlights
- [x] `migrations/065_highlights.sql` — `highlights` table (id, guild_id, message_id, channel_id, author_id, added_by, created_at)
- [x] DB operations — `AddHighlight`, `GetHighlights`, `RemoveHighlight`
- [x] `/highlight` command with `add` (message link), `list`, and `remove` subcommands
- [x] Update `internal/bot/bot.go` to register the `highlight` command

### Phase 76 — Advanced User Configuration
- [x] `migrations/066_user_config.sql` — `user_config` table (user_id, dnd_mode, dm_notifications)
- [x] DB operations — `SetUserConfig`, `GetUserConfig`
- [x] `/settings` command with `dnd` and `dm-notifications` subcommands in `internal/bot/commands/settings.go`
- [x] Update `internal/bot/bot.go` to register the `settings` command

### Phase 77 — Role Rewards Extension
- [x] Update `level_roles` to include `coins_reward` for reaching a level
- [x] DB operations — `GetLevelRoleReward`, `SetLevelRoleReward`
- [x] Update `/levelrole add` to accept a coins reward amount
- [x] Update `messageCreateHandler` to award coins when assigning a level role

### Phase 78 — Nickname Automation
- [x] `migrations/068_nicknames.sql` — `nickname_config` table (guild_id, template)
- [x] DB operations — `SetNicknameTemplate`, `GetNicknameTemplate`
- [x] `/nicktemplate` command to configure the format (e.g. `[Member] {user}`)
- [x] Update `guildMemberAddHandler` to apply the nickname template when a user joins



- Implemented Phase 79 Temporary User Bans features: added migrations `069_temp_bans.up.sql` and `069_temp_bans.down.sql` with table `active_bans`.
- Added DB operations `AddTempBan`, `GetActiveTempBans`, `RemoveTempBan`, and `MarkAllUserModActionsResolved` in `internal/db/db.go`.
- Updated `/ban` command in `internal/bot/commands/ban.go` to accept an optional `duration` parameter (e.g., "1h", "7d"), parse it, and save the temporary ban.
- Added background goroutine `checkTempBans` in `internal/bot/bot.go` to automatically unban users when their temporary ban duration expires and mark the mod action as resolved.

### Phase 79 — Temporary User Bans
- [x] Update `migrations/060_mod_log_updates.sql` if needed, or create `migrations/069_temp_bans.sql` with `active_bans` table (user_id, guild_id, unban_at)
- [x] DB operations — `AddTempBan`, `GetActiveTempBans`, `RemoveTempBan`
- [x] Enhance `/ban` command to accept an optional `duration` parameter (e.g. "1h", "7d")
- [x] Background goroutine to periodically unban users when their temp ban duration expires

### Phase 80 — Moderation Unban and Clear Warnings
- [x] DB operations — `RemoveWarning`, `ClearWarnings`
- [x] `/unban` command to unban a user by their user ID
- [x] `/clearwarnings` command to clear a user's warning history
- [x] Update `internal/bot/bot.go` to register the `unban` and `clearwarnings` commands

### Phase 81 — Channel Moderation Commands
- [x] Add `/lock` command in `internal/bot/commands/lock.go` to deny SendMessages for the `@everyone` role in the current channel.
- [x] Add `/unlock` command in `internal/bot/commands/unlock.go` to remove the SendMessages deny overwrite for the `@everyone` role in the current channel.
- [x] Add `/slowmode` command in `internal/bot/commands/slowmode.go` to set the channel slowmode duration.
- [x] Update `internal/bot/bot.go` to register the `lock`, `unlock`, and `slowmode` commands.

### Phase 82 — Command Cooldowns
- [x] `migrations/070_command_cooldowns.sql` — `command_cooldowns` table (user_id, command, expires_at)
- [x] DB operations — `SetCommandCooldown`, `GetCommandCooldown`
- [x] Update `bot.go` interaction handler to enforce command cooldowns based on database state
- [x] Add `/cooldown` command to manage custom command cooldowns

### Phase 84 — Auto-Responder Enhancement
- [x] `migrations/071_auto_responder_update.sql` — add `is_regex` boolean to `auto_responders` table
- [x] DB operations — update `AddAutoResponder` and `ListAutoResponders` to support `is_regex`
- [x] `/autoresponder` command — enhance `add` to accept an `is_regex` flag
- [x] Update message handler to evaluate regex auto-responders using `regexp` package



- Implemented Phase 84 Auto-Responder Enhancement features: added migrations `071_auto_responder_update.up.sql` and `071_auto_responder_update.down.sql` with `is_regex` column to `auto_responders` table.
- Added DB operations `AddAutoResponder`, `ListAllAutoResponders`, and `ListAutoResponders` in `internal/db/db.go` to support `is_regex` and pre-compile regular expressions.
- Enhanced `/autoresponder add` command in `internal/bot/commands/autoresponder.go` to parse the `is_regex` flag.
- Implemented Phase 82 Command Cooldowns features: added migrations `070_command_cooldowns.up.sql` and `070_command_cooldowns.down.sql` with table `command_cooldowns`.
- Added DB operations `SetCommandCooldown` and `GetCommandCooldown` in `internal/db/db.go`.
- Updated `interactionCreateHandler` in `internal/bot/bot.go` to enforce command cooldowns.
- Added `/cooldown` command in `internal/bot/commands/cooldown.go` to manage custom command cooldowns.
- Updated `internal/bot/bot.go` to register the `cooldown` command.


### Phase 85 — Advanced Anti-Spam
- [x] `migrations/072_anti_spam.sql` — `anti_spam_config` table (guild_id, message_limit, time_window, mute_duration)
- [x] DB operations — `SetAntiSpamConfig`, `GetAntiSpamConfig`
- [x] `/antispam` command — `setup` to configure limit, time window, and mute duration
- [x] Update message handler to track message rate per user and auto-mute if exceeded

### Phase 86 — Advanced Logging System
- [x] `migrations/073_advanced_logging.sql` — `advanced_log_config` table (guild_id, events, channel_id)
- [x] DB operations — `SetAdvancedLogConfig`, `GetAdvancedLogConfig`
- [x] `/advancedlog` command — configure detailed event logging per channel
- [x] Implement enhanced event tracking and routing

### Phase 87 — Reaction Roles Logging
- [x] Update `messageReactionAddHandler` in `internal/bot/bot.go` to log role assignments if `advanced_log_config` enables role logging.
- [x] Update `messageReactionRemoveHandler` in `internal/bot/bot.go` to log role removals if `advanced_log_config` enables role logging.

 — Dynamic Voice Channels


- Implemented Phase 94 Role Commands: added `/role` command in `internal/bot/commands/role.go` with `add`, `remove`, and `info` subcommands to manage user roles directly using Discord API. Registered it in `internal/bot/bot.go`.
- Implemented Phase 95 Sticky Roles System: added migrations `080_sticky_roles.up.sql` and `080_sticky_roles.down.sql` with table `sticky_roles`.
- Added DB operations `SaveStickyRole`, `GetStickyRoles`, and `RemoveStickyRole` in `internal/db/db.go`.
- Added `/stickyrole` command in `internal/bot/commands/stickyrole.go` with `add`, `remove`, and `list` subcommands.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to restore sticky roles when a user leaves and rejoins a server. Registered the `stickyrole` command.


- Implemented Phase 96 Reaction Roles Menus features: added migrations `081_reaction_menus.up.sql` and `081_reaction_menus.down.sql` with tables `reaction_menus` and `reaction_menu_items`.
- Added DB operations `CreateReactionMenu`, `AddReactionMenuItem`, and `GetReactionMenuItems` in `internal/db/db.go`.
- Added `/reactionmenu` command with `create` and `add-role` subcommands in `internal/bot/commands/reactionmenu.go`. Registered it in `internal/bot/bot.go`.
- Updated `messageReactionAddHandler` and `messageReactionRemoveHandler` in `internal/bot/bot.go` to assign and remove roles based on configured menus.

### Phase 93 — Auto-Publish (Crosspost) Messages
- [x] `migrations/079_auto_publish.sql` — `auto_publish_config` table (guild_id, channel_id)
- [x] DB operations — `SetAutoPublishChannel`, `GetAutoPublishChannel`, `RemoveAutoPublishChannel`
- [x] `/autopublish` command with `add` and `remove` subcommands
- [x] Update `messageCreateHandler` to automatically crosspost (`ChannelMessageCrosspost`) messages in configured announcement channels

### Phase 96 — Reaction Roles Menus
- [x] `migrations/081_reaction_menus.sql` — `reaction_menus` table (message_id, guild_id, channel_id) and `reaction_menu_items` (message_id, emoji, role_id)
- [x] DB operations — `CreateReactionMenu`, `AddReactionMenuItem`, `GetReactionMenuItems`
- [x] `/reactionmenu` command — `create` and `add-role` subcommands
- [x] Update `messageReactionAddHandler` and `messageReactionRemoveHandler` to assign/remove roles based on `reaction_menu_items`

### Phase 97 — Welcome Messages System
- [x] `migrations/082_welcome_messages.sql` — `welcome_messages` table (guild_id, channel_id, message)
- [x] DB operations — `SetWelcomeMessage`, `GetWelcomeMessage`, `RemoveWelcomeMessage`
- [x] `/welcome` command with `set` and `remove` subcommands
- [x] Update `guildMemberAddHandler` to send the welcome message when a user joins

### Phase 98 — Goodbye Messages System
- [x] `migrations/083_goodbye_messages.sql` — `goodbye_messages` table (guild_id, channel_id, message)
- [x] DB operations — `SetGoodbyeMessage`, `GetGoodbyeMessage`, `RemoveGoodbyeMessage`
- [x] `/goodbye` command with `set` and `remove` subcommands
- [x] Update `guildMemberRemoveHandler` to send the goodbye message when a user leaves


### Phase 102 — Leveling Roles Blacklist System
- [x] `migrations/087_leveling_blacklist.sql` — `leveling_blacklist` table (id, guild_id, role_id)
- [x] DB operations — `AddLevelingBlacklist`, `RemoveLevelingBlacklist`, `IsRoleBlacklisted`
- [x] `/levelblacklist` command with `add`, `remove`, and `list` subcommands
- [x] Update `messageCreateHandler` to skip awarding XP if the user has a blacklisted role

### Phase 111 — Keyword Notifications
- [x] `migrations/096_keyword_notifications.up.sql` — `keyword_notifications` table (id, user_id, guild_id, keyword)
- [x] DB operations — `AddKeywordNotification`, `RemoveKeywordNotification`, `GetKeywordNotifications`
- [x] `/keyword` command with `add`, `remove`, and `list` subcommands
- [x] Update `messageCreateHandler` to DM users when their keyword is mentioned in the guild

### Completed Work

- Implemented Phase 111 Keyword Notifications: added migrations `096_keyword_notifications.up.sql` and `096_keyword_notifications.down.sql` with table `keyword_notifications`.
- Added DB operations `AddKeywordNotification`, `RemoveKeywordNotification`, and `GetKeywordNotifications` in `internal/db/db.go`.
- Added `/keyword` command with `add`, `remove`, and `list` subcommands in `internal/bot/commands/keyword.go`. Registered it in `internal/bot/bot.go`.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to send DM notifications to users when their keywords are mentioned.


- Implemented Phase 110 Auto-Delete Channels: added migrations `095_auto_delete.up.sql` and `095_auto_delete.down.sql` with table `auto_delete_config`.
- Added DB operations `SetAutoDelete`, `GetAutoDelete`, and `RemoveAutoDelete` in `internal/db/db.go`.
- Added `/autodelete` command with `setup` and `remove` subcommands in `internal/bot/commands/autodelete.go`. Registered it in `internal/bot/bot.go`.
- Updated `messageCreateHandler` in `internal/bot/bot.go` to automatically delete messages in configured channels after the specified delay.












- Implemented Phase 107 Thread Automation Config: added migrations `092_thread_automation.up.sql` and `092_thread_automation.down.sql` with table `thread_automation_config`.
- Added DB operations `SetThreadAutomation`, `GetThreadAutomation`, and `RemoveThreadAutomation` in `internal/db/db.go`.
- Added `/threadauto` command in `internal/bot/commands/threadauto.go` with `setup` and `remove` subcommands. Registered it in `internal/bot/bot.go`.
- Updated `threadCreateHandler` in `internal/bot/bot.go` to automatically join newly created threads in configured channels.


- Implemented Phase 106 Message Translation features: added migrations `091_translation_config.up.sql` and `091_translation_config.down.sql` with table `translation_config`.
- Added DB operations `SetTranslationConfig` and `GetTranslationConfig` in `internal/db/db.go`.
- Added `/translate` command with `text` and `set-default` subcommands in `internal/bot/commands/translate.go`.
- Updated `internal/bot/bot.go` to register the `translate` command.

- Implemented Phase 104 User Warn Level Automation: added migrations `089_warn_automation.up.sql` and `089_warn_automation.down.sql` with table `warn_automation_config`.
- Added DB operations `AddWarnAutomationRule`, `RemoveWarnAutomationRule`, and `GetWarnAutomationRules` in `internal/db/db.go`.
- Added `/warnautomod` command in `internal/bot/commands/warnautomod.go` with `add` and `remove` subcommands. Registered it in `internal/bot/bot.go`.
- Updated `/warn` command in `internal/bot/commands/warn.go` to evaluate user warning counts and automatically trigger mute, kick, or ban actions based on configured automation rules.

- Implemented Phase 103 Leveling Channel Blacklist System: added migrations `088_leveling_channel_blacklist.up.sql` and `088_leveling_channel_blacklist.down.sql` with table `leveling_channel_blacklist`.
- Added DB operations `AddLevelingChannelBlacklist`, `RemoveLevelingChannelBlacklist`, and `GetLevelingChannelBlacklists` in `internal/db/db.go`.
- Added `/levelchannelblacklist` command in `internal/bot/commands/levelchannelblacklist.go` with `add`, `remove`, and `list` subcommands. Registered it in `internal/bot/bot.go`.

- Implemented Phase 102 Leveling Roles Blacklist System: added migrations `087_leveling_blacklist.up.sql` and `087_leveling_blacklist.down.sql` with table `leveling_blacklist`.
- Added DB operations `AddLevelingBlacklist`, `RemoveLevelingBlacklist`, `IsRoleBlacklisted`, and `GetLevelingBlacklists` in `internal/db/db.go`.
- Added `/levelblacklist` command in `internal/bot/commands/levelblacklist.go` with `add`, `remove`, and `list` subcommands. Registered it in `internal/bot/bot.go`.

- Implemented Phase 101 Welcome DMs System: added migrations `086_welcome_dms.up.sql` and `086_welcome_dms.down.sql` with table `welcome_dm_config`.
- Added DB operations `SetWelcomeDM`, `GetWelcomeDM`, and `ToggleWelcomeDM` in `internal/db/db.go`.
- Added `/welcomedm` command in `internal/bot/commands/welcomedm.go` with `set`, `enable`, and `disable` subcommands.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to send the configured welcome DM when a user joins the server. Registered the `welcomedm` command.

- Implemented Phase 99 Member Count Channel System: added migrations `084_member_count.up.sql` and `084_member_count.down.sql` with table `member_count_config`.
- Added DB operations `SetMemberCountChannel`, `GetMemberCountChannel`, `RemoveMemberCountChannel`, and `GetAllMemberCountChannels` in `internal/db/db.go`.
- Added `/membercount` command with `setup` and `remove` subcommands in `internal/bot/commands/membercount.go`.
- Added a background goroutine `memberCountLoop` in `internal/bot/bot.go` to update configured member count channels periodically, respecting Discord's rate limits. Registered the `membercount` command.

- Implemented Phase 98 Goodbye Messages System: added migrations `083_goodbye_messages.up.sql` and `083_goodbye_messages.down.sql` with table `goodbye_messages`.
- Added DB operations `SetGoodbyeMessage`, `GetGoodbyeMessage`, and `RemoveGoodbyeMessage` in `internal/db/db.go`.
- Added `/goodbye` command in `internal/bot/commands/goodbye.go` to support `set` and `remove` subcommands.
- Updated `guildMemberRemoveHandler` in `internal/bot/bot.go` to send the goodbye message when a user leaves the server. Registered the `goodbye` command.


- Implemented Phase 97 Welcome Messages System: added migrations `082_welcome_messages.up.sql` and `082_welcome_messages.down.sql` with table `welcome_messages`.
- Added DB operations `SetWelcomeMessage`, `GetWelcomeMessage`, and `RemoveWelcomeMessage` in `internal/db/db.go`.
- Updated `/welcome` command in `internal/bot/commands/welcome.go` to support `set` and `remove` subcommands while preserving existing functionality.
- Updated `guildMemberAddHandler` in `internal/bot/bot.go` to send the welcome message when a user joins the server.


### Phase 99 — Member Count Channel System
- [x] `migrations/084_member_count.sql` — `member_count_config` table (guild_id, channel_id)
- [x] DB operations — `SetMemberCountChannel`, `GetMemberCountChannel`, `RemoveMemberCountChannel`
- [x] `/membercount` command with `setup` and `remove` subcommands
- [x] Update `guildMemberAddHandler` and `guildMemberRemoveHandler` to update the channel name (e.g. `Members: 123`)

### Phase 100 — Temporary Roles
- [x] `migrations/085_temp_roles.sql` — `temp_roles` table (id, guild_id, user_id, role_id, expires_at)
- [x] DB operations — `AddTempRole`, `GetExpiredTempRoles`, `RemoveTempRole`
- [x] `/temprole` command with `add` and `remove` subcommands
- [x] Background goroutine to periodically remove expired temporary roles

- Implemented Phase 100 Temporary Roles System: added migrations `085_temp_roles.up.sql` and `085_temp_roles.down.sql` with table `temp_roles`.
- Added DB operations `AddTempRole`, `GetExpiredTempRoles`, `RemoveTempRole`, and `RemoveTempRoleByGuildUserRole` in `internal/db/db.go`.
- Added `/temprole` command in `internal/bot/commands/temprole.go` with `add` and `remove` subcommands.
- Added a background goroutine `checkTempRoles` in `internal/bot/bot.go` to periodically remove expired temporary roles. Registered the `temprole` command.

### Phase 101 — Welcome DMs System
- [x] `migrations/086_welcome_dms.sql` — `welcome_dm_config` table (guild_id, message, is_enabled)
- [x] DB operations — `SetWelcomeDM`, `GetWelcomeDM`, `ToggleWelcomeDM`
- [x] `/welcomedm` command with `set`, `enable`, and `disable` subcommands
- [x] Update `guildMemberAddHandler` to send the configured DM to new users

### Phase 110 — Auto-Delete Channels
- [x] `migrations/095_auto_delete.up.sql` — `auto_delete_config` table (guild_id, channel_id, delete_after)
- [x] DB operations — `SetAutoDelete`, `GetAutoDelete`, `RemoveAutoDelete`
- [x] `/autodelete` command with `setup` and `remove` subcommands
- [x] Update `messageCreateHandler` to offload a goroutine that waits `delete_after` seconds and deletes the message in configured channels

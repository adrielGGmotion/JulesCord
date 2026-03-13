# JulesCord Features

JulesCord is a feature-rich, scalable Discord bot with an integrated dashboard. Below is an overview of its major features.

## Leveling & Economy System

The bot implements an active XP and coin reward system:
- **XP on Message:** Users earn XP by sending messages in text channels. To prevent spamming, a strict **1-minute cooldown per user per channel** is enforced before they can receive XP again.
- **Level Ups:** When enough XP is accumulated, users will level up, and the bot announces their new level in the channel.
- **Economy:** Users can claim a daily coin reward via the `/daily` command (24-hour cooldown). Their balances can be checked with `/coins`.

## Moderation System

A comprehensive suite for managing communities:
- Standard moderation actions such as `/warn`, `/kick`, `/ban`, and `/purge` are provided.
- **Mod Log Channel:** If configured, all moderation actions are posted as detailed embeds in a designated logging channel. This feature ensures total visibility for server administrators.
- **Database Tracking:** Every moderation action (including warnings) is stored in the database, allowing moderators to look up a user's past infractions with `/warnings`.

## Guild Configuration

Servers can highly customize their experience:
- Features like the mod log channel, welcome message channels, and auto-assigned roles are persistent and stored per-guild.
- **Auto-Role on Join:** When a new member joins the server, the bot can automatically grant them a predefined role if configured via `/config set-auto-role`.
- **Welcome Messages:** Automated welcome greetings are sent when new members join the server.

## Reaction Roles

Server administrators can create message-based role assignments:
- Use `/reactionrole add` to bind an emoji to a role on a specific message.
- When users add the reaction to the message, the role is granted. If the reaction is removed, the role is automatically removed.

## Scheduled Announcements

- Using the `/schedule` command, announcements can be queued for future delivery.
- A background goroutine continuously polls the database and dispatches any due announcements automatically.

## Web Dashboard

A complete frontend React application (Vite + Tailwind CSS):
- View overall bot statistics, total guilds, and total users.
- Connects to the backend API (`/ws`) via WebSocket for real-time status updates (pushing data every 5 seconds).
- Includes dynamic charts (via `recharts`) showing metrics like command usage popularity.
- Moderators can browse comprehensive tables of logs, user XP leaderboards, and more.

## Observability

The bot incorporates enterprise-grade metrics for monitoring its performance:
- **Structured Logging:** Uses standard library `log/slog` for structured JSON logs across the application.
- **Prometheus Metrics:** Tracks command execution counts, command latency histograms, and database query durations. These are exposed via the `/metrics` endpoint and visualized on the web dashboard.

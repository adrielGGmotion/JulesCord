# JulesCord Commands

This document lists all the available Discord slash commands for JulesCord. Commands are categorized by their functionality.

## Core Commands

| Command | Description |
|---|---|
| `/about` | Describes JulesCord and its autonomous build loop. |
| `/stats` | Displays bot statistics including guild count, user count, uptime, and total commands run. |
| `/help` | Dynamically lists all registered commands with their descriptions. |
| `/ping` | Returns the current bot latency (API and WebSocket ping). |
| `/changelog` | Reads recent git commits from the GitHub API and summarizes changes. |

## Moderation

| Command | Description | Permission |
|---|---|---|
| `/warn @user reason` | Issues a warning to a user. Logs the action in the database and the moderation log channel (if configured). | Kick Members |
| `/warnings @user` | Lists all active warnings for a specified user. | None |
| `/kick @user reason` | Kicks a user from the server with an optional reason. Logs the action in the moderation log channel. | Kick Members |
| `/ban @user reason` | Bans a user from the server with an optional reason. Logs the action in the moderation log channel. | Ban Members |
| `/purge count` | Bulk deletes a specified number of messages (between 2 and 100) from the current channel. | Manage Messages |

## Leveling & Economy

| Command | Description |
|---|---|
| `/rank` | Displays a user's current XP, level, and server rank based on XP accumulation. |
| `/leaderboard` | Displays the top 10 users in the server ranked by XP. |
| `/daily` | Claims a daily coin reward. Can only be used once every 24 hours per user. |
| `/coins` | Displays the current coin balance of a user. |

## Guild Configuration

| Command | Description | Permission |
|---|---|---|
| `/config view` | Displays the current server configuration settings. | Administrator |
| `/config set-log-channel #channel` | Sets the designated channel for moderation action logging. | Administrator |
| `/config set-welcome-channel #channel` | Sets the designated channel for welcome messages when users join. | Administrator |
| `/config set-mod-role @role` | Designates a role as a moderator role (can bypass certain moderation checks or be granted specific permissions). | Administrator |
| `/config set-auto-role @role` | Sets a role to be automatically assigned to all new members upon joining the server. | Administrator |

## Advanced Features

| Command | Description | Permission |
|---|---|---|
| `/reactionrole add message_id emoji @role` | Creates a new reaction role listener. When a user reacts with the specified emoji on the specified message, they are granted the role. | Administrator |
| `/reactionrole remove message_id emoji` | Removes an existing reaction role configuration for the specified message and emoji. | Administrator |
| `/schedule add time message` | Schedules a future announcement to be sent in the current channel at the specified time. | Administrator |

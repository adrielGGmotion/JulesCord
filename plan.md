1. **Create Migrations**:
   - Create `migrations/055_auto_threads.up.sql` to create the `auto_threads_config` table (`guild_id`, `channel_id`, `thread_name_template`).
   - Create `migrations/055_auto_threads.down.sql` to drop the table.
2. **Add Database Methods in `internal/db/db.go`**:
   - `SetAutoThread(ctx, guildID, channelID, template)`
   - `GetAutoThread(ctx, guildID, channelID)`
   - `RemoveAutoThread(ctx, guildID, channelID)`
3. **Add `/autothread` Slash Command**:
   - Create `internal/bot/commands/autothread.go`.
   - Implement `/autothread setup` (admin only).
   - Implement `/autothread remove` (admin only).
4. **Update Message Handler in `internal/bot/bot.go`**:
   - In `messageCreateHandler`, check if the message is in an auto-thread channel.
   - If so, use `s.MessageThreadStartComplex` to create a thread on the message using the template string.
5. **Register Command**:
   - Add `commands.AutoThread(database)` to the registry in `internal/bot/bot.go`.
6. **Pre-commit Checks**:
   - Run the pre-commit steps to ensure testing, compilation, and linting.

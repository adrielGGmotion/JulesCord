1. **Database Migrations**
   - Create `migrations/070_command_cooldowns.up.sql` to define the `command_cooldowns` table with `user_id`, `command`, and `expires_at`. The primary key should be `(user_id, command)`.
   - Create `migrations/070_command_cooldowns.down.sql` to drop the table.
   - Verify the migration files with `ls -l migrations/070_*` or `cat`.

2. **Database Methods**
   - After confirming with `grep` in `internal/db/db.go`, implement `SetCommandCooldown(ctx context.Context, userID string, command string, duration time.Duration) error` in `internal/db/db.go` and its interface. This method will use an upsert (`INSERT ... ON CONFLICT (user_id, command) DO UPDATE SET expires_at = EXCLUDED.expires_at`).
   - Implement `GetCommandCooldown(ctx context.Context, userID string, command string) (time.Time, error)` in `internal/db/db.go` and its interface. This method will fetch the `expires_at` column.
   - Use `git diff` to verify changes in `internal/db/db.go`.

3. **Interaction Handler Update**
   - Locate the `interactionCreateHandler` in `internal/bot/bot.go` using `grep`.
   - Update `internal/bot/bot.go` inside the `if b.DB != nil && i.Type == discordgo.InteractionApplicationCommand {` block.
   - Before `b.Registry.Dispatch(s, i)`, check the database using `GetCommandCooldown` for `commandName`.
   - If the user is on cooldown (i.e., `expires_at > time.Now()`), send an ephemeral message stating the cooldown duration remaining and return early. Only return early for `InteractionApplicationCommand` commands, don't return entirely from the function if you don't dispatch, or just `return` if you send the response since dispatch won't happen. Wait, `b.Registry.Dispatch(s, i)` handles dispatching for both ApplicationCommands and other interactions (maybe Message Components if registered, though they usually aren't via `Registry`). But `Dispatch` is for all types. Actually, `Registry.Dispatch` handles `InteractionApplicationCommand`. So if on cooldown, send the response and `return`.
   - Use `git diff` to verify the modifications to `internal/bot/bot.go`.

4. **Add `/cooldown` Command**
   - Locate the registry block in `internal/bot/bot.go`.
   - Create `internal/bot/commands/cooldown.go`.
   - Implement the `/cooldown` slash command with the name `cooldown` and require `ManageServer` permissions.
   - It should accept `user` (User), `command` (String), and `duration` (String) as options.
   - The command will parse the duration and call `SetCommandCooldown` to set a custom cooldown for the user and command.
   - Update `internal/bot/bot.go` to register the new command via `registry.Add(commands.Cooldown(database))`.
   - Use `git diff` to verify the changes to `internal/bot/bot.go` and `cat` for `internal/bot/commands/cooldown.go`.

5. **Compile, Test and Verify**
   - Ensure the bot code compiles and tests pass using `go test ./...`.
   - Start the bot using the confirmed entry point (`go run cmd/bot/main.go`).
   - Test setting a cooldown with `/cooldown`.
   - Test that the cooldown prevents the user from using the command, and works successfully once the cooldown passes.

6. **Complete pre commit steps**
   - Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.

7. **Submit**
   - Submit the changes.

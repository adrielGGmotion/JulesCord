1. **Migration File Creation**
   - Create `migrations/041_level_backgrounds.up.sql`:
     ```sql
     ALTER TABLE user_economy ADD COLUMN background_url TEXT;
     ```
   - Create `migrations/041_level_backgrounds.down.sql`:
     ```sql
     ALTER TABLE user_economy DROP COLUMN background_url;
     ```

2. **Database Updates in `internal/db/db.go`**
   - Update `UserEconomy` struct to include `BackgroundURL *string`.
   - Update `GetUserEconomy` to SELECT and scan `background_url`. Note: Need to be careful about `SELECT guild_id, user_id, xp, level, coins, last_daily_at` -> `SELECT guild_id, user_id, xp, level, coins, last_daily_at, background_url`.
   - Add a new DB method `SetBackgroundURL(ctx context.Context, guildID, userID, backgroundURL string) error`.

3. **Command Updates in `internal/bot/commands/rank.go`**
   - Convert `rank` from a simple command into a command with subcommands:
     - `view`: (Default) The existing logic to check user rank, but updated to use `econ.BackgroundURL` in the embed's `Image` field if it's set.
     - `set-background`: Takes a `url` parameter and calls `SetBackgroundURL` to set the user's background.
     - `role-rewards`: Fetches `GetLevelRoles` and sends an embed listing all role rewards for the server.

   Note: I need to ensure subcommands work cleanly. A common pattern in this bot for converting top-level commands to subcommand structure is creating the `SubCommand` type and switching on `i.ApplicationCommandData().Options[0].Name`. Wait, the existing `/rank` command has an optional `user` argument.
   If I change it to subcommands:
   `/rank view [user]`
   `/rank set-background <url>`
   `/rank role-rewards`
   This is a breaking change for the top-level `/rank [user]` behavior because Discord doesn't allow mixing subcommands with base-level arguments.
   Let's check if there are examples of this. In many cases, it's better to make the base command have subcommands. I will implement `view`, `set-background`, and `role-rewards` as subcommands.

4. **Verify changes and run Pre-commit Steps**
   - Call `pre_commit_instructions` and follow testing procedures.

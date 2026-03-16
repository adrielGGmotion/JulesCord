1. **Create Migrations (`migrations/059_user_roles.up.sql` and `migrations/059_user_roles.down.sql`)**
   - Create a `user_roles` table to store an array of `role_ids` mapped to `user_id` and `guild_id`.
   - Setup constraints and indexing (e.g. `PRIMARY KEY (user_id, guild_id)`).

2. **Add Database Operations (`internal/db/db.go`)**
   - Add `SaveUserRoles(ctx context.Context, userID, guildID string, roleIDs []string) error` to insert or update the saved roles.
   - Add `GetUserRoles(ctx context.Context, userID, guildID string) ([]string, error)` to retrieve the stored role IDs.

3. **Add `guildMemberRemoveHandler` (`internal/bot/bot.go`)**
   - Add `bot.guildMemberRemoveHandler` that triggers when a user leaves a server.
   - Extract the user's current roles (`m.Roles`).
   - If the user had roles, call `b.DB.SaveUserRoles` to persist them.
   - Register the handler in `bot.go` (`b.Session.AddHandler(b.guildMemberRemoveHandler)`).

4. **Update `guildMemberAddHandler` (`internal/bot/bot.go`)**
   - Fetch previously saved roles using `b.DB.GetUserRoles`.
   - If roles are found, iterate through them and reassign them using `s.GuildMemberRoleAdd`.

5. **Complete pre commit steps**
   - Complete pre commit steps to make sure proper testing, verifications, reviews and reflections are done.

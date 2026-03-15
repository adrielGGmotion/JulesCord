1. Modify `internal/db/db.go`
   - Add `GetAllPendingRemindersForGuild(ctx context.Context, guildID string) ([]Reminder, error)`
   - Add `DeleteAllRemindersForUser(ctx context.Context, userID string) (int64, error)`
   - Add `SnoozeReminder(ctx context.Context, id int, duration time.Duration) error`
2. Modify `internal/bot/commands/remind.go`
   - Add `/remind list-all` subcommand (Admin only - default permission set to admin)
   - Add `/remind delete-all` subcommand
   - Implement `handleListAllReminders` and `handleDeleteAllReminders`
3. Modify `internal/bot/bot.go`
   - Update `checkReminders()` to send message with an `ActionsRow` and a `Button` component: custom ID `snooze_{id}`
   - In `interactionCreateHandler`, check for `strings.HasPrefix(customID, "snooze_")`. Parse ID, snooze for a default duration (e.g., 10 minutes), and update the DB. Remove the button/update the message to say "Snoozed for 10 minutes".
4. Compile and test
   - `go build ./cmd/bot/`
   - Run the bot and manually test or just observe no errors.
5. Update `AGENTS.md` and commit

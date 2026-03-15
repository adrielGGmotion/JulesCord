1. **Create Migration:** Create `migrations/044_reaction_roles_update.up.sql` and `migrations/044_reaction_roles_update.down.sql` to alter the `reaction_roles` table. I will add `emoji_id` and `is_custom` columns. Wait, looking at the existing code, it seems `reaction_roles` table stores the `emoji` as `name:id` format when it's a custom emoji, since in `messageReactionAddHandler`:
   ```go
		emojiName := r.Emoji.Name
		if r.Emoji.ID != "" {
			emojiName = fmt.Sprintf("%s:%s", r.Emoji.Name, r.Emoji.ID)
		}
   ```
   So what exactly does the prompt ask?
   "- [ ] `migrations/044_reaction_roles_update.sql` — add `emoji_id` and `is_custom` columns to support custom emojis"
   "- [ ] DB operations — update `AddReactionRole`, `RemoveReactionRole`, and `GetReactionRoles` to handle custom emojis"
   "- [ ] `/reactionrole` command — update to parse and store custom emojis"
   "- [ ] Update `MessageReactionAdd` / `MessageReactionRemove` handlers to check for custom emojis"

   Ah, maybe it didn't support custom emojis perfectly, or we need to explicitly track `emoji_id` and `is_custom` for better querying. Let me check the structure.

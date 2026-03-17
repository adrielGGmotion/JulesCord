import re

with open("AGENTS.md", "r") as f:
    content = f.read()

# We need to append Phase 80 since Phase 79 is done
tasks = """
### Phase 80 — Moderation Unban and Clear Warnings
- [ ] DB operations — `RemoveWarning`, `ClearWarnings`
- [ ] `/unban` command to unban a user by their user ID
- [ ] `/clearwarnings` command to clear a user's warning history
- [ ] Update `internal/bot/bot.go` to register the `unban` and `clearwarnings` commands
"""

if "Phase 80" not in content:
    with open("AGENTS.md", "a") as f:
        f.write(tasks)

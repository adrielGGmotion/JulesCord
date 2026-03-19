import re

with open('AGENTS.md', 'r') as f:
    content = f.read()

# Check off Phase 128
content = content.replace(
    "- [ ] `migrations/113_economy_lotteries.up.sql` — `lotteries` table (id, guild_id, prize, ticket_price, end_time) and `lottery_tickets` table (lottery_id, user_id, count)",
    "- [x] `migrations/113_economy_lotteries.up.sql` — `lotteries` table (id, guild_id, prize, ticket_price, end_time) and `lottery_tickets` table (lottery_id, user_id, count)"
)
content = content.replace(
    "- [ ] DB operations — `CreateLottery`, `BuyLotteryTicket`, `GetActiveLotteries`, `ResolveLottery`",
    "- [x] DB operations — `CreateLottery`, `BuyLotteryTicket`, `GetActiveLotteries`, `ResolveLottery`"
)
content = content.replace(
    "- [ ] `/lottery` command with `create`, `buy`, and `list` subcommands",
    "- [x] `/lottery` command with `create`, `buy`, and `list` subcommands"
)
content = content.replace(
    "- [ ] Add a background goroutine `lotteryLoop` in `internal/bot/bot.go` to resolve ended lotteries and award coins to a random ticket holder",
    "- [x] Add a background goroutine `lotteryLoop` in `internal/bot/bot.go` to resolve ended lotteries and award coins to a random ticket holder"
)

# Add to completed work
completed_work_entry = """- Implemented Phase 128 Economy Lotteries: added migrations `113_economy_lotteries.up.sql` and `113_economy_lotteries.down.sql` with tables `lotteries` and `lottery_tickets`.
- Added DB operations `CreateLottery`, `BuyLotteryTicket`, `GetActiveLotteries`, and `ResolveLottery` in `internal/db/db.go`.
- Added `/lottery` command in `internal/bot/commands/lottery.go` with `create`, `buy`, and `list` subcommands.
- Added a background goroutine `lotteryLoop` in `internal/bot/bot.go` to resolve ended lotteries and award coins to a random ticket holder, and registered the `lottery` command.

"""

content = re.sub(r'(## Completed Work\n)', r'\1' + completed_work_entry, content)

with open('AGENTS.md', 'w') as f:
    f.write(content)

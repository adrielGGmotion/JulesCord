CREATE TABLE IF NOT EXISTS lotteries (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(20) NOT NULL,
    prize INTEGER NOT NULL DEFAULT 0,
    ticket_price INTEGER NOT NULL DEFAULT 0,
    end_time TIMESTAMP NOT NULL,
    UNIQUE(id, guild_id)
);

CREATE TABLE IF NOT EXISTS lottery_tickets (
    lottery_id INTEGER NOT NULL REFERENCES lotteries(id) ON DELETE CASCADE,
    user_id VARCHAR(20) NOT NULL,
    count INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY(lottery_id, user_id)
);

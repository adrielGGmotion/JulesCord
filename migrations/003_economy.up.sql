CREATE TABLE IF NOT EXISTS user_economy (
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    xp BIGINT NOT NULL DEFAULT 0,
    level INT NOT NULL DEFAULT 0,
    coins BIGINT NOT NULL DEFAULT 0,
    last_daily_at TIMESTAMPTZ,
    PRIMARY KEY (guild_id, user_id)
);

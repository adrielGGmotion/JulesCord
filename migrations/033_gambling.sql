CREATE TABLE IF NOT EXISTS gambling_stats (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    coins_won BIGINT NOT NULL DEFAULT 0,
    coins_lost BIGINT NOT NULL DEFAULT 0,
    games_played INT NOT NULL DEFAULT 0,
    games_won INT NOT NULL DEFAULT 0,
    games_lost INT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id)
);

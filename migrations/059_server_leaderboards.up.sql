CREATE TABLE IF NOT EXISTS server_leaderboards (
    guild_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    points BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id)
);

CREATE TABLE IF NOT EXISTS voice_time_stats (
    guild_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    total_seconds BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY(guild_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_voice_time_stats_guild_total ON voice_time_stats(guild_id, total_seconds DESC);

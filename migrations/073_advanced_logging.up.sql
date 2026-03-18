CREATE TABLE IF NOT EXISTS advanced_log_config (
    guild_id VARCHAR(255) PRIMARY KEY,
    events TEXT NOT NULL,
    channel_id VARCHAR(255) NOT NULL
);
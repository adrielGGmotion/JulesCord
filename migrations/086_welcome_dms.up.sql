CREATE TABLE IF NOT EXISTS welcome_dm_config (
    guild_id VARCHAR(255) PRIMARY KEY,
    message TEXT NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS voice_generator_config (
    guild_id VARCHAR(255) PRIMARY KEY,
    base_channel_id VARCHAR(255) NOT NULL,
    max_channels INT NOT NULL DEFAULT 5
);

CREATE TABLE IF NOT EXISTS forwarding_config (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    source_channel_id VARCHAR(255) NOT NULL,
    target_channel_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(guild_id, source_channel_id, target_channel_id)
);
CREATE INDEX idx_forwarding_config_guild_source ON forwarding_config(guild_id, source_channel_id);

CREATE TABLE IF NOT EXISTS leveling_channel_blacklist (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (guild_id, channel_id)
);

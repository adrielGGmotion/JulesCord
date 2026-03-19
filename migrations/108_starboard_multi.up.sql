CREATE TABLE IF NOT EXISTS starboard_multi_config (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    emoji VARCHAR(255) NOT NULL,
    threshold INTEGER NOT NULL DEFAULT 3,
    UNIQUE(guild_id, channel_id, emoji)
);

CREATE TABLE IF NOT EXISTS starboard_multi_messages (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    original_message_id VARCHAR(255) NOT NULL,
    starboard_id INTEGER REFERENCES starboard_multi_config(id) ON DELETE CASCADE,
    starboard_message_id VARCHAR(255),
    stars INTEGER NOT NULL DEFAULT 0,
    UNIQUE(guild_id, original_message_id, starboard_id)
);

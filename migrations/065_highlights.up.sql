CREATE TABLE IF NOT EXISTS highlights (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) NOT NULL,
    added_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(guild_id, message_id)
);

CREATE TABLE IF NOT EXISTS thread_watchers (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    UNIQUE(guild_id, channel_id, user_id)
);

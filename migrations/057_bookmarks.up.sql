CREATE TABLE IF NOT EXISTS bookmarks (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    note TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, message_id)
);

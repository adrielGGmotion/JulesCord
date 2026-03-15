CREATE TABLE IF NOT EXISTS counting_config (
    guild_id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    current_number INTEGER NOT NULL DEFAULT 0,
    last_user_id TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

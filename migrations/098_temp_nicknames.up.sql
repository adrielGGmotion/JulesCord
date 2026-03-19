CREATE TABLE IF NOT EXISTS temp_nicknames (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    original_nickname TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(guild_id, user_id)
);

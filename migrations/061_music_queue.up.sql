CREATE TABLE IF NOT EXISTS music_queue (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    query TEXT NOT NULL,
    position INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_music_queue_guild_id ON music_queue(guild_id);

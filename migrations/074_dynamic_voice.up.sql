CREATE TABLE IF NOT EXISTS dynamic_voice_config (
    guild_id TEXT PRIMARY KEY REFERENCES guilds(id) ON DELETE CASCADE,
    category_id TEXT NOT NULL,
    trigger_channel_id TEXT NOT NULL
);

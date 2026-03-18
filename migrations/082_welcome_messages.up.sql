CREATE TABLE IF NOT EXISTS welcome_messages (
    guild_id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    message TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS server_log_config (
    guild_id TEXT PRIMARY KEY REFERENCES guilds(id),
    channel_id TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

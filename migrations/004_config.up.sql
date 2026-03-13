CREATE TABLE IF NOT EXISTS guild_config (
    guild_id TEXT PRIMARY KEY REFERENCES guilds(id),
    mod_log_channel_id TEXT,
    welcome_channel_id TEXT,
    mod_role_id TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

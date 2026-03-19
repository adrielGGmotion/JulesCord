CREATE TABLE IF NOT EXISTS translation_config (
    guild_id TEXT PRIMARY KEY REFERENCES guilds(id),
    default_language TEXT NOT NULL
);

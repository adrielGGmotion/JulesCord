CREATE TABLE IF NOT EXISTS thread_config (
    guild_id TEXT PRIMARY KEY,
    auto_archive_duration INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS thread_automation_config (
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    auto_join BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (guild_id, channel_id)
);

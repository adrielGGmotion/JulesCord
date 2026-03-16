CREATE TABLE IF NOT EXISTS auto_threads_config (
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    thread_name_template TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (guild_id, channel_id)
);

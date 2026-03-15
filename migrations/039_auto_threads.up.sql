CREATE TABLE IF NOT EXISTS auto_thread_config (
    channel_id VARCHAR(255) PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    thread_name_template VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_auto_thread_guild_id ON auto_thread_config(guild_id);

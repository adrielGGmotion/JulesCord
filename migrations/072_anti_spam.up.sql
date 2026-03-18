CREATE TABLE IF NOT EXISTS anti_spam_config (
    guild_id VARCHAR(255) PRIMARY KEY,
    message_limit INT NOT NULL DEFAULT 5,
    time_window INT NOT NULL DEFAULT 5,
    mute_duration VARCHAR(255) NOT NULL DEFAULT '10m',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

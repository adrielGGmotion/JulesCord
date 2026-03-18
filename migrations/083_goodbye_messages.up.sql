CREATE TABLE IF NOT EXISTS goodbye_messages (
    guild_id VARCHAR(255) PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    message TEXT NOT NULL
);

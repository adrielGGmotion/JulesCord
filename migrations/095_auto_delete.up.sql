CREATE TABLE IF NOT EXISTS auto_delete_config (
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    delete_after INTEGER NOT NULL,
    PRIMARY KEY (guild_id, channel_id)
);

CREATE TABLE IF NOT EXISTS autothread_config (
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    thread_name_template VARCHAR(255) DEFAULT '',
    PRIMARY KEY (guild_id, channel_id)
);

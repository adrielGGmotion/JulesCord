CREATE TABLE IF NOT EXISTS voice_roles (
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    role_id VARCHAR(255) NOT NULL,
    PRIMARY KEY (guild_id, channel_id)
);

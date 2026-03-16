CREATE TABLE IF NOT EXISTS voice_xp (
    user_id VARCHAR(255) NOT NULL,
    guild_id VARCHAR(255) NOT NULL,
    join_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (user_id, guild_id)
);

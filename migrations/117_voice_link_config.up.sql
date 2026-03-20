CREATE TABLE IF NOT EXISTS voice_link_config (
    guild_id TEXT NOT NULL,
    voice_channel_id TEXT NOT NULL,
    text_channel_id TEXT NOT NULL,
    PRIMARY KEY (guild_id, voice_channel_id)
);

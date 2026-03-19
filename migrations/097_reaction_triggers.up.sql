CREATE TABLE IF NOT EXISTS reaction_triggers (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    keyword TEXT NOT NULL,
    emoji TEXT NOT NULL,
    UNIQUE(guild_id, keyword)
);

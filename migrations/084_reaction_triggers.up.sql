CREATE TABLE IF NOT EXISTS reaction_triggers (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    trigger_word TEXT NOT NULL,
    emoji TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_reaction_triggers_guild_id ON reaction_triggers(guild_id);

CREATE TABLE IF NOT EXISTS marriages (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    user1_id TEXT NOT NULL,
    user2_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending' or 'accepted'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Ensure a user can only be in one marriage per guild
CREATE UNIQUE INDEX idx_marriages_guild_user1 ON marriages(guild_id, user1_id);
CREATE UNIQUE INDEX idx_marriages_guild_user2 ON marriages(guild_id, user2_id);

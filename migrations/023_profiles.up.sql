CREATE TABLE IF NOT EXISTS user_profiles (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    bio TEXT,
    color TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (guild_id, user_id),
    FOREIGN KEY (guild_id) REFERENCES guilds(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

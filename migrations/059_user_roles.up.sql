CREATE TABLE IF NOT EXISTS user_roles (
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    role_ids TEXT[] NOT NULL,
    PRIMARY KEY (user_id, guild_id)
);

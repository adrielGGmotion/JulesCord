CREATE TABLE IF NOT EXISTS sticky_roles (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    PRIMARY KEY (guild_id, user_id, role_id)
);

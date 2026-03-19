CREATE TABLE IF NOT EXISTS leveling_blacklist (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    role_id VARCHAR(255) NOT NULL,
    UNIQUE (guild_id, role_id)
);

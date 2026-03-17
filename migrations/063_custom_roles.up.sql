CREATE TABLE IF NOT EXISTS custom_roles (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    role_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    color INTEGER NOT NULL,
    icon_url VARCHAR(2048) DEFAULT '',
    UNIQUE(guild_id, user_id)
);

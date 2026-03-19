CREATE TABLE IF NOT EXISTS message_snippets (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    UNIQUE(guild_id, name)
);

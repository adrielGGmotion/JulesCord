CREATE TABLE IF NOT EXISTS shop_items (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    name TEXT NOT NULL,
    description TEXT,
    price BIGINT NOT NULL,
    role_id TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(guild_id, name)
);

CREATE TABLE IF NOT EXISTS user_inventory (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    item_id INT NOT NULL REFERENCES shop_items(id) ON DELETE CASCADE,
    acquired_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

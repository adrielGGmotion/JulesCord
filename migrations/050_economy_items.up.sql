CREATE TABLE IF NOT EXISTS user_items (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    item_id INT NOT NULL REFERENCES shop_items(id) ON DELETE CASCADE,
    quantity INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(guild_id, user_id, item_id)
);

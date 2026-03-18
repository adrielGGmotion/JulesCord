CREATE TABLE IF NOT EXISTS reaction_menus (
    message_id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reaction_menu_items (
    message_id TEXT NOT NULL,
    emoji TEXT NOT NULL,
    role_id TEXT NOT NULL,
    PRIMARY KEY (message_id, emoji),
    FOREIGN KEY (message_id) REFERENCES reaction_menus(message_id) ON DELETE CASCADE
);

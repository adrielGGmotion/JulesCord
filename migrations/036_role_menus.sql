CREATE TABLE IF NOT EXISTS role_menus (
    message_id TEXT PRIMARY KEY,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_menu_options (
    message_id TEXT REFERENCES role_menus(message_id) ON DELETE CASCADE,
    role_id TEXT NOT NULL,
    emoji TEXT NOT NULL,
    label TEXT NOT NULL,
    description TEXT,
    PRIMARY KEY (message_id, role_id)
);

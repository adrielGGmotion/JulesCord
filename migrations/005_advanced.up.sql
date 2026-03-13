CREATE TABLE IF NOT EXISTS reaction_roles (
    message_id VARCHAR(255) NOT NULL,
    emoji VARCHAR(255) NOT NULL,
    role_id VARCHAR(255) NOT NULL,
    PRIMARY KEY (message_id, emoji)
);

CREATE TABLE IF NOT EXISTS scheduled_announcements (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    send_at TIMESTAMP WITH TIME ZONE NOT NULL,
    sent BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE guild_config ADD COLUMN IF NOT EXISTS auto_role_id VARCHAR(255);

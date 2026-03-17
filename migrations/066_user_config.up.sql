CREATE TABLE IF NOT EXISTS user_config (
    user_id VARCHAR(255) PRIMARY KEY,
    dnd_mode BOOLEAN DEFAULT FALSE,
    dm_notifications BOOLEAN DEFAULT TRUE
);

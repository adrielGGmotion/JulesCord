CREATE TABLE IF NOT EXISTS warn_automation_config (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    warning_threshold INT NOT NULL,
    action TEXT NOT NULL, -- 'mute', 'kick', 'ban'
    duration TEXT, -- Only applicable if action is 'mute' (e.g. '10m', '1h') or temporary ban ('7d')
    UNIQUE(guild_id, warning_threshold)
);

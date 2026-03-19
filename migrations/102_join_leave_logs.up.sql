CREATE TABLE IF NOT EXISTS join_leave_log_config (
    guild_id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    log_joins BOOLEAN DEFAULT TRUE,
    log_leaves BOOLEAN DEFAULT TRUE
);

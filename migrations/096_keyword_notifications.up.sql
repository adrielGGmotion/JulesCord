CREATE TABLE IF NOT EXISTS keyword_notifications (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    keyword TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_keyword_notif_guild ON keyword_notifications(guild_id);

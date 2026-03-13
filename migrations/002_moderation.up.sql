CREATE TABLE IF NOT EXISTS warnings (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    moderator_id TEXT NOT NULL REFERENCES users(id),
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS mod_actions (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    moderator_id TEXT NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

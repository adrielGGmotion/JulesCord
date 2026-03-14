CREATE TABLE IF NOT EXISTS reputation (
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    rep BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id)
);

CREATE TABLE IF NOT EXISTS reputation_log (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds(id),
    sender_id TEXT NOT NULL REFERENCES users(id),
    receiver_id TEXT NOT NULL REFERENCES users(id),
    given_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS guilds (
    id TEXT PRIMARY KEY,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    global_name TEXT,
    avatar_url TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS command_log (
    id SERIAL PRIMARY KEY,
    command_name TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id),
    guild_id TEXT REFERENCES guilds(id),
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

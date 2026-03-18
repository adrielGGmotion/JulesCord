CREATE TABLE IF NOT EXISTS temp_roles (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    role_id VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE(guild_id, user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_temp_roles_expires_at ON temp_roles(expires_at);

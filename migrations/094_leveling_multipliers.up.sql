CREATE TABLE IF NOT EXISTS leveling_multipliers (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    role_id VARCHAR(255) NOT NULL,
    multiplier NUMERIC(5,2) NOT NULL DEFAULT 1.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(guild_id, role_id)
);

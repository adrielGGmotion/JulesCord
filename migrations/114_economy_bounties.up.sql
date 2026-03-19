CREATE TABLE IF NOT EXISTS bounties (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    target_user_id VARCHAR(255) NOT NULL,
    bounty_amount BIGINT NOT NULL,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (guild_id, target_user_id)
);

CREATE INDEX IF NOT EXISTS idx_bounties_guild_id ON bounties(guild_id);

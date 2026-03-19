CREATE TABLE IF NOT EXISTS coinflip_bets (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    host_id VARCHAR(255) NOT NULL,
    opponent_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    side VARCHAR(10) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_coinflip_bets_guild ON coinflip_bets(guild_id);

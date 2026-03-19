CREATE TABLE IF NOT EXISTS trades (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    sender_id VARCHAR(255) NOT NULL,
    receiver_id VARCHAR(255) NOT NULL,
    sender_amount INT NOT NULL,
    receiver_amount INT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trades_guild_status ON trades(guild_id, status);

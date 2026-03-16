CREATE TABLE IF NOT EXISTS transfers (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    sender_id VARCHAR(255) NOT NULL,
    receiver_id VARCHAR(255) NOT NULL,
    amount BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transfers_sender ON transfers(guild_id, sender_id);
CREATE INDEX IF NOT EXISTS idx_transfers_receiver ON transfers(guild_id, receiver_id);

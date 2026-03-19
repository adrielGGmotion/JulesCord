CREATE TABLE IF NOT EXISTS heists (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    target_user_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'planning',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS heist_participants (
    heist_id INTEGER REFERENCES heists(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    PRIMARY KEY (heist_id, user_id)
);

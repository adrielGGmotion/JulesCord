CREATE TABLE IF NOT EXISTS polls (
    id VARCHAR(255) PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL,
    question TEXT NOT NULL,
    options_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_polls_message_id ON polls(message_id);

CREATE TABLE IF NOT EXISTS poll_votes (
    poll_id VARCHAR(255) REFERENCES polls(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    option_index INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (poll_id, user_id)
);

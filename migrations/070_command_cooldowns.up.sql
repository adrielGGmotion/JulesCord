CREATE TABLE IF NOT EXISTS command_cooldowns (
    user_id VARCHAR(255) NOT NULL,
    command VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (user_id, command)
);

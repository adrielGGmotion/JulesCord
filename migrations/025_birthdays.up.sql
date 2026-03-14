CREATE TABLE IF NOT EXISTS birthday_config (
    guild_id VARCHAR(255) PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS birthdays (
    guild_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    birth_month INT NOT NULL CHECK (birth_month >= 1 AND birth_month <= 12),
    birth_day INT NOT NULL CHECK (birth_day >= 1 AND birth_day <= 31),
    last_announced_year INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (guild_id, user_id)
);

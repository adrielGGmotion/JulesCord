CREATE TABLE giveaways (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    channel_id VARCHAR(255) NOT NULL,
    message_id VARCHAR(255) NOT NULL UNIQUE,
    prize VARCHAR(255) NOT NULL,
    winner_count INT NOT NULL DEFAULT 1,
    end_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ended BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT giveaways_message_id_unique UNIQUE (message_id)
);

CREATE TABLE giveaway_entrants (
    id SERIAL PRIMARY KEY,
    giveaway_id INT NOT NULL REFERENCES giveaways(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT giveaway_entrants_unique UNIQUE (giveaway_id, user_id)
);

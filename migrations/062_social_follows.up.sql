CREATE TABLE IF NOT EXISTS social_follows (
    follower_id VARCHAR(255) NOT NULL,
    following_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (follower_id, following_id)
);

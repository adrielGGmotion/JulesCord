CREATE TABLE IF NOT EXISTS global_economy (
    user_id TEXT PRIMARY KEY,
    total_coins BIGINT NOT NULL DEFAULT 0
);

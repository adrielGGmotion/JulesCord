CREATE TABLE IF NOT EXISTS available_jobs (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    salary INTEGER NOT NULL DEFAULT 100,
    required_level INTEGER NOT NULL DEFAULT 1,
    UNIQUE(guild_id, name)
);

ALTER TABLE user_economy ADD COLUMN IF NOT EXISTS job_id INTEGER REFERENCES available_jobs(id) ON DELETE SET NULL;

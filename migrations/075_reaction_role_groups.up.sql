CREATE TABLE IF NOT EXISTS reaction_role_groups (
    id SERIAL PRIMARY KEY,
    guild_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_exclusive BOOLEAN DEFAULT false,
    max_roles INT DEFAULT 0
);

ALTER TABLE reaction_roles
ADD COLUMN group_id INT REFERENCES reaction_role_groups(id) ON DELETE SET NULL;

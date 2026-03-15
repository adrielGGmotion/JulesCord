ALTER TABLE reaction_roles
ADD COLUMN emoji_id VARCHAR(255) DEFAULT '',
ADD COLUMN is_custom BOOLEAN DEFAULT false;

ALTER TABLE reaction_roles
DROP CONSTRAINT reaction_roles_pkey;

ALTER TABLE reaction_roles
ADD PRIMARY KEY (message_id, emoji, emoji_id);

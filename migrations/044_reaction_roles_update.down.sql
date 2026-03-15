ALTER TABLE reaction_roles
DROP CONSTRAINT reaction_roles_pkey;

ALTER TABLE reaction_roles
DROP COLUMN emoji_id;

ALTER TABLE reaction_roles
DROP COLUMN is_custom;

ALTER TABLE reaction_roles
ADD PRIMARY KEY (message_id, emoji);

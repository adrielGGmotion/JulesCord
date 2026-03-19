ALTER TABLE voice_generator_config
ADD COLUMN allow_custom_names BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN default_name_template VARCHAR(255);

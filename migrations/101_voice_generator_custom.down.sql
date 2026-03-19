ALTER TABLE voice_generator_config
DROP COLUMN IF EXISTS allow_custom_names,
DROP COLUMN IF EXISTS default_name_template;

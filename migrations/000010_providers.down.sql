ALTER TABLE jobs DROP COLUMN model;
ALTER TABLE jobs DROP COLUMN provider_type;
ALTER TABLE jobs DROP COLUMN provider_config_id;
DROP TABLE IF EXISTS provider_credentials;
DROP TABLE IF EXISTS provider_models;
DROP TABLE IF EXISTS provider_configs;

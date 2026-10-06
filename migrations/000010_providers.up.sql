CREATE TABLE provider_configs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  provider_type VARCHAR(16) NOT NULL CHECK (provider_type IN ('openai', 'gemini', 'mock')),
  display_name VARCHAR(100) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  status VARCHAR(16) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'healthy', 'invalid', 'rate_limited', 'quota_exhausted', 'unavailable', 'disabled')),
  last_validated_at TIMESTAMPTZ NULL,
  last_error TEXT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (organization_id, provider_type, display_name)
);

CREATE TABLE provider_models (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  provider_config_id UUID NOT NULL REFERENCES provider_configs(id) ON DELETE CASCADE,
  model VARCHAR(100) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (provider_config_id, model)
);

CREATE TABLE provider_credentials (
  provider_config_id UUID PRIMARY KEY REFERENCES provider_configs(id) ON DELETE CASCADE,
  api_key_encrypted TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX provider_configs_org_idx ON provider_configs(organization_id);

ALTER TABLE jobs ADD COLUMN provider_config_id UUID NULL REFERENCES provider_configs(id) ON DELETE RESTRICT;
ALTER TABLE jobs ADD COLUMN provider_type VARCHAR(16) NULL;
ALTER TABLE jobs ADD COLUMN model VARCHAR(100) NULL;

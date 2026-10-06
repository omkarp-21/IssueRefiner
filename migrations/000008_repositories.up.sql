CREATE TABLE github_tokens (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  access_token TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE repositories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  github_id BIGINT NOT NULL,
  owner VARCHAR(39) NOT NULL,
  name VARCHAR(100) NOT NULL,
  full_name VARCHAR(200) NOT NULL,
  description TEXT NULL,
  default_branch VARCHAR(100) NOT NULL,
  html_url TEXT NOT NULL,
  private BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  synced_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (organization_id, github_id),
  UNIQUE (organization_id, owner, name)
);

CREATE INDEX repositories_org_idx ON repositories(organization_id);

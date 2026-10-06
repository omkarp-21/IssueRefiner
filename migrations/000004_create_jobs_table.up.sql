CREATE TABLE jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  repo_owner VARCHAR(39) NOT NULL,
  repo_name VARCHAR(100) NOT NULL,
  issue_number INT NOT NULL CHECK (issue_number > 0),
  status VARCHAR(16) NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed')),
  input_title TEXT NOT NULL DEFAULT '',
  input_body TEXT NOT NULL DEFAULT '',
  result TEXT NULL,
  error TEXT NULL,
  attempts INT NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX jobs_user_id_created_idx ON jobs(user_id, created_at DESC);

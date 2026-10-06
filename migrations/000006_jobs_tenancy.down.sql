ALTER TABLE jobs ADD COLUMN user_id UUID NULL REFERENCES users(id) ON DELETE CASCADE;
UPDATE jobs SET user_id = created_by;
ALTER TABLE jobs DROP COLUMN organization_id;
ALTER TABLE jobs DROP COLUMN created_by;
DROP INDEX IF EXISTS jobs_org_created_idx;
CREATE INDEX jobs_user_id_created_idx ON jobs(user_id, created_at DESC);

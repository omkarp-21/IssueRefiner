ALTER TABLE jobs ADD COLUMN organization_id UUID NULL REFERENCES organizations(id) ON DELETE CASCADE;
ALTER TABLE jobs ADD COLUMN created_by UUID NULL REFERENCES users(id) ON DELETE CASCADE;

-- One personal organization per user that already owns jobs.
INSERT INTO organizations (id, name)
SELECT gen_random_uuid(), u.username || ' / personal'
FROM users u WHERE u.id IN (SELECT DISTINCT user_id FROM jobs);

INSERT INTO organization_members (organization_id, user_id, role)
SELECT o.id, u.id, 'owner'
FROM organizations o JOIN users u ON o.name = u.username || ' / personal';

-- Move every existing job into its submitter's personal organization.
UPDATE jobs j SET organization_id = m.organization_id, created_by = j.user_id
FROM organization_members m WHERE m.user_id = j.user_id;

ALTER TABLE jobs ALTER COLUMN organization_id SET NOT NULL;
ALTER TABLE jobs ALTER COLUMN created_by SET NOT NULL;

DROP INDEX IF EXISTS jobs_user_id_created_idx;
CREATE INDEX jobs_org_created_idx ON jobs(organization_id, created_at DESC);

ALTER TABLE jobs DROP COLUMN user_id;

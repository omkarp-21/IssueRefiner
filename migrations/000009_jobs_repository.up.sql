ALTER TABLE jobs ADD COLUMN repository_id UUID NULL REFERENCES repositories(id) ON DELETE RESTRICT;

-- Link every existing job to its connected repository, scoped by organization.
-- Jobs without a matching connected repository keep repository_id NULL.
UPDATE jobs j SET repository_id = r.id
FROM repositories r
WHERE r.organization_id = j.organization_id
AND r.owner = j.repo_owner
AND r.name = j.repo_name;

ALTER TABLE jobs DROP COLUMN repo_owner;
ALTER TABLE jobs DROP COLUMN repo_name;

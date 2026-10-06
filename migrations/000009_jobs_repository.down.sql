ALTER TABLE jobs ADD COLUMN repo_owner VARCHAR(39) NULL;
ALTER TABLE jobs ADD COLUMN repo_name VARCHAR(100) NULL;
UPDATE jobs j SET repo_owner = r.owner, repo_name = r.name
FROM repositories r WHERE r.id = j.repository_id;
ALTER TABLE jobs DROP COLUMN repository_id;

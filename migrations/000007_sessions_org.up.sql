ALTER TABLE sessions ADD COLUMN organization_id UUID NULL REFERENCES organizations(id) ON DELETE CASCADE;

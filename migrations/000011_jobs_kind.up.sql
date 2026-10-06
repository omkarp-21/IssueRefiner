ALTER TABLE jobs ADD COLUMN kind VARCHAR(16) NOT NULL DEFAULT 'refine' CHECK (kind IN ('refine', 'investigate'));

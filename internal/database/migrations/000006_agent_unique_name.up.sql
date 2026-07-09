CREATE UNIQUE INDEX IF NOT EXISTS idx_agents_name ON agents (name) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_automations_name ON automations (name) WHERE deleted_at IS NULL;

ALTER TABLE agents DROP COLUMN IF EXISTS version;

CREATE TABLE IF NOT EXISTS required_agents (
    id UUID PRIMARY KEY,
    agent_name VARCHAR(255) NOT NULL,
    agent_type VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_required_agents_name
    ON required_agents (agent_name) WHERE deleted_at IS NULL;

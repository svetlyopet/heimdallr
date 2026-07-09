CREATE TABLE IF NOT EXISTS server_agents (
    server_id UUID NOT NULL REFERENCES servers (id),
    agent_id UUID NOT NULL REFERENCES agents (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (server_id, agent_id)
);

CREATE INDEX IF NOT EXISTS idx_server_agents_agent_id ON server_agents (agent_id);

INSERT INTO server_agents (server_id, agent_id, created_at)
SELECT server_id, id, created_at
FROM agents
WHERE server_id IS NOT NULL
  AND deleted_at IS NULL;

ALTER TABLE agents DROP COLUMN IF EXISTS server_id;
ALTER TABLE agents DROP COLUMN IF EXISTS server;

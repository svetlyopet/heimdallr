ALTER TABLE server_agents RENAME TO agents;

ALTER TABLE agents ADD COLUMN server VARCHAR(255) NOT NULL DEFAULT '';

UPDATE agents
SET server = (
    SELECT hostname FROM servers WHERE servers.id = agents.server_id
);

DROP INDEX IF EXISTS idx_server_agents_server_id;
CREATE INDEX IF NOT EXISTS idx_agents_server_id ON agents (server_id);

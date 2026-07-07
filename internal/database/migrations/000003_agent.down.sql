DROP INDEX IF EXISTS idx_agents_server_id;

ALTER TABLE agents DROP COLUMN server;

ALTER TABLE agents RENAME TO server_agents;

CREATE INDEX IF NOT EXISTS idx_server_agents_server_id ON server_agents (server_id);

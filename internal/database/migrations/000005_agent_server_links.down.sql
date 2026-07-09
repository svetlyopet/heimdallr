ALTER TABLE agents ADD COLUMN IF NOT EXISTS server_id UUID REFERENCES servers (id) ON DELETE SET NULL;
ALTER TABLE agents ADD COLUMN IF NOT EXISTS server VARCHAR(255) NOT NULL DEFAULT '';

UPDATE agents
SET server_id = server_agents.server_id,
    server = servers.hostname
FROM server_agents
JOIN servers ON servers.id = server_agents.server_id
WHERE agents.id = server_agents.agent_id;

DROP TABLE IF EXISTS server_agents;

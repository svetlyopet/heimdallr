ALTER TABLE agents DROP CONSTRAINT IF EXISTS server_agents_server_id_fkey;
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_server_id_fkey;

ALTER TABLE agents ALTER COLUMN server_id DROP NOT NULL;

ALTER TABLE agents
    ADD CONSTRAINT agents_server_id_fkey
    FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE SET NULL;

-- Down migration requires no NULL server_id rows before restoring NOT NULL.
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_server_id_fkey;

UPDATE agents SET server_id = (SELECT id FROM servers LIMIT 1) WHERE server_id IS NULL;

ALTER TABLE agents ALTER COLUMN server_id SET NOT NULL;

ALTER TABLE agents
    ADD CONSTRAINT agents_server_id_fkey
    FOREIGN KEY (server_id) REFERENCES servers (id);

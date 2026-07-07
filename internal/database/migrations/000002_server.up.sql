CREATE TABLE IF NOT EXISTS servers (
    id UUID PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    operating_system VARCHAR(255) NOT NULL DEFAULT '',
    hypervisor VARCHAR(255) NOT NULL DEFAULT '',
    location VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_servers_hostname ON servers (hostname) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS server_agents (
    id UUID PRIMARY KEY,
    server_id UUID NOT NULL REFERENCES servers (id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(255) NOT NULL DEFAULT '',
    version VARCHAR(255) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_server_agents_server_id ON server_agents (server_id);

CREATE TABLE IF NOT EXISTS server_jobs (
    server_id UUID NOT NULL REFERENCES servers (id),
    job_id VARCHAR(255) NOT NULL,
    automation_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (server_id, job_id, automation_id),
    FOREIGN KEY (job_id, automation_id) REFERENCES jobs (id, automation_id)
);

CREATE INDEX IF NOT EXISTS idx_server_jobs_server_id ON server_jobs (server_id);

CREATE TABLE IF NOT EXISTS server_releases (
    server_id UUID NOT NULL REFERENCES servers (id),
    release_id UUID NOT NULL REFERENCES releases (id),
    application_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (server_id, release_id),
    FOREIGN KEY (release_id) REFERENCES releases (id)
);

CREATE INDEX IF NOT EXISTS idx_server_releases_server_id ON server_releases (server_id);

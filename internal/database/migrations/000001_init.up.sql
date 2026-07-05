CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash CHAR(64) NOT NULL,
    roles TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS providers (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_name ON providers (name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS automations (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL,
    provider VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL REFERENCES providers (id),
    cost_savings DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_automations_provider_id ON automations (provider_id);

CREATE TABLE IF NOT EXISTS jobs (
    id VARCHAR(255) NOT NULL,
    automation_id UUID NOT NULL REFERENCES automations (id),
    automation VARCHAR(255) NOT NULL,
    provider_id UUID NOT NULL,
    provider VARCHAR(255) NOT NULL,
    status VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    output TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (id, automation_id)
);

CREATE INDEX IF NOT EXISTS idx_jobs_automation_id ON jobs (automation_id);
CREATE INDEX IF NOT EXISTS idx_jobs_provider_id ON jobs (provider_id);

CREATE TABLE IF NOT EXISTS applications (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    repository_url VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_applications_name ON applications (name) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS releases (
    id UUID PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES applications (id),
    application VARCHAR(255) NOT NULL,
    version VARCHAR(255) NOT NULL,
    commit_sha VARCHAR(255) NOT NULL DEFAULT '',
    pipeline_url VARCHAR(255) NOT NULL DEFAULT '',
    branch VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_releases_application_version ON releases (application_id, version) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_releases_application_id ON releases (application_id);

CREATE TABLE IF NOT EXISTS reports (
    id VARCHAR(255) NOT NULL,
    release_id UUID NOT NULL REFERENCES releases (id),
    application_id UUID NOT NULL,
    application VARCHAR(255) NOT NULL,
    version VARCHAR(255) NOT NULL,
    type VARCHAR(255) NOT NULL,
    status VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL DEFAULT '',
    url VARCHAR(255) NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}',
    output TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    PRIMARY KEY (id, release_id)
);

CREATE INDEX IF NOT EXISTS idx_reports_release_id ON reports (release_id);
CREATE INDEX IF NOT EXISTS idx_reports_application_id ON reports (application_id);

CREATE TABLE IF NOT EXISTS api_tokens (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    scopes TEXT NOT NULL,
    created_by UUID REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_tokens_token_hash ON api_tokens (token_hash) WHERE deleted_at IS NULL;

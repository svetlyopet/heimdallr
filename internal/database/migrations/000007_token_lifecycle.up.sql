ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS kind VARCHAR(32) NOT NULL DEFAULT 'api';
ALTER TABLE api_tokens ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ NULL;

UPDATE api_tokens SET kind = 'session' WHERE name LIKE 'session-%' AND kind = 'api';

UPDATE api_tokens
SET expires_at = CASE
    WHEN kind = 'session' THEN NOW() + INTERVAL '24 hours'
    ELSE NOW() + INTERVAL '90 days'
END
WHERE expires_at IS NULL;

ALTER TABLE api_tokens ALTER COLUMN expires_at SET NOT NULL;

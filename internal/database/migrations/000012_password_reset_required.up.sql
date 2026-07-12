ALTER TABLE users ADD COLUMN password_reset_required BOOLEAN NOT NULL DEFAULT false;

UPDATE users
SET password_reset_required = true
WHERE length(password_hash) = 64
  AND password_hash ~ '^[0-9a-fA-F]{64}$';

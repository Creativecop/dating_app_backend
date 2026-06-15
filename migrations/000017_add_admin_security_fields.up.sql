ALTER TABLE admin_users
ADD COLUMN password_changed_at TIMESTAMPTZ,
ADD COLUMN token_version INT NOT NULL DEFAULT 1;

UPDATE admin_users
SET token_version = 1
WHERE token_version < 1;

ALTER TABLE admin_sessions
ADD COLUMN revoked_reason VARCHAR(80);

CREATE INDEX idx_admin_users_token_version
ON admin_users(token_version);

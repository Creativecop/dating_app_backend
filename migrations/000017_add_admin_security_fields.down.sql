DROP INDEX IF EXISTS idx_admin_users_token_version;

ALTER TABLE admin_sessions
DROP COLUMN IF EXISTS revoked_reason;

ALTER TABLE admin_users
DROP COLUMN IF EXISTS token_version,
DROP COLUMN IF EXISTS password_changed_at;

DROP INDEX IF EXISTS idx_admin_audit_logs_request_id;

ALTER TABLE admin_audit_logs
DROP COLUMN IF EXISTS request_id;

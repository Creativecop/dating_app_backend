ALTER TABLE admin_audit_logs
ADD COLUMN IF NOT EXISTS request_id VARCHAR(100);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_request_id
ON admin_audit_logs(request_id)
WHERE request_id IS NOT NULL;

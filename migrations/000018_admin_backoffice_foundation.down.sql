DROP TRIGGER IF EXISTS trg_user_restrictions_set_updated_at ON user_restrictions;
DROP INDEX IF EXISTS idx_user_restrictions_type_status;
DROP INDEX IF EXISTS idx_user_restrictions_user_status;
DROP INDEX IF EXISTS user_restrictions_active_unique;
DROP TABLE IF EXISTS user_restrictions;

DROP TRIGGER IF EXISTS trg_admin_audit_logs_insert_only ON admin_audit_logs;
DROP FUNCTION IF EXISTS prevent_admin_audit_log_mutation();

ALTER TABLE admin_audit_logs
DROP CONSTRAINT IF EXISTS admin_audit_logs_actor_type_check;

ALTER TABLE admin_audit_logs
DROP COLUMN IF EXISTS reason,
DROP COLUMN IF EXISTS actor_type;

DROP TRIGGER IF EXISTS trg_admin_role_assignments_set_updated_at ON admin_role_assignments;
DROP INDEX IF EXISTS idx_admin_role_assignments_role_status;
DROP INDEX IF EXISTS idx_admin_role_assignments_admin_user_status;
DROP INDEX IF EXISTS admin_role_assignments_active_unique;
DROP TABLE IF EXISTS admin_role_assignments;

ALTER TABLE admin_users
DROP CONSTRAINT IF EXISTS admin_users_status_check;

UPDATE admin_users
SET status = 'SUSPENDED'
WHERE status IN ('INVITED', 'DISABLED', 'LOCKED');

ALTER TABLE admin_users
ADD CONSTRAINT admin_users_status_check CHECK (
  status IN ('ACTIVE', 'SUSPENDED')
);

ALTER TABLE admin_users
DROP COLUMN IF EXISTS must_change_password;

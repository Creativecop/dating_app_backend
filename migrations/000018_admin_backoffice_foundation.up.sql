ALTER TABLE admin_users
ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE admin_users
SET status = 'DISABLED'
WHERE status = 'SUSPENDED';

ALTER TABLE admin_users
DROP CONSTRAINT IF EXISTS admin_users_status_check;

ALTER TABLE admin_users
ADD CONSTRAINT admin_users_status_check CHECK (
  status IN ('INVITED', 'ACTIVE', 'DISABLED', 'LOCKED')
);

CREATE TABLE admin_role_assignments (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  admin_user_id BIGINT NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,

  role VARCHAR(60) NOT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',

  assigned_by_admin_user_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
  removed_by_admin_user_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,

  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  removed_at TIMESTAMPTZ,
  reason TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT admin_role_assignments_role_check CHECK (
    role IN (
      'SUPER_ADMIN',
      'PLATFORM_ADMIN',
      'OPS_MANAGER',
      'FINANCE_ADMIN',
      'TRUST_SAFETY',
      'SUPPORT_AGENT',
      'GIFT_MANAGER',
      'AGENCY_MANAGER',
      'RESELLER_MANAGER',
      'ANALYST'
    )
  ),

  CONSTRAINT admin_role_assignments_status_check CHECK (
    status IN ('ACTIVE', 'REMOVED')
  ),

  CONSTRAINT admin_role_assignments_removed_check CHECK (
    (status = 'ACTIVE' AND removed_at IS NULL)
    OR
    (status = 'REMOVED' AND removed_at IS NOT NULL)
  )
);

CREATE UNIQUE INDEX admin_role_assignments_active_unique
ON admin_role_assignments(admin_user_id, role)
WHERE status = 'ACTIVE';

CREATE INDEX idx_admin_role_assignments_admin_user_status
ON admin_role_assignments(admin_user_id, status);

CREATE INDEX idx_admin_role_assignments_role_status
ON admin_role_assignments(role, status);

CREATE TRIGGER trg_admin_role_assignments_set_updated_at
BEFORE UPDATE ON admin_role_assignments
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE admin_audit_logs
ADD COLUMN actor_type VARCHAR(30) NOT NULL DEFAULT 'ADMIN',
ADD COLUMN reason TEXT;

ALTER TABLE admin_audit_logs
ADD CONSTRAINT admin_audit_logs_actor_type_check CHECK (
  actor_type IN ('ADMIN', 'SYSTEM')
);

CREATE OR REPLACE FUNCTION prevent_admin_audit_log_mutation()
RETURNS TRIGGER AS $$
BEGIN
  RAISE EXCEPTION 'admin_audit_logs is insert-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_admin_audit_logs_insert_only
BEFORE UPDATE OR DELETE ON admin_audit_logs
FOR EACH ROW
EXECUTE FUNCTION prevent_admin_audit_log_mutation();

CREATE TABLE user_restrictions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  restriction_type VARCHAR(80) NOT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
  reason TEXT NOT NULL,

  created_by_admin_user_id BIGINT NOT NULL REFERENCES admin_users(id) ON DELETE RESTRICT,
  revoked_by_admin_user_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
  revoked_at TIMESTAMPTZ,
  revocation_reason TEXT,
  expires_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_restrictions_type_check CHECK (
    restriction_type IN (
      'FULL_PLATFORM_BAN',
      'LIVE_CREATE_BAN',
      'COMMENT_BAN',
      'GIFT_SEND_BAN',
      'RESELLER_OPERATION_BAN',
      'AGENCY_OPERATION_BAN'
    )
  ),

  CONSTRAINT user_restrictions_status_check CHECK (
    status IN ('ACTIVE', 'REVOKED')
  ),

  CONSTRAINT user_restrictions_revoked_check CHECK (
    (status = 'ACTIVE' AND revoked_at IS NULL)
    OR
    (status = 'REVOKED' AND revoked_at IS NOT NULL)
  )
);

CREATE UNIQUE INDEX user_restrictions_active_unique
ON user_restrictions(user_id, restriction_type)
WHERE status = 'ACTIVE';

CREATE INDEX idx_user_restrictions_user_status
ON user_restrictions(user_id, status);

CREATE INDEX idx_user_restrictions_type_status
ON user_restrictions(restriction_type, status);

CREATE TRIGGER trg_user_restrictions_set_updated_at
BEFORE UPDATE ON user_restrictions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

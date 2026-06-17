ALTER TABLE user_restrictions
DROP CONSTRAINT IF EXISTS user_restrictions_lifecycle_check;

ALTER TABLE user_restrictions
DROP CONSTRAINT IF EXISTS user_restrictions_status_check;

UPDATE user_restrictions
SET
  status = 'REVOKED',
  revoked_at = COALESCE(revoked_at, NOW()),
  revocation_reason = COALESCE(revocation_reason, 'Expired before rollback')
WHERE status = 'EXPIRED';

ALTER TABLE user_restrictions
ADD CONSTRAINT user_restrictions_status_check CHECK (
  status IN ('ACTIVE', 'REVOKED')
);

ALTER TABLE user_restrictions
ADD CONSTRAINT user_restrictions_revoked_check CHECK (
  (status = 'ACTIVE' AND revoked_at IS NULL)
  OR
  (status = 'REVOKED' AND revoked_at IS NOT NULL)
);

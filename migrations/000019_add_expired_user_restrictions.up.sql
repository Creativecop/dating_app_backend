ALTER TABLE user_restrictions
DROP CONSTRAINT IF EXISTS user_restrictions_revoked_check;

ALTER TABLE user_restrictions
DROP CONSTRAINT IF EXISTS user_restrictions_status_check;

ALTER TABLE user_restrictions
ADD CONSTRAINT user_restrictions_status_check CHECK (
  status IN ('ACTIVE', 'REVOKED', 'EXPIRED')
);

ALTER TABLE user_restrictions
ADD CONSTRAINT user_restrictions_lifecycle_check CHECK (
  (status = 'ACTIVE' AND revoked_at IS NULL)
  OR
  (status = 'REVOKED' AND revoked_at IS NOT NULL)
  OR
  (status = 'EXPIRED')
);

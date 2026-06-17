CREATE INDEX IF NOT EXISTS idx_user_restrictions_status_type_created
ON user_restrictions(status, restriction_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_restrictions_created_by_created
ON user_restrictions(created_by_admin_user_id, created_at DESC);

DO $$
BEGIN
  IF to_regclass('idx_reports_status_created') IS NULL
     AND to_regclass('idx_reports_admin_status_created') IS NULL THEN
    CREATE INDEX idx_reports_status_created
    ON reports(status, created_at DESC);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_reports_reviewed_at
ON reports(reviewed_at DESC)
WHERE reviewed_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_reports_reason_created
ON reports(reason_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_action_created
ON admin_audit_logs(action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_manual_payment_requests_status_created
ON manual_payment_requests(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_manual_payment_requests_status_submitted
ON manual_payment_requests(status, submitted_at DESC);

CREATE INDEX IF NOT EXISTS idx_manual_payment_requests_provider_created
ON manual_payment_requests(payment_provider, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_manual_payment_requests_plan_created
ON manual_payment_requests(plan_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_manual_payment_requests_reviewed_at
ON manual_payment_requests(reviewed_at DESC)
WHERE reviewed_at IS NOT NULL;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM reports
    WHERE status NOT IN ('OPEN', 'IN_REVIEW', 'RESOLVED', 'DISMISSED', 'PENDING', 'REVIEWED', 'ACTIONED')
  ) THEN
    RAISE EXCEPTION 'unsupported reports.status value found before admin report review migration';
  END IF;
END $$;

DROP INDEX IF EXISTS reports_open_dedupe_unique;

ALTER TABLE reports
DROP CONSTRAINT IF EXISTS reports_status_check;

ALTER TABLE reports
ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS reviewed_by_admin_user_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS review_reason TEXT,
ADD COLUMN IF NOT EXISTS review_note TEXT,
ADD COLUMN IF NOT EXISTS review_action_type VARCHAR(50),
ADD COLUMN IF NOT EXISTS review_metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

UPDATE reports
SET status = CASE status
  WHEN 'OPEN' THEN 'PENDING'
  WHEN 'IN_REVIEW' THEN 'PENDING'
  WHEN 'RESOLVED' THEN 'REVIEWED'
  ELSE status
END;

UPDATE reports
SET
  reviewed_at = COALESCE(reviewed_at, updated_at),
  review_reason = COALESCE(review_reason, 'Migrated legacy report status'),
  review_action_type = COALESCE(review_action_type, 'NONE')
WHERE status IN ('REVIEWED', 'DISMISSED', 'ACTIONED');

ALTER TABLE reports
ADD CONSTRAINT reports_status_check CHECK (
  status IN ('PENDING', 'REVIEWED', 'DISMISSED', 'ACTIONED')
);

ALTER TABLE reports
ADD CONSTRAINT reports_review_action_type_check CHECK (
  review_action_type IS NULL OR review_action_type IN ('NONE', 'RESTRICT_USER')
);

CREATE UNIQUE INDEX reports_open_dedupe_unique
ON reports(reporter_user_id, target_type, target_uuid, reason_id)
WHERE status = 'PENDING';

CREATE INDEX IF NOT EXISTS idx_reports_admin_status_created
ON reports(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_reviewed_at
ON reports(reviewed_at DESC)
WHERE reviewed_at IS NOT NULL;

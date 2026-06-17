DROP INDEX IF EXISTS idx_reports_reviewed_at;
DROP INDEX IF EXISTS idx_reports_admin_status_created;
DROP INDEX IF EXISTS reports_open_dedupe_unique;

ALTER TABLE reports
DROP CONSTRAINT IF EXISTS reports_review_action_type_check;

ALTER TABLE reports
DROP CONSTRAINT IF EXISTS reports_status_check;

UPDATE reports
SET status = CASE status
  WHEN 'PENDING' THEN 'OPEN'
  WHEN 'REVIEWED' THEN 'RESOLVED'
  WHEN 'ACTIONED' THEN 'RESOLVED'
  ELSE status
END;

ALTER TABLE reports
DROP COLUMN IF EXISTS review_metadata,
DROP COLUMN IF EXISTS review_action_type,
DROP COLUMN IF EXISTS review_note,
DROP COLUMN IF EXISTS review_reason,
DROP COLUMN IF EXISTS reviewed_by_admin_user_id,
DROP COLUMN IF EXISTS reviewed_at;

ALTER TABLE reports
ADD CONSTRAINT reports_status_check CHECK (
  status IN ('OPEN', 'IN_REVIEW', 'RESOLVED', 'DISMISSED')
);

CREATE UNIQUE INDEX reports_open_dedupe_unique
ON reports(reporter_user_id, target_type, target_uuid, reason_id)
WHERE status IN ('OPEN', 'IN_REVIEW');

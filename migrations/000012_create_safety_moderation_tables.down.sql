DROP TRIGGER IF EXISTS trg_safety_settings_set_updated_at ON safety_settings;
DROP INDEX IF EXISTS idx_safety_settings_user_id;
DROP TABLE IF EXISTS safety_settings;

DROP INDEX IF EXISTS idx_user_blocks_report_id;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'user_blocks_source_check'
  ) THEN
    ALTER TABLE user_blocks DROP CONSTRAINT user_blocks_source_check;
  END IF;
END $$;

ALTER TABLE user_blocks
DROP COLUMN IF EXISTS report_id,
DROP COLUMN IF EXISTS source,
DROP COLUMN IF EXISTS note,
DROP COLUMN IF EXISTS reason_code;

DROP INDEX IF EXISTS idx_moderation_actions_target_user;
DROP INDEX IF EXISTS idx_moderation_actions_case;
DROP TABLE IF EXISTS moderation_actions;

ALTER TABLE reports
DROP CONSTRAINT IF EXISTS fk_reports_moderation_case;

DROP TRIGGER IF EXISTS trg_reports_set_updated_at ON reports;
DROP INDEX IF EXISTS reports_open_dedupe_unique;
DROP INDEX IF EXISTS idx_reports_target;
DROP INDEX IF EXISTS idx_reports_status;
DROP INDEX IF EXISTS idx_reports_reported_user;
DROP INDEX IF EXISTS idx_reports_reporter;
DROP TABLE IF EXISTS reports;

DROP TRIGGER IF EXISTS trg_moderation_cases_set_updated_at ON moderation_cases;
DROP INDEX IF EXISTS moderation_cases_open_target_unique;
DROP INDEX IF EXISTS idx_moderation_cases_target;
DROP INDEX IF EXISTS idx_moderation_cases_status;
DROP INDEX IF EXISTS idx_moderation_cases_subject;
DROP TABLE IF EXISTS moderation_cases;

DROP TRIGGER IF EXISTS trg_report_reasons_set_updated_at ON report_reasons;
DROP INDEX IF EXISTS idx_report_reasons_active;
DROP TABLE IF EXISTS report_reasons;

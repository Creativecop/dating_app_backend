CREATE TABLE report_reasons (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  reason_code VARCHAR(80) NOT NULL UNIQUE,
  title VARCHAR(150) NOT NULL,
  description TEXT,
  applies_to TEXT[] NOT NULL DEFAULT ARRAY['USER', 'PROFILE', 'MESSAGE', 'MEDIA', 'MATCH'],

  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_reasons_active
ON report_reasons(is_active, sort_order);

CREATE TRIGGER trg_report_reasons_set_updated_at
BEFORE UPDATE ON report_reasons
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE moderation_cases (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  subject_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

  target_type VARCHAR(30),
  target_uuid UUID,

  status VARCHAR(30) NOT NULL DEFAULT 'OPEN',
  priority VARCHAR(30) NOT NULL DEFAULT 'NORMAL',

  report_count INT NOT NULL DEFAULT 1,

  opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  assigned_admin_id BIGINT,
  resolved_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT moderation_cases_target_type_check CHECK (
    target_type IS NULL OR target_type IN ('USER', 'PROFILE', 'MESSAGE', 'MEDIA', 'MATCH')
  ),

  CONSTRAINT moderation_cases_status_check CHECK (
    status IN ('OPEN', 'IN_REVIEW', 'ACTION_TAKEN', 'DISMISSED', 'CLOSED')
  ),

  CONSTRAINT moderation_cases_priority_check CHECK (
    priority IN ('LOW', 'NORMAL', 'HIGH', 'URGENT')
  )
);

CREATE INDEX idx_moderation_cases_subject
ON moderation_cases(subject_user_id, created_at DESC);

CREATE INDEX idx_moderation_cases_status
ON moderation_cases(status);

CREATE INDEX idx_moderation_cases_target
ON moderation_cases(target_type, target_uuid);

CREATE UNIQUE INDEX moderation_cases_open_target_unique
ON moderation_cases(subject_user_id, target_type, target_uuid)
WHERE status IN ('OPEN', 'IN_REVIEW');

CREATE TRIGGER trg_moderation_cases_set_updated_at
BEFORE UPDATE ON moderation_cases
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE reports (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  reporter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  reported_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

  target_type VARCHAR(30) NOT NULL,
  target_uuid UUID NOT NULL,

  reason_id BIGINT NOT NULL REFERENCES report_reasons(id) ON DELETE RESTRICT,

  note TEXT,
  evidence_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

  status VARCHAR(30) NOT NULL DEFAULT 'OPEN',
  severity VARCHAR(30) NOT NULL DEFAULT 'MEDIUM',

  moderation_case_id BIGINT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT reports_target_type_check CHECK (
    target_type IN ('USER', 'PROFILE', 'MESSAGE', 'MEDIA', 'MATCH')
  ),

  CONSTRAINT reports_status_check CHECK (
    status IN ('OPEN', 'IN_REVIEW', 'RESOLVED', 'DISMISSED')
  ),

  CONSTRAINT reports_severity_check CHECK (
    severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')
  ),

  CONSTRAINT reports_note_length_check CHECK (
    note IS NULL OR length(note) <= 1000
  )
);

CREATE INDEX idx_reports_reporter
ON reports(reporter_user_id, created_at DESC);

CREATE INDEX idx_reports_reported_user
ON reports(reported_user_id, created_at DESC);

CREATE INDEX idx_reports_status
ON reports(status);

CREATE INDEX idx_reports_target
ON reports(target_type, target_uuid);

CREATE UNIQUE INDEX reports_open_dedupe_unique
ON reports(reporter_user_id, target_type, target_uuid, reason_id)
WHERE status IN ('OPEN', 'IN_REVIEW');

CREATE TRIGGER trg_reports_set_updated_at
BEFORE UPDATE ON reports
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE reports
ADD CONSTRAINT fk_reports_moderation_case
FOREIGN KEY (moderation_case_id)
REFERENCES moderation_cases(id)
ON DELETE SET NULL;

CREATE TABLE moderation_actions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  moderation_case_id BIGINT REFERENCES moderation_cases(id) ON DELETE SET NULL,

  actor_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
  target_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

  action_type VARCHAR(50) NOT NULL,
  reason TEXT,

  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT moderation_actions_type_check CHECK (
    action_type IN (
      'REPORT_CREATED',
      'USER_BLOCKED',
      'USER_WARNED',
      'USER_SUSPENDED',
      'USER_BANNED',
      'REPORT_DISMISSED',
      'CASE_CLOSED'
    )
  )
);

CREATE INDEX idx_moderation_actions_case
ON moderation_actions(moderation_case_id, created_at DESC);

CREATE INDEX idx_moderation_actions_target_user
ON moderation_actions(target_user_id, created_at DESC);

ALTER TABLE user_blocks
ADD COLUMN IF NOT EXISTS reason_code VARCHAR(80),
ADD COLUMN IF NOT EXISTS note TEXT,
ADD COLUMN IF NOT EXISTS source VARCHAR(30) NOT NULL DEFAULT 'MANUAL',
ADD COLUMN IF NOT EXISTS report_id BIGINT REFERENCES reports(id) ON DELETE SET NULL;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'user_blocks_source_check'
  ) THEN
    ALTER TABLE user_blocks
    ADD CONSTRAINT user_blocks_source_check
    CHECK (source IN ('MANUAL', 'REPORT_FLOW', 'MODERATION'));
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_user_blocks_report_id
ON user_blocks(report_id);

CREATE TABLE safety_settings (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

  allow_message_requests BOOLEAN NOT NULL DEFAULT TRUE,
  auto_hide_blocked_users BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_safety_settings_user_id
ON safety_settings(user_id);

CREATE TRIGGER trg_safety_settings_set_updated_at
BEFORE UPDATE ON safety_settings
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

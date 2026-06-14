ALTER TABLE devices
ADD COLUMN IF NOT EXISTS push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS fcm_token_updated_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS last_push_success_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS last_push_failure_at TIMESTAMPTZ,
ADD COLUMN IF NOT EXISTS push_failure_count INT NOT NULL DEFAULT 0;

WITH ranked_tokens AS (
  SELECT
    id,
    ROW_NUMBER() OVER (
      PARTITION BY fcm_token
      ORDER BY
        COALESCE(fcm_token_updated_at, last_active_at, updated_at, created_at) DESC,
        id DESC
    ) AS token_rank
  FROM devices
  WHERE fcm_token IS NOT NULL
)
UPDATE devices d
SET
  fcm_token = NULL,
  push_enabled = FALSE,
  updated_at = NOW()
FROM ranked_tokens rt
WHERE d.id = rt.id
  AND rt.token_rank > 1;

CREATE INDEX IF NOT EXISTS idx_devices_user_push_enabled
ON devices(user_id, push_enabled);

CREATE UNIQUE INDEX IF NOT EXISTS devices_fcm_token_unique
ON devices(fcm_token)
WHERE fcm_token IS NOT NULL;

CREATE TABLE notification_settings (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

  push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  new_match_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  chat_message_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  super_like_enabled BOOLEAN NOT NULL DEFAULT TRUE,

  quiet_hours_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  quiet_hours_start TIME,
  quiet_hours_end TIME,
  timezone VARCHAR(80) NOT NULL DEFAULT 'Asia/Dhaka',

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT notification_settings_quiet_hours_check CHECK (
    quiet_hours_enabled = FALSE
    OR (quiet_hours_start IS NOT NULL AND quiet_hours_end IS NOT NULL)
  )
);

CREATE INDEX idx_notification_settings_user_id
ON notification_settings(user_id);

CREATE TRIGGER trg_notification_settings_set_updated_at
BEFORE UPDATE ON notification_settings
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE notifications (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  notification_type VARCHAR(50) NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  data JSONB NOT NULL DEFAULT '{}'::jsonb,

  read_at TIMESTAMPTZ,
  clicked_at TIMESTAMPTZ,
  dedupe_key TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT notifications_type_check CHECK (
    notification_type IN ('NEW_MATCH', 'CHAT_MESSAGE', 'SUPER_LIKE', 'SYSTEM')
  )
);

CREATE UNIQUE INDEX notifications_dedupe_unique
ON notifications(user_id, dedupe_key)
WHERE dedupe_key IS NOT NULL;

CREATE INDEX idx_notifications_user_created
ON notifications(user_id, created_at DESC, id DESC);

CREATE INDEX idx_notifications_user_read
ON notifications(user_id, read_at);

CREATE TABLE push_delivery_logs (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  notification_id BIGINT REFERENCES notifications(id) ON DELETE SET NULL,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  device_id BIGINT REFERENCES devices(id) ON DELETE SET NULL,

  provider VARCHAR(50) NOT NULL DEFAULT 'FCM',
  status VARCHAR(30) NOT NULL,

  provider_message_id TEXT,
  error_code TEXT,
  error_message TEXT,

  attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT push_delivery_logs_status_check CHECK (
    status IN ('SENT', 'FAILED', 'SKIPPED')
  )
);

CREATE INDEX idx_push_delivery_logs_user
ON push_delivery_logs(user_id, attempted_at DESC);

CREATE INDEX idx_push_delivery_logs_notification
ON push_delivery_logs(notification_id);

CREATE INDEX idx_push_delivery_logs_device
ON push_delivery_logs(device_id, attempted_at DESC);

CREATE UNIQUE INDEX push_delivery_logs_sent_unique
ON push_delivery_logs(notification_id, device_id, provider)
WHERE status = 'SENT';

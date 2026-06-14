DROP INDEX IF EXISTS push_delivery_logs_sent_unique;
DROP INDEX IF EXISTS idx_push_delivery_logs_device;
DROP INDEX IF EXISTS idx_push_delivery_logs_notification;
DROP INDEX IF EXISTS idx_push_delivery_logs_user;
DROP TABLE IF EXISTS push_delivery_logs;

DROP INDEX IF EXISTS idx_notifications_user_read;
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP INDEX IF EXISTS notifications_dedupe_unique;
DROP TABLE IF EXISTS notifications;

DROP TRIGGER IF EXISTS trg_notification_settings_set_updated_at ON notification_settings;
DROP INDEX IF EXISTS idx_notification_settings_user_id;
DROP TABLE IF EXISTS notification_settings;

DROP INDEX IF EXISTS devices_fcm_token_unique;
DROP INDEX IF EXISTS idx_devices_user_push_enabled;

ALTER TABLE devices
DROP COLUMN IF EXISTS push_failure_count,
DROP COLUMN IF EXISTS last_push_failure_at,
DROP COLUMN IF EXISTS last_push_success_at,
DROP COLUMN IF EXISTS fcm_token_updated_at,
DROP COLUMN IF EXISTS push_enabled;

DROP TRIGGER IF EXISTS trg_devices_set_updated_at ON devices;
DROP TABLE IF EXISTS devices;

DROP TRIGGER IF EXISTS trg_user_sessions_set_updated_at ON user_sessions;
DROP TABLE IF EXISTS user_sessions;

DROP TABLE IF EXISTS otp_delivery_logs;

DROP TRIGGER IF EXISTS trg_otp_codes_set_updated_at ON otp_codes;
DROP TABLE IF EXISTS otp_codes;

DROP TRIGGER IF EXISTS trg_users_set_updated_at ON users;
DROP TABLE IF EXISTS users;

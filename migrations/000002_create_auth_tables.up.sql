CREATE TABLE users (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  phone VARCHAR(30),
  email CITEXT,

  phone_verified_at TIMESTAMPTZ,
  email_verified_at TIMESTAMPTZ,

  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
  onboarding_status VARCHAR(30) NOT NULL DEFAULT 'PROFILE_REQUIRED',

  last_login_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT users_status_check CHECK (
    status IN ('ACTIVE', 'SUSPENDED', 'BANNED', 'DELETED')
  ),

  CONSTRAINT users_onboarding_status_check CHECK (
    onboarding_status IN ('PENDING', 'PROFILE_REQUIRED', 'COMPLETED')
  ),

  CONSTRAINT users_phone_or_email_required CHECK (
    phone IS NOT NULL OR email IS NOT NULL
  )
);

CREATE UNIQUE INDEX users_phone_unique_active
ON users(phone)
WHERE phone IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX users_email_unique_active
ON users(email)
WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

CREATE TRIGGER trg_users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE otp_codes (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

  channel VARCHAR(30) NOT NULL,
  purpose VARCHAR(50) NOT NULL,

  identifier_hash TEXT NOT NULL,

  phone VARCHAR(30),
  email CITEXT,

  otp_hash TEXT NOT NULL,

  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,

  attempt_count INT NOT NULL DEFAULT 0,
  max_attempts INT NOT NULL DEFAULT 5,
  resend_count INT NOT NULL DEFAULT 0,

  ip_address VARCHAR(100),
  user_agent TEXT,
  device_id VARCHAR(150),

  delivery_status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
  delivery_provider VARCHAR(50),
  provider_message_id TEXT,
  delivery_attempts INT NOT NULL DEFAULT 0,
  last_delivery_attempt_at TIMESTAMPTZ,
  sent_at TIMESTAMPTZ,
  failed_at TIMESTAMPTZ,
  delivery_error TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT otp_codes_channel_check CHECK (
    channel IN ('WHATSAPP', 'EMAIL')
  ),

  CONSTRAINT otp_codes_purpose_check CHECK (
    purpose IN ('LOGIN', 'REGISTER', 'VERIFY_PHONE', 'VERIFY_EMAIL', 'RESET')
  ),

  CONSTRAINT otp_codes_delivery_status_check CHECK (
    delivery_status IN ('PENDING', 'QUEUED', 'SENDING', 'SENT', 'FAILED', 'EXPIRED')
  )
);

CREATE INDEX idx_otp_codes_identifier_hash ON otp_codes(identifier_hash);
CREATE INDEX idx_otp_codes_phone ON otp_codes(phone);
CREATE INDEX idx_otp_codes_email ON otp_codes(email);
CREATE INDEX idx_otp_codes_expires_at ON otp_codes(expires_at);
CREATE INDEX idx_otp_codes_consumed_at ON otp_codes(consumed_at);
CREATE INDEX idx_otp_codes_delivery_status ON otp_codes(delivery_status);

CREATE TRIGGER trg_otp_codes_set_updated_at
BEFORE UPDATE ON otp_codes
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE otp_delivery_logs (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  otp_code_id BIGINT NOT NULL REFERENCES otp_codes(id) ON DELETE CASCADE,

  channel VARCHAR(30) NOT NULL,
  provider VARCHAR(50),
  identifier TEXT NOT NULL,

  status VARCHAR(30) NOT NULL,
  provider_message_id TEXT,
  error_message TEXT,

  attempted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_otp_delivery_logs_otp_code_id ON otp_delivery_logs(otp_code_id);
CREATE INDEX idx_otp_delivery_logs_status ON otp_delivery_logs(status);

CREATE TABLE user_sessions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  refresh_token_hash TEXT NOT NULL UNIQUE,
  token_family_id UUID NOT NULL DEFAULT gen_random_uuid(),

  device_id VARCHAR(150),
  device_name VARCHAR(150),
  platform VARCHAR(50),

  ip_address VARCHAR(100),
  user_agent TEXT,

  expires_at TIMESTAMPTZ NOT NULL,
  last_used_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  replaced_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_token_family_id ON user_sessions(token_family_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
CREATE INDEX idx_user_sessions_revoked_at ON user_sessions(revoked_at);

CREATE TRIGGER trg_user_sessions_set_updated_at
BEFORE UPDATE ON user_sessions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE devices (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  device_id VARCHAR(150) NOT NULL,
  device_name VARCHAR(150),
  platform VARCHAR(50),

  fcm_token TEXT,

  app_version VARCHAR(50),
  os_version VARCHAR(100),

  last_active_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT devices_user_device_unique UNIQUE (user_id, device_id)
);

CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE INDEX idx_devices_device_id ON devices(device_id);
CREATE INDEX idx_devices_fcm_token ON devices(fcm_token);

CREATE TRIGGER trg_devices_set_updated_at
BEFORE UPDATE ON devices
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

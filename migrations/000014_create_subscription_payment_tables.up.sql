CREATE TABLE admin_users (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  email CITEXT NOT NULL UNIQUE,
  name VARCHAR(120),
  password_hash TEXT NOT NULL,

  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',
  last_login_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT admin_users_status_check CHECK (
    status IN ('ACTIVE', 'SUSPENDED')
  )
);

CREATE INDEX idx_admin_users_status
ON admin_users(status);

CREATE TRIGGER trg_admin_users_set_updated_at
BEFORE UPDATE ON admin_users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE admin_sessions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  admin_user_id BIGINT NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,

  refresh_token_hash TEXT NOT NULL UNIQUE,
  token_family_id UUID NOT NULL DEFAULT gen_random_uuid(),

  ip_address VARCHAR(100),
  user_agent TEXT,

  expires_at TIMESTAMPTZ NOT NULL,
  last_used_at TIMESTAMPTZ,
  revoked_at TIMESTAMPTZ,
  replaced_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_sessions_admin_user_id
ON admin_sessions(admin_user_id);

CREATE INDEX idx_admin_sessions_expires_at
ON admin_sessions(expires_at);

CREATE INDEX idx_admin_sessions_revoked_at
ON admin_sessions(revoked_at);

CREATE TRIGGER trg_admin_sessions_set_updated_at
BEFORE UPDATE ON admin_sessions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE admin_user_permissions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  admin_user_id BIGINT NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
  permission_code VARCHAR(120) NOT NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT admin_user_permissions_unique UNIQUE (
    admin_user_id,
    permission_code
  )
);

CREATE INDEX idx_admin_user_permissions_admin_user_id
ON admin_user_permissions(admin_user_id);

CREATE TABLE admin_audit_logs (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  admin_user_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,

  action VARCHAR(120) NOT NULL,
  resource_type VARCHAR(120) NOT NULL,
  resource_uuid UUID,

  before_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  after_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

  ip_address VARCHAR(100),
  user_agent TEXT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_logs_admin_created
ON admin_audit_logs(admin_user_id, created_at DESC);

CREATE INDEX idx_admin_audit_logs_resource
ON admin_audit_logs(resource_type, resource_uuid);

CREATE TABLE subscription_plans (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  plan_code VARCHAR(80) NOT NULL UNIQUE,
  name VARCHAR(120) NOT NULL,
  description TEXT,

  price_amount INT NOT NULL,
  currency VARCHAR(10) NOT NULL DEFAULT 'BDT',
  duration_days INT NOT NULL,

  entitlements JSONB NOT NULL DEFAULT '{}'::jsonb,

  sort_order INT NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT subscription_plans_price_check CHECK (price_amount >= 0),
  CONSTRAINT subscription_plans_duration_check CHECK (duration_days > 0)
);

CREATE INDEX idx_subscription_plans_active_sort
ON subscription_plans(is_active, sort_order, id);

CREATE TRIGGER trg_subscription_plans_set_updated_at
BEFORE UPDATE ON subscription_plans
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE manual_payment_requests (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  plan_id BIGINT REFERENCES subscription_plans(id) ON DELETE SET NULL,

  plan_code_snapshot VARCHAR(80) NOT NULL,
  plan_name_snapshot VARCHAR(120) NOT NULL,
  price_amount_snapshot INT NOT NULL,
  currency_snapshot VARCHAR(10) NOT NULL,
  duration_days_snapshot INT NOT NULL,
  entitlements_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

  payment_provider VARCHAR(50) NOT NULL,
  payment_reference VARCHAR(120),
  payer_phone VARCHAR(30),
  note TEXT,

  status VARCHAR(30) NOT NULL DEFAULT 'PENDING',

  submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at TIMESTAMPTZ,
  reviewed_by_admin_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,
  rejection_reason TEXT,

  subscription_id BIGINT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT manual_payment_requests_status_check CHECK (
    status IN ('PENDING', 'UNDER_REVIEW', 'APPROVED', 'REJECTED', 'CANCELED')
  ),
  CONSTRAINT manual_payment_requests_price_check CHECK (price_amount_snapshot >= 0),
  CONSTRAINT manual_payment_requests_duration_check CHECK (duration_days_snapshot > 0)
);

CREATE UNIQUE INDEX manual_payment_requests_reference_unique
ON manual_payment_requests(payment_provider, payment_reference)
WHERE payment_reference IS NOT NULL;

CREATE UNIQUE INDEX manual_payment_requests_one_open_per_user
ON manual_payment_requests(user_id)
WHERE status IN ('PENDING', 'UNDER_REVIEW');

CREATE INDEX idx_manual_payment_requests_user_created
ON manual_payment_requests(user_id, created_at DESC);

CREATE INDEX idx_manual_payment_requests_status_created
ON manual_payment_requests(status, created_at DESC);

CREATE TRIGGER trg_manual_payment_requests_set_updated_at
BEFORE UPDATE ON manual_payment_requests
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_subscriptions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  plan_id BIGINT REFERENCES subscription_plans(id) ON DELETE SET NULL,
  payment_request_id BIGINT REFERENCES manual_payment_requests(id) ON DELETE SET NULL,

  plan_code VARCHAR(80) NOT NULL,
  plan_name VARCHAR(120) NOT NULL,
  source VARCHAR(50) NOT NULL DEFAULT 'MANUAL_PAYMENT',
  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',

  starts_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  canceled_at TIMESTAMPTZ,

  entitlements_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_subscriptions_status_check CHECK (
    status IN ('ACTIVE', 'EXPIRED', 'CANCELED')
  ),
  CONSTRAINT user_subscriptions_time_check CHECK (expires_at > starts_at)
);

CREATE INDEX idx_user_subscriptions_user_status_expires
ON user_subscriptions(user_id, status, expires_at DESC);

CREATE INDEX idx_user_subscriptions_payment_request
ON user_subscriptions(payment_request_id);

CREATE TRIGGER trg_user_subscriptions_set_updated_at
BEFORE UPDATE ON user_subscriptions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

ALTER TABLE manual_payment_requests
ADD CONSTRAINT fk_manual_payment_requests_subscription
FOREIGN KEY (subscription_id)
REFERENCES user_subscriptions(id)
ON DELETE SET NULL;

CREATE TABLE payment_review_actions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  payment_request_id BIGINT NOT NULL REFERENCES manual_payment_requests(id) ON DELETE CASCADE,
  admin_user_id BIGINT REFERENCES admin_users(id) ON DELETE SET NULL,

  action VARCHAR(30) NOT NULL,
  note TEXT,

  before_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
  after_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT payment_review_actions_action_check CHECK (
    action IN ('APPROVED', 'REJECTED')
  )
);

CREATE INDEX idx_payment_review_actions_request_created
ON payment_review_actions(payment_request_id, created_at DESC);

CREATE INDEX idx_payment_review_actions_admin_created
ON payment_review_actions(admin_user_id, created_at DESC);

DROP TABLE IF EXISTS payment_review_actions;

ALTER TABLE manual_payment_requests
DROP CONSTRAINT IF EXISTS fk_manual_payment_requests_subscription;

DROP TRIGGER IF EXISTS trg_user_subscriptions_set_updated_at ON user_subscriptions;
DROP TABLE IF EXISTS user_subscriptions;

DROP TRIGGER IF EXISTS trg_manual_payment_requests_set_updated_at ON manual_payment_requests;
DROP TABLE IF EXISTS manual_payment_requests;

DROP TRIGGER IF EXISTS trg_subscription_plans_set_updated_at ON subscription_plans;
DROP TABLE IF EXISTS subscription_plans;

DROP TABLE IF EXISTS admin_audit_logs;
DROP TABLE IF EXISTS admin_user_permissions;

DROP TRIGGER IF EXISTS trg_admin_sessions_set_updated_at ON admin_sessions;
DROP TABLE IF EXISTS admin_sessions;

DROP TRIGGER IF EXISTS trg_admin_users_set_updated_at ON admin_users;
DROP TABLE IF EXISTS admin_users;

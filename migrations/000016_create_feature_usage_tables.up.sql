CREATE TABLE user_feature_usage (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  feature_key VARCHAR(80) NOT NULL,
  usage_date DATE NOT NULL,

  used_count INT NOT NULL DEFAULT 0,
  used_seconds INT NOT NULL DEFAULT 0,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_feature_usage_unique UNIQUE (
    user_id,
    feature_key,
    usage_date
  ),
  CONSTRAINT user_feature_usage_count_check CHECK (used_count >= 0),
  CONSTRAINT user_feature_usage_seconds_check CHECK (used_seconds >= 0)
);

CREATE INDEX idx_user_feature_usage_user_date
ON user_feature_usage(user_id, usage_date);

CREATE INDEX idx_user_feature_usage_feature_date
ON user_feature_usage(feature_key, usage_date);

CREATE TRIGGER trg_user_feature_usage_set_updated_at
BEFORE UPDATE ON user_feature_usage
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

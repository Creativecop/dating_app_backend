CREATE TABLE discovery_actions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  target_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  action_type VARCHAR(30) NOT NULL,
  client_action_id UUID NOT NULL,
  action_date DATE NOT NULL,

  source VARCHAR(50) NOT NULL DEFAULT 'DISCOVERY_FEED',
  target_distance_meters INT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT discovery_actions_not_self CHECK (
    actor_user_id <> target_user_id
  ),

  CONSTRAINT discovery_actions_type_check CHECK (
    action_type IN ('LIKE', 'PASS', 'SUPER_LIKE')
  ),

  CONSTRAINT discovery_actions_unique_pair UNIQUE (
    actor_user_id,
    target_user_id
  ),

  CONSTRAINT discovery_actions_client_action_unique UNIQUE (
    actor_user_id,
    client_action_id
  )
);

CREATE INDEX idx_discovery_actions_actor
ON discovery_actions(actor_user_id, created_at DESC);

CREATE INDEX idx_discovery_actions_target
ON discovery_actions(target_user_id, created_at DESC);

CREATE INDEX idx_discovery_actions_actor_target
ON discovery_actions(actor_user_id, target_user_id);

CREATE INDEX idx_discovery_actions_super_like_daily
ON discovery_actions(actor_user_id, action_type, action_date);

CREATE TRIGGER trg_discovery_actions_set_updated_at
BEFORE UPDATE ON discovery_actions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE matches (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_low_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_high_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  initiated_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

  status VARCHAR(30) NOT NULL DEFAULT 'ACTIVE',

  matched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unmatched_at TIMESTAMPTZ,
  unmatched_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT matches_user_order_check CHECK (
    user_low_id < user_high_id
  ),

  CONSTRAINT matches_status_check CHECK (
    status IN ('ACTIVE', 'UNMATCHED')
  ),

  CONSTRAINT matches_unique_pair UNIQUE (
    user_low_id,
    user_high_id
  )
);

CREATE INDEX idx_matches_user_low
ON matches(user_low_id);

CREATE INDEX idx_matches_user_high
ON matches(user_high_id);

CREATE INDEX idx_matches_status
ON matches(status);

CREATE TRIGGER trg_matches_set_updated_at
BEFORE UPDATE ON matches
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

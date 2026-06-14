ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

UPDATE profiles
SET completed_at = updated_at
WHERE profile_status = 'ACTIVE'
  AND completed_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_discovery_feed
ON profiles(profile_status, discovery_eligible, completed_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_locations_location
ON user_locations
USING GIST (location);

CREATE TABLE user_blocks (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  blocker_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  blocked_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_blocks_no_self_block CHECK (
    blocker_user_id <> blocked_user_id
  ),

  CONSTRAINT user_blocks_unique_pair UNIQUE (
    blocker_user_id,
    blocked_user_id
  )
);

CREATE INDEX idx_user_blocks_blocker
ON user_blocks(blocker_user_id);

CREATE INDEX idx_user_blocks_blocked
ON user_blocks(blocked_user_id);

CREATE TABLE discovery_impressions (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  viewer_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  candidate_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  source VARCHAR(50) NOT NULL DEFAULT 'DISCOVERY_FEED',

  distance_meters INT,
  rank_position INT,

  shown_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT discovery_impressions_not_self CHECK (
    viewer_user_id <> candidate_user_id
  )
);

CREATE INDEX idx_discovery_impressions_viewer_created
ON discovery_impressions(viewer_user_id, created_at DESC);

CREATE INDEX idx_discovery_impressions_candidate_created
ON discovery_impressions(candidate_user_id, created_at DESC);

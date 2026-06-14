DROP INDEX IF EXISTS idx_discovery_impressions_candidate_created;
DROP INDEX IF EXISTS idx_discovery_impressions_viewer_created;
DROP TABLE IF EXISTS discovery_impressions;

DROP INDEX IF EXISTS idx_user_blocks_blocked;
DROP INDEX IF EXISTS idx_user_blocks_blocker;
DROP TABLE IF EXISTS user_blocks;

DROP INDEX IF EXISTS idx_profiles_discovery_feed;

ALTER TABLE profiles
DROP COLUMN IF EXISTS completed_at;

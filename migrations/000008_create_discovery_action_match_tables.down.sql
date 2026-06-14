DROP TRIGGER IF EXISTS trg_matches_set_updated_at ON matches;
DROP INDEX IF EXISTS idx_matches_status;
DROP INDEX IF EXISTS idx_matches_user_high;
DROP INDEX IF EXISTS idx_matches_user_low;
DROP TABLE IF EXISTS matches;

DROP TRIGGER IF EXISTS trg_discovery_actions_set_updated_at ON discovery_actions;
DROP INDEX IF EXISTS idx_discovery_actions_super_like_daily;
DROP INDEX IF EXISTS idx_discovery_actions_actor_target;
DROP INDEX IF EXISTS idx_discovery_actions_target;
DROP INDEX IF EXISTS idx_discovery_actions_actor;
DROP TABLE IF EXISTS discovery_actions;

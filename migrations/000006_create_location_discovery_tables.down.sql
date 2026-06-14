DROP TRIGGER IF EXISTS trg_discovery_preferences_set_updated_at ON discovery_preferences;
DROP INDEX IF EXISTS idx_discovery_preferences_show_me;
DROP INDEX IF EXISTS idx_discovery_preferences_user_id;
DROP TABLE IF EXISTS discovery_preferences;

DROP TRIGGER IF EXISTS trg_user_locations_set_updated_at ON user_locations;
DROP INDEX IF EXISTS idx_user_locations_last_updated_at;
DROP INDEX IF EXISTS idx_user_locations_source;
DROP INDEX IF EXISTS idx_user_locations_location;
DROP TABLE IF EXISTS user_locations;

ALTER TABLE profiles
DROP COLUMN IF EXISTS discovery_eligibility_updated_at;

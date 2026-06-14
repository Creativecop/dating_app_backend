DROP INDEX IF EXISTS user_media_variants_unique_type;
DROP INDEX IF EXISTS idx_user_media_variants_object_key;
DROP INDEX IF EXISTS idx_user_media_variants_media_id;
DROP TABLE IF EXISTS user_media_variants;

DROP TRIGGER IF EXISTS trg_user_media_set_updated_at ON user_media;
DROP INDEX IF EXISTS user_media_one_active_intro_video;
DROP INDEX IF EXISTS user_media_one_primary_profile_photo;
DROP INDEX IF EXISTS idx_user_media_deleted_at;
DROP INDEX IF EXISTS idx_user_media_moderation_status;
DROP INDEX IF EXISTS idx_user_media_processing_status;
DROP INDEX IF EXISTS idx_user_media_user_purpose_deleted;
DROP INDEX IF EXISTS idx_user_media_uuid;
DROP INDEX IF EXISTS idx_user_media_user_id;
DROP TABLE IF EXISTS user_media;

ALTER TABLE profiles
DROP COLUMN IF EXISTS discovery_eligible;

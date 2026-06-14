ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS discovery_eligible BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE user_media (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  media_type VARCHAR(30) NOT NULL,
  media_purpose VARCHAR(50) NOT NULL,

  processing_status VARCHAR(30) NOT NULL DEFAULT 'UPLOADED',
  moderation_status VARCHAR(30) NOT NULL DEFAULT 'PENDING',

  is_primary BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order INT NOT NULL DEFAULT 0,

  original_file_name TEXT,
  mime_type VARCHAR(120),
  size_bytes BIGINT,

  width INT,
  height INT,
  duration_seconds INT,

  checksum_sha256 TEXT,

  processing_error TEXT,
  rejection_reason TEXT,

  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  processed_at TIMESTAMPTZ,
  failed_at TIMESTAMPTZ,
  approved_at TIMESTAMPTZ,
  deleted_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_media_type_check CHECK (
    media_type IN ('PHOTO', 'VIDEO')
  ),

  CONSTRAINT user_media_purpose_check CHECK (
    media_purpose IN ('PROFILE_PHOTO', 'INTRO_VIDEO')
  ),

  CONSTRAINT user_media_processing_status_check CHECK (
    processing_status IN ('UPLOADED', 'PROCESSING', 'READY', 'FAILED', 'DELETED')
  ),

  CONSTRAINT user_media_moderation_status_check CHECK (
    moderation_status IN ('PENDING', 'APPROVED', 'REJECTED', 'FLAGGED')
  )
);

CREATE INDEX idx_user_media_user_id ON user_media(user_id);
CREATE INDEX idx_user_media_uuid ON user_media(uuid);
CREATE INDEX idx_user_media_user_purpose_deleted
ON user_media(user_id, media_purpose, deleted_at);
CREATE INDEX idx_user_media_processing_status ON user_media(processing_status);
CREATE INDEX idx_user_media_moderation_status ON user_media(moderation_status);
CREATE INDEX idx_user_media_deleted_at ON user_media(deleted_at);

CREATE UNIQUE INDEX user_media_one_primary_profile_photo
ON user_media(user_id)
WHERE media_purpose = 'PROFILE_PHOTO'
  AND is_primary = TRUE
  AND deleted_at IS NULL;

CREATE UNIQUE INDEX user_media_one_active_intro_video
ON user_media(user_id)
WHERE media_purpose = 'INTRO_VIDEO'
  AND deleted_at IS NULL;

CREATE TRIGGER trg_user_media_set_updated_at
BEFORE UPDATE ON user_media
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_media_variants (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  media_id BIGINT NOT NULL REFERENCES user_media(id) ON DELETE CASCADE,

  variant_type VARCHAR(30) NOT NULL,

  storage_provider VARCHAR(50) NOT NULL,
  bucket TEXT,
  object_key TEXT NOT NULL,
  public_url TEXT,

  mime_type VARCHAR(120),
  size_bytes BIGINT,

  width INT,
  height INT,
  duration_seconds INT,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_media_variants_type_check CHECK (
    variant_type IN ('ORIGINAL', 'DISPLAY', 'THUMBNAIL', 'BLURRED', 'TRANSCODED')
  )
);

CREATE INDEX idx_user_media_variants_media_id ON user_media_variants(media_id);
CREATE INDEX idx_user_media_variants_object_key ON user_media_variants(object_key);

CREATE UNIQUE INDEX user_media_variants_unique_type
ON user_media_variants(media_id, variant_type);

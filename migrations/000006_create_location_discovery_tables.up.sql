CREATE EXTENSION IF NOT EXISTS postgis;

ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS discovery_eligibility_updated_at TIMESTAMPTZ;

CREATE TABLE user_locations (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

  latitude DOUBLE PRECISION NOT NULL,
  longitude DOUBLE PRECISION NOT NULL,
  location GEOGRAPHY(POINT, 4326)
    GENERATED ALWAYS AS (
      ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)::geography
    ) STORED,

  accuracy_meters DOUBLE PRECISION,
  is_precise BOOLEAN NOT NULL DEFAULT FALSE,
  location_consent_at TIMESTAMPTZ,

  city VARCHAR(120),
  country VARCHAR(120),
  source VARCHAR(30) NOT NULL DEFAULT 'GPS',

  last_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT user_locations_latitude_check CHECK (
    latitude >= -90 AND latitude <= 90
  ),

  CONSTRAINT user_locations_longitude_check CHECK (
    longitude >= -180 AND longitude <= 180
  ),

  CONSTRAINT user_locations_accuracy_check CHECK (
    accuracy_meters IS NULL OR (accuracy_meters > 0 AND accuracy_meters <= 10000)
  ),

  CONSTRAINT user_locations_source_check CHECK (
    source IN ('GPS', 'MANUAL', 'IP')
  )
);

CREATE INDEX idx_user_locations_location
ON user_locations
USING GIST (location);

CREATE INDEX idx_user_locations_source ON user_locations(source);
CREATE INDEX idx_user_locations_last_updated_at ON user_locations(last_updated_at);

CREATE TRIGGER trg_user_locations_set_updated_at
BEFORE UPDATE ON user_locations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE discovery_preferences (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,

  min_age INT NOT NULL DEFAULT 18,
  max_age INT NOT NULL DEFAULT 40,

  preferred_genders TEXT[] NOT NULL DEFAULT ARRAY['EVERYONE']::TEXT[],

  max_distance_km INT NOT NULL DEFAULT 50,

  verified_only BOOLEAN NOT NULL DEFAULT FALSE,
  show_me_in_discovery BOOLEAN NOT NULL DEFAULT FALSE,
  hide_distance BOOLEAN NOT NULL DEFAULT FALSE,

  is_default BOOLEAN NOT NULL DEFAULT TRUE,
  customized_at TIMESTAMPTZ,
  activated_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT discovery_preferences_age_check CHECK (
    min_age >= 18 AND max_age <= 80 AND min_age <= max_age
  ),

  CONSTRAINT discovery_preferences_distance_check CHECK (
    max_distance_km >= 1 AND max_distance_km <= 200
  ),

  CONSTRAINT discovery_preferences_genders_check CHECK (
    array_length(preferred_genders, 1) IS NOT NULL
  )
);

CREATE INDEX idx_discovery_preferences_user_id ON discovery_preferences(user_id);
CREATE INDEX idx_discovery_preferences_show_me ON discovery_preferences(show_me_in_discovery);

CREATE TRIGGER trg_discovery_preferences_set_updated_at
BEFORE UPDATE ON discovery_preferences
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

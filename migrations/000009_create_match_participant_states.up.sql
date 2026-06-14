ALTER TABLE matches
ADD COLUMN IF NOT EXISTS unmatch_reason_code VARCHAR(50),
ADD COLUMN IF NOT EXISTS unmatch_reason_note TEXT;

CREATE INDEX IF NOT EXISTS idx_matches_active_order
ON matches(status, matched_at DESC, id DESC);

CREATE TABLE match_participants (
  id BIGSERIAL PRIMARY KEY,
  uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),

  match_id BIGINT NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
  user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,

  seen_at TIMESTAMPTZ,
  last_opened_at TIMESTAMPTZ,
  hidden_at TIMESTAMPTZ,

  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  CONSTRAINT match_participants_unique UNIQUE (
    match_id,
    user_id
  )
);

CREATE INDEX idx_match_participants_user
ON match_participants(user_id, created_at DESC);

CREATE INDEX idx_match_participants_match
ON match_participants(match_id);

CREATE INDEX idx_match_participants_user_hidden
ON match_participants(user_id, hidden_at, match_id);

CREATE TRIGGER trg_match_participants_set_updated_at
BEFORE UPDATE ON match_participants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

INSERT INTO match_participants (match_id, user_id)
SELECT id, user_low_id
FROM matches
ON CONFLICT (match_id, user_id) DO NOTHING;

INSERT INTO match_participants (match_id, user_id)
SELECT id, user_high_id
FROM matches
ON CONFLICT (match_id, user_id) DO NOTHING;

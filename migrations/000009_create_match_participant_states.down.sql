DROP TRIGGER IF EXISTS trg_match_participants_set_updated_at ON match_participants;
DROP INDEX IF EXISTS idx_match_participants_user_hidden;
DROP INDEX IF EXISTS idx_match_participants_match;
DROP INDEX IF EXISTS idx_match_participants_user;
DROP TABLE IF EXISTS match_participants;

DROP INDEX IF EXISTS idx_matches_active_order;

ALTER TABLE matches
DROP COLUMN IF EXISTS unmatch_reason_note,
DROP COLUMN IF EXISTS unmatch_reason_code;

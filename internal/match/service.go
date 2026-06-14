package match

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db           *gorm.DB
	repo         *Repository
	blockChecker BlockChecker
}

type BlockChecker interface {
	IsBlockedEitherDirection(ctx context.Context, userAID uint64, userBID uint64) (bool, error)
}

type matchRow struct {
	MatchID          uint64
	MatchUUID        string
	MatchedAt        time.Time
	Status           string
	SeenAt           *time.Time
	LastOpenedAt     *time.Time
	OtherUserID      uint64
	OtherUserUUID    string
	ProfileUUID      *string
	DisplayName      *string
	DateOfBirth      *time.Time
	ShowAge          bool
	Bio              *string
	PrimaryMediaUUID *string
}

type photoRow struct {
	MediaUUID string
}

type promptRow struct {
	Question string
	Answer   string
}

type lifestyleRow struct {
	QuestionKey string
	Question    string
	AnswerJSON  string
}

type normalizedUnmatchRequest struct {
	ReasonCode *string
	ReasonNote *string
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db, repo: NewRepository(db)}
}

func (s *Service) SetBlockChecker(checker BlockChecker) {
	s.blockChecker = checker
}

func (s *Service) List(ctx context.Context, userID uint64, rawLimit string, rawCursor string) (*MatchListResponse, error) {
	limit, err := normalizeListLimit(rawLimit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeListCursor(rawCursor)
	if err != nil {
		return nil, err
	}

	rows, err := s.listRows(ctx, userID, limit+1, cursor)
	if err != nil {
		return nil, err
	}

	response := &MatchListResponse{
		Items: make([]MatchListItem, 0, minInt(limit, len(rows))),
	}
	visibleRows := rows
	if len(rows) > limit {
		visibleRows = rows[:limit]
		last := visibleRows[len(visibleRows)-1]
		next, err := encodeListCursor(listCursor{MatchedAt: last.MatchedAt, MatchID: last.MatchID})
		if err != nil {
			return nil, err
		}
		response.NextCursor = &next
	}
	for _, row := range visibleRows {
		response.Items = append(response.Items, row.toListItem())
	}
	return response, nil
}

func (s *Service) Detail(ctx context.Context, userID uint64, rawMatchUUID string) (*MatchDetailResponse, error) {
	matchUUID, err := parseMatchUUID(rawMatchUUID)
	if err != nil {
		return nil, err
	}
	row, err := s.visibleActiveMatchByUUID(ctx, userID, matchUUID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, notFoundError()
	}

	photos, err := s.photos(ctx, row.OtherUserID)
	if err != nil {
		return nil, err
	}
	interests, err := s.interests(ctx, row.OtherUserID)
	if err != nil {
		return nil, err
	}
	prompts, err := s.prompts(ctx, row.OtherUserID)
	if err != nil {
		return nil, err
	}
	lifestyle, err := s.lifestyleAnswers(ctx, row.OtherUserID)
	if err != nil {
		return nil, err
	}

	return &MatchDetailResponse{
		MatchUUID: row.MatchUUID,
		MatchedAt: row.MatchedAt,
		IsNew:     row.SeenAt == nil,
		User: MatchUserDetail{
			UserUUID:         row.OtherUserUUID,
			ProfileUUID:      row.ProfileUUID,
			DisplayName:      row.DisplayName,
			Age:              row.age(),
			Bio:              row.Bio,
			Photos:           photos,
			Interests:        interests,
			Prompts:          prompts,
			LifestyleAnswers: lifestyle,
		},
	}, nil
}

func (s *Service) MarkSeen(ctx context.Context, userID uint64, rawMatchUUID string) (*MarkSeenResponse, error) {
	matchUUID, err := parseMatchUUID(rawMatchUUID)
	if err != nil {
		return nil, err
	}
	row, err := s.visibleActiveMatchByUUID(ctx, userID, matchUUID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, notFoundError()
	}

	now := time.Now().UTC()
	var updated MatchParticipant
	err = s.db.WithContext(ctx).Raw(`
		UPDATE match_participants
		SET seen_at = COALESCE(seen_at, ?),
		    last_opened_at = ?,
		    updated_at = ?
		WHERE match_id = ?
		  AND user_id = ?
		RETURNING seen_at, last_opened_at
	`, now, now, now, row.MatchID, userID).Scan(&updated).Error
	if err != nil {
		return nil, err
	}

	return &MarkSeenResponse{
		MatchUUID:    row.MatchUUID,
		SeenAt:       updated.SeenAt,
		LastOpenedAt: derefTime(updated.LastOpenedAt, now),
	}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID uint64, rawMatchUUID string, req UnmatchRequest) (*UnmatchResponse, error) {
	matchUUID, err := parseMatchUUID(rawMatchUUID)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeUnmatchRequest(req)
	if err != nil {
		return nil, err
	}

	result := &UnmatchResponse{MatchUUID: matchUUID.String(), Status: StatusUnmatched}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.lockParticipantMatch(ctx, tx, userID, matchUUID)
		if err != nil {
			return err
		}
		if row == nil {
			return notFoundError()
		}
		result.MatchUUID = row.MatchUUID
		if row.Status == StatusUnmatched {
			return nil
		}
		if row.Status != StatusActive {
			return notFoundError()
		}

		now := time.Now().UTC()
		if err := tx.WithContext(ctx).Exec(`
			UPDATE matches
			SET status = 'UNMATCHED',
			    unmatched_at = ?,
			    unmatched_by_user_id = ?,
			    unmatch_reason_code = ?,
			    unmatch_reason_note = ?,
			    updated_at = ?
			WHERE id = ?
		`, now, userID, normalized.ReasonCode, normalized.ReasonNote, now, row.MatchID).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Exec(`
			UPDATE conversations
			SET status = 'CLOSED',
			    updated_at = ?
			WHERE match_id = ?
		`, now, row.MatchID).Error
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) CanViewProfileMedia(ctx context.Context, viewerID uint64, ownerID uint64, _ string, variant string) bool {
	if viewerID == ownerID {
		return true
	}
	if variant != VariantDisplay && variant != VariantThumbnail {
		return false
	}
	if s.blockChecker != nil {
		blocked, err := s.blockChecker.IsBlockedEitherDirection(ctx, viewerID, ownerID)
		if err != nil || blocked {
			return false
		}
	}
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM matches m
		  JOIN match_participants viewer_mp
		    ON viewer_mp.match_id = m.id
		   AND viewer_mp.user_id = ?
		   AND viewer_mp.hidden_at IS NULL
		  JOIN match_participants owner_mp
		    ON owner_mp.match_id = m.id
		   AND owner_mp.user_id = ?
		  JOIN users owner_u
		    ON owner_u.id = ?
		   AND owner_u.status = 'ACTIVE'
		   AND owner_u.deleted_at IS NULL
		  WHERE m.status = 'ACTIVE'
		    AND (
		      (m.user_low_id = ? AND m.user_high_id = ?)
		      OR (m.user_low_id = ? AND m.user_high_id = ?)
		    )
		    AND NOT EXISTS (
		      SELECT 1
		      FROM user_blocks ub
		      WHERE (ub.blocker_user_id = ? AND ub.blocked_user_id = ?)
		         OR (ub.blocker_user_id = ? AND ub.blocked_user_id = ?)
		    )
		)
	`, viewerID, ownerID, ownerID, viewerID, ownerID, ownerID, viewerID, viewerID, ownerID, ownerID, viewerID).Scan(&exists).Error
	return err == nil && exists
}

func (s *Service) listRows(ctx context.Context, userID uint64, limit int, cursor *listCursor) ([]matchRow, error) {
	hasCursor := cursor != nil
	cursorMatchedAt := time.Time{}
	cursorMatchID := uint64(0)
	if cursor != nil {
		cursorMatchedAt = cursor.MatchedAt
		cursorMatchID = cursor.MatchID
	}

	var rows []matchRow
	err := s.db.WithContext(ctx).Raw(baseMatchRowsSQL()+`
  AND (
    ? = FALSE
    OR m.matched_at < ?
    OR (
      m.matched_at = ?
      AND m.id < ?
    )
  )
ORDER BY m.matched_at DESC, m.id DESC
LIMIT ?
`, userID, userID, userID, userID, userID, userID, hasCursor, cursorMatchedAt, cursorMatchedAt, cursorMatchID, limit).Scan(&rows).Error
	return rows, err
}

func (s *Service) visibleActiveMatchByUUID(ctx context.Context, userID uint64, matchUUID uuid.UUID) (*matchRow, error) {
	var row matchRow
	err := s.db.WithContext(ctx).Raw(baseMatchRowsSQL()+`
  AND m.uuid = ?
LIMIT 1
`, userID, userID, userID, userID, userID, userID, matchUUID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.MatchID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) lockParticipantMatch(ctx context.Context, tx *gorm.DB, userID uint64, matchUUID uuid.UUID) (*matchRow, error) {
	var row matchRow
	err := tx.WithContext(ctx).Raw(`
SELECT
  m.id AS match_id,
  m.uuid::text AS match_uuid,
  m.matched_at,
  m.status
FROM matches m
JOIN match_participants mp
  ON mp.match_id = m.id
 AND mp.user_id = ?
 AND mp.hidden_at IS NULL
WHERE m.uuid = ?
  AND (m.user_low_id = ? OR m.user_high_id = ?)
  AND NOT EXISTS (
    SELECT 1
    FROM user_blocks ub
    WHERE (
      ub.blocker_user_id = ?
      AND ub.blocked_user_id = CASE WHEN m.user_low_id = ? THEN m.user_high_id ELSE m.user_low_id END
    )
    OR (
      ub.blocked_user_id = ?
      AND ub.blocker_user_id = CASE WHEN m.user_low_id = ? THEN m.user_high_id ELSE m.user_low_id END
    )
  )
FOR UPDATE OF m
`, userID, matchUUID, userID, userID, userID, userID, userID, userID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.MatchID == 0 {
		return nil, nil
	}
	return &row, nil
}

func baseMatchRowsSQL() string {
	return `
SELECT
  m.id AS match_id,
  m.uuid::text AS match_uuid,
  m.matched_at,
  m.status,
  mp.seen_at,
  mp.last_opened_at,
  other_u.id AS other_user_id,
  other_u.uuid::text AS other_user_uuid,
  p.uuid::text AS profile_uuid,
  p.display_name,
  p.date_of_birth,
  COALESCE(p.show_age, FALSE) AS show_age,
  p.bio,
  pm.uuid::text AS primary_media_uuid
FROM match_participants mp
JOIN matches m ON m.id = mp.match_id
JOIN users other_u
  ON other_u.id = CASE WHEN m.user_low_id = ? THEN m.user_high_id ELSE m.user_low_id END
LEFT JOIN profiles p ON p.user_id = other_u.id
LEFT JOIN user_media pm ON pm.user_id = other_u.id
  AND pm.media_purpose = 'PROFILE_PHOTO'
  AND pm.processing_status = 'READY'
  AND pm.moderation_status = 'APPROVED'
  AND pm.is_primary = TRUE
  AND pm.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM user_media_variants thumbnail_variant
    WHERE thumbnail_variant.media_id = pm.id
      AND thumbnail_variant.variant_type = 'THUMBNAIL'
  )
WHERE mp.user_id = ?
  AND mp.hidden_at IS NULL
  AND m.status = 'ACTIVE'
  AND (m.user_low_id = ? OR m.user_high_id = ?)
  AND other_u.status = 'ACTIVE'
  AND other_u.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM user_blocks ub
    WHERE (
      ub.blocker_user_id = ?
      AND ub.blocked_user_id = other_u.id
    )
    OR (
      ub.blocked_user_id = ?
      AND ub.blocker_user_id = other_u.id
    )
  )
`
}

func (s *Service) photos(ctx context.Context, userID uint64) ([]MatchPhoto, error) {
	var rows []photoRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT m.uuid::text AS media_uuid
		FROM user_media m
		WHERE m.user_id = ?
		  AND m.media_purpose = 'PROFILE_PHOTO'
		  AND m.processing_status = 'READY'
		  AND m.moderation_status = 'APPROVED'
		  AND m.deleted_at IS NULL
		  AND EXISTS (
		    SELECT 1
		    FROM user_media_variants v
		    WHERE v.media_id = m.id
		      AND v.variant_type = 'DISPLAY'
		  )
		  AND EXISTS (
		    SELECT 1
		    FROM user_media_variants v
		    WHERE v.media_id = m.id
		      AND v.variant_type = 'THUMBNAIL'
		  )
		ORDER BY m.is_primary DESC, m.sort_order ASC, m.uploaded_at ASC
	`, userID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	photos := make([]MatchPhoto, 0, len(rows))
	for _, row := range rows {
		photos = append(photos, photoForUUID(row.MediaUUID, true))
	}
	return photos, nil
}

func (s *Service) interests(ctx context.Context, userID uint64) ([]string, error) {
	var payload string
	err := s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(json_agg(i.name ORDER BY i.sort_order ASC, i.name ASC)::text, '[]')
		FROM user_interests ui
		JOIN interests i ON i.id = ui.interest_id AND i.is_active = TRUE
		WHERE ui.user_id = ?
	`, userID).Scan(&payload).Error
	if err != nil {
		return nil, err
	}
	var result []string
	if strings.TrimSpace(payload) == "" {
		return []string{}, nil
	}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) prompts(ctx context.Context, userID uint64) ([]MatchPromptAnswer, error) {
	var rows []promptRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT pqq.question, upp.answer
		FROM user_profile_prompts upp
		JOIN profile_prompt_questions pqq ON pqq.id = upp.prompt_question_id AND pqq.is_active = TRUE
		WHERE upp.user_id = ?
		ORDER BY pqq.sort_order ASC
	`, userID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]MatchPromptAnswer, 0, len(rows))
	for _, row := range rows {
		result = append(result, MatchPromptAnswer{Question: row.Question, Answer: row.Answer})
	}
	return result, nil
}

func (s *Service) lifestyleAnswers(ctx context.Context, userID uint64) ([]MatchLifestyleAnswer, error) {
	var rows []lifestyleRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT lq.question_key, lq.question, ula.answer::text AS answer_json
		FROM user_lifestyle_answers ula
		JOIN lifestyle_questions lq ON lq.id = ula.lifestyle_question_id AND lq.is_active = TRUE
		WHERE ula.user_id = ?
		ORDER BY lq.sort_order ASC
	`, userID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]MatchLifestyleAnswer, 0, len(rows))
	for _, row := range rows {
		var answer any
		if strings.TrimSpace(row.AnswerJSON) != "" {
			if err := json.Unmarshal([]byte(row.AnswerJSON), &answer); err != nil {
				return nil, err
			}
		}
		result = append(result, MatchLifestyleAnswer{
			QuestionKey: row.QuestionKey,
			Question:    row.Question,
			Answer:      answer,
		})
	}
	return result, nil
}

func (row matchRow) toListItem() MatchListItem {
	return MatchListItem{
		MatchUUID: row.MatchUUID,
		MatchedAt: row.MatchedAt,
		IsNew:     row.SeenAt == nil,
		User: MatchUserPreview{
			UserUUID:     row.OtherUserUUID,
			ProfileUUID:  row.ProfileUUID,
			DisplayName:  row.DisplayName,
			Age:          row.age(),
			Bio:          row.Bio,
			PrimaryPhoto: row.primaryPhoto(),
		},
	}
}

func (row matchRow) age() *int {
	if !row.ShowAge || row.DateOfBirth == nil {
		return nil
	}
	value := ageFromDOB(*row.DateOfBirth, time.Now().UTC())
	return &value
}

func (row matchRow) primaryPhoto() *MatchPhoto {
	if row.PrimaryMediaUUID == nil || strings.TrimSpace(*row.PrimaryMediaUUID) == "" {
		return nil
	}
	photo := photoForUUID(*row.PrimaryMediaUUID, false)
	return &photo
}

func photoForUUID(mediaUUID string, includeDisplay bool) MatchPhoto {
	photo := MatchPhoto{
		MediaUUID:    mediaUUID,
		ThumbnailURL: "/api/v1/media/" + mediaUUID + "/thumbnail",
	}
	if includeDisplay {
		photo.DisplayURL = "/api/v1/media/" + mediaUUID + "/display"
	}
	return photo
}

func parseMatchUUID(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.UUID{}, validationError("Match UUID is invalid", map[string]any{"matchUuid": raw})
	}
	return parsed, nil
}

func normalizeUnmatchRequest(req UnmatchRequest) (normalizedUnmatchRequest, error) {
	code, err := normalizeOptionalText(req.ReasonCode, 50, "reasonCode")
	if err != nil {
		return normalizedUnmatchRequest{}, err
	}
	note, err := normalizeOptionalText(req.ReasonNote, 300, "reasonNote")
	if err != nil {
		return normalizedUnmatchRequest{}, err
	}
	return normalizedUnmatchRequest{ReasonCode: code, ReasonNote: note}, nil
}

func normalizeOptionalText(value *string, max int, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > max {
		return nil, validationError(field+" is too long", map[string]any{"field": field, "max": max})
	}
	return &trimmed, nil
}

func ageFromDOB(dob time.Time, now time.Time) int {
	dob = time.Date(dob.UTC().Year(), dob.UTC().Month(), dob.UTC().Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	age := today.Year() - dob.Year()
	if today.Month() < dob.Month() || (today.Month() == dob.Month() && today.Day() < dob.Day()) {
		age--
	}
	return int(math.Max(float64(age), 0))
}

func derefTime(value *time.Time, fallback time.Time) time.Time {
	if value == nil {
		return fallback
	}
	return *value
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

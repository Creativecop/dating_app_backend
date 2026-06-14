package discovery

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"
)

const (
	defaultFeedLimit = 20
	maxFeedLimit     = 50
)

type feedRow struct {
	CandidateUserID   uint64
	CandidateUserUUID string
	ProfileUUID       string
	DisplayName       *string
	DateOfBirth       *time.Time
	ShowAge           bool
	Bio               *string
	HideDistance      bool
	DistanceMeters    int
	CompletedAt       time.Time
	PrimaryMediaUUID  string
	InterestsJSON     string
	PromptsJSON       string
}

type promptJSON struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

func (s *Service) Feed(ctx context.Context, viewerUserID uint64, rawLimit string, rawCursor string) (*FeedResponse, error) {
	readiness, err := s.Readiness(ctx, viewerUserID)
	if err != nil {
		return nil, err
	}
	if !readiness.DiscoveryEligible {
		return nil, discoveryNotReadyError(readinessErrorDetails(readiness))
	}

	limit, err := normalizeFeedLimit(rawLimit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeFeedCursor(rawCursor)
	if err != nil {
		return nil, err
	}

	rows, err := s.feedRows(ctx, viewerUserID, limit+1, cursor)
	if err != nil {
		return nil, err
	}

	response := &FeedResponse{
		Items: make([]DiscoveryProfilePreview, 0, minInt(limit, len(rows))),
	}
	visibleRows := rows
	if len(rows) > limit {
		visibleRows = rows[:limit]
		last := visibleRows[len(visibleRows)-1]
		next, err := encodeFeedCursor(feedCursor{
			DistanceMeters:  last.DistanceMeters,
			CompletedAt:     last.CompletedAt,
			CandidateUserID: last.CandidateUserID,
		})
		if err != nil {
			return nil, err
		}
		response.NextCursor = &next
	}
	for _, row := range visibleRows {
		item, err := row.toPreview()
		if err != nil {
			return nil, err
		}
		response.Items = append(response.Items, item)
	}
	return response, nil
}

func (s *Service) ProfileDetail(ctx context.Context, viewerUserID uint64, targetUserUUID string) (*DiscoveryProfilePreview, error) {
	snapshot, err := s.CheckTargetDiscoverable(ctx, viewerUserID, targetUserUUID, DiscoverabilityModeProfileDetail)
	if err != nil {
		return nil, err
	}
	row, err := s.profileRow(ctx, snapshot.TargetUserID, snapshot.DistanceMeters)
	if err != nil {
		return nil, err
	}
	response, err := row.toPreview()
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) feedRows(ctx context.Context, viewerUserID uint64, limit int, cursor *feedCursor) ([]feedRow, error) {
	freshSince := time.Now().UTC().AddDate(0, 0, -s.cfg.LocationMaxAgeDays)
	query := baseDiscoverableCandidatesSQL(DiscoverabilityModeFeed) + `
SELECT *
FROM candidates
WHERE (
  ? = FALSE
  OR distance_meters > ?
  OR (
    distance_meters = ?
    AND completed_at < ?
  )
  OR (
    distance_meters = ?
    AND completed_at = ?
    AND candidate_user_id > ?
  )
)
ORDER BY distance_meters ASC, completed_at DESC, candidate_user_id ASC
LIMIT ?
`
	hasCursor := cursor != nil
	cursorDistance := 0
	cursorCompletedAt := time.Time{}
	cursorCandidateID := uint64(0)
	if cursor != nil {
		cursorDistance = cursor.DistanceMeters
		cursorCompletedAt = cursor.CompletedAt
		cursorCandidateID = cursor.CandidateUserID
	}

	var rows []feedRow
	err := s.db.WithContext(ctx).Raw(
		query,
		viewerUserID,
		freshSince,
		freshSince,
		hasCursor,
		cursorDistance,
		cursorDistance,
		cursorCompletedAt,
		cursorDistance,
		cursorCompletedAt,
		cursorCandidateID,
		limit,
	).Scan(&rows).Error
	return rows, err
}

func (s *Service) profileRow(ctx context.Context, candidateUserID uint64, distanceMeters int) (feedRow, error) {
	var row feedRow
	err := s.db.WithContext(ctx).Raw(`
SELECT
  cu.id AS candidate_user_id,
  cu.uuid::text AS candidate_user_uuid,
  cp.uuid::text AS profile_uuid,
  cp.display_name,
  cp.date_of_birth,
  cp.show_age,
  cp.bio,
  cdp.hide_distance,
  ?::int AS distance_meters,
  cp.completed_at,
  pm.uuid::text AS primary_media_uuid,
  COALESCE((
    SELECT json_agg(i.name ORDER BY i.sort_order ASC, i.name ASC)::text
    FROM user_interests ui
    JOIN interests i ON i.id = ui.interest_id AND i.is_active = TRUE
    WHERE ui.user_id = cu.id
  ), '[]') AS interests_json,
  COALESCE((
    SELECT json_agg(json_build_object('question', pqq.question, 'answer', upp.answer) ORDER BY pqq.sort_order ASC)::text
    FROM user_profile_prompts upp
    JOIN profile_prompt_questions pqq ON pqq.id = upp.prompt_question_id AND pqq.is_active = TRUE
    WHERE upp.user_id = cu.id
  ), '[]') AS prompts_json
FROM users cu
JOIN profiles cp ON cp.user_id = cu.id
JOIN discovery_preferences cdp ON cdp.user_id = cu.id
JOIN user_media pm ON pm.user_id = cu.id
  AND pm.media_purpose = 'PROFILE_PHOTO'
  AND pm.processing_status = 'READY'
  AND pm.moderation_status = 'APPROVED'
  AND pm.is_primary = TRUE
  AND pm.deleted_at IS NULL
WHERE cu.id = ?
`, distanceMeters, candidateUserID).Scan(&row).Error
	if err != nil {
		return feedRow{}, err
	}
	if row.CandidateUserID == 0 {
		return feedRow{}, targetNotDiscoverableError()
	}
	return row, nil
}

func (row feedRow) toPreview() (DiscoveryProfilePreview, error) {
	interests := make([]string, 0)
	if strings.TrimSpace(row.InterestsJSON) != "" {
		if err := json.Unmarshal([]byte(row.InterestsJSON), &interests); err != nil {
			return DiscoveryProfilePreview{}, err
		}
	}
	prompts := make([]DiscoveryPromptAnswer, 0)
	if strings.TrimSpace(row.PromptsJSON) != "" {
		var decoded []promptJSON
		if err := json.Unmarshal([]byte(row.PromptsJSON), &decoded); err != nil {
			return DiscoveryProfilePreview{}, err
		}
		for _, prompt := range decoded {
			prompts = append(prompts, DiscoveryPromptAnswer{
				Question: prompt.Question,
				Answer:   prompt.Answer,
			})
		}
	}

	var age *int
	if row.ShowAge && row.DateOfBirth != nil {
		value := ageFromDOB(*row.DateOfBirth, time.Now().UTC())
		age = &value
	}

	var distanceKM *int
	if !row.HideDistance {
		value := int(math.Round(float64(row.DistanceMeters) / 1000.0))
		if value < 1 && row.DistanceMeters > 0 {
			value = 1
		}
		distanceKM = &value
	}

	return DiscoveryProfilePreview{
		UserUUID:       row.CandidateUserUUID,
		ProfileUUID:    row.ProfileUUID,
		DisplayName:    row.DisplayName,
		Age:            age,
		DistanceKM:     distanceKM,
		DistanceHidden: row.HideDistance,
		Bio:            row.Bio,
		PrimaryPhoto: DiscoveryPrimaryPhoto{
			MediaUUID:    row.PrimaryMediaUUID,
			DisplayURL:   "/api/v1/media/" + row.PrimaryMediaUUID + "/display",
			ThumbnailURL: "/api/v1/media/" + row.PrimaryMediaUUID + "/thumbnail",
		},
		Interests: interests,
		Prompts:   prompts,
	}, nil
}

func baseDiscoverableCandidatesSQL(mode DiscoverabilityMode) string {
	actionExclusion := ""
	if mode == DiscoverabilityModeFeed {
		actionExclusion = `
    AND NOT EXISTS (
      SELECT 1
      FROM discovery_actions da
      WHERE da.actor_user_id = viewer.viewer_user_id
        AND da.target_user_id = cu.id
    )`
	}
	return `
WITH viewer AS (
  SELECT
    u.id AS viewer_user_id,
    vp.date_of_birth AS viewer_dob,
    vp.gender AS viewer_gender,
    vdp.min_age,
    vdp.max_age,
    vdp.preferred_genders,
    vdp.max_distance_km,
    vul.location AS viewer_location
  FROM users u
  JOIN profiles vp ON vp.user_id = u.id
  JOIN discovery_preferences vdp ON vdp.user_id = u.id
  JOIN user_locations vul ON vul.user_id = u.id
  WHERE u.id = ?
    AND u.status = 'ACTIVE'
    AND u.deleted_at IS NULL
    AND vp.profile_status = 'ACTIVE'
    AND vp.discovery_eligible = TRUE
    AND vp.date_of_birth IS NOT NULL
    AND vp.gender IS NOT NULL
    AND vdp.show_me_in_discovery = TRUE
    AND vul.source IN ('GPS', 'MANUAL')
    AND vul.is_precise = TRUE
    AND vul.location_consent_at IS NOT NULL
    AND vul.last_updated_at >= ?
),
candidates AS (
  SELECT
    cu.id AS candidate_user_id,
    cu.uuid::text AS candidate_user_uuid,
    cp.uuid::text AS profile_uuid,
    cp.display_name,
    cp.date_of_birth,
    cp.show_age,
    cp.gender,
    cp.bio,
    cp.completed_at,
    cdp.hide_distance,
    pm.uuid::text AS primary_media_uuid,
    ST_Distance(cl.location, viewer.viewer_location)::int AS distance_meters,
    COALESCE((
      SELECT json_agg(i.name ORDER BY i.sort_order ASC, i.name ASC)::text
      FROM user_interests ui
      JOIN interests i ON i.id = ui.interest_id AND i.is_active = TRUE
      WHERE ui.user_id = cu.id
    ), '[]') AS interests_json,
    COALESCE((
      SELECT json_agg(json_build_object('question', pqq.question, 'answer', upp.answer) ORDER BY pqq.sort_order ASC)::text
      FROM user_profile_prompts upp
      JOIN profile_prompt_questions pqq ON pqq.id = upp.prompt_question_id AND pqq.is_active = TRUE
      WHERE upp.user_id = cu.id
    ), '[]') AS prompts_json
  FROM viewer
  JOIN user_locations cl ON cl.user_id <> viewer.viewer_user_id
    AND ST_DWithin(cl.location, viewer.viewer_location, viewer.max_distance_km * 1000)
  JOIN users cu ON cu.id = cl.user_id
  JOIN profiles cp ON cp.user_id = cu.id
  JOIN discovery_preferences cdp ON cdp.user_id = cu.id
  JOIN user_media pm ON pm.user_id = cu.id
    AND pm.media_purpose = 'PROFILE_PHOTO'
    AND pm.processing_status = 'READY'
    AND pm.moderation_status = 'APPROVED'
    AND pm.is_primary = TRUE
    AND pm.deleted_at IS NULL
  WHERE cu.status = 'ACTIVE'
    AND cu.deleted_at IS NULL
    AND cp.profile_status = 'ACTIVE'
    AND cp.discovery_eligible = TRUE
    AND cp.completed_at IS NOT NULL
    AND cp.date_of_birth IS NOT NULL
    AND cp.gender IS NOT NULL
    AND cdp.show_me_in_discovery = TRUE
    AND cl.source IN ('GPS', 'MANUAL')
    AND cl.is_precise = TRUE
    AND cl.location_consent_at IS NOT NULL
    AND cl.last_updated_at >= ?
    AND NOT EXISTS (
      SELECT 1
      FROM user_blocks ub
      WHERE (ub.blocker_user_id = viewer.viewer_user_id AND ub.blocked_user_id = cu.id)
         OR (ub.blocker_user_id = cu.id AND ub.blocked_user_id = viewer.viewer_user_id)
    )
    AND EXISTS (
      SELECT 1
      FROM user_media_variants display_variant
      WHERE display_variant.media_id = pm.id
        AND display_variant.variant_type = 'DISPLAY'
    )
    AND EXISTS (
      SELECT 1
      FROM user_media_variants thumbnail_variant
      WHERE thumbnail_variant.media_id = pm.id
        AND thumbnail_variant.variant_type = 'THUMBNAIL'
    )
    AND DATE_PART('year', AGE(CURRENT_DATE, cp.date_of_birth))::int BETWEEN viewer.min_age AND viewer.max_age
    AND DATE_PART('year', AGE(CURRENT_DATE, viewer.viewer_dob))::int BETWEEN cdp.min_age AND cdp.max_age
    AND ('EVERYONE' = ANY(viewer.preferred_genders) OR cp.gender = ANY(viewer.preferred_genders))
    AND ('EVERYONE' = ANY(cdp.preferred_genders) OR viewer.viewer_gender = ANY(cdp.preferred_genders))
` + actionExclusion + `
)
`
}

func normalizeFeedLimit(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultFeedLimit, nil
	}
	value, err := parsePositiveInt(raw)
	if err != nil {
		return 0, validationError("Limit must be a positive integer", map[string]any{"field": "limit"})
	}
	if value > maxFeedLimit {
		value = maxFeedLimit
	}
	return value, nil
}

func parsePositiveInt(raw string) (int, error) {
	var value int
	for _, ch := range strings.TrimSpace(raw) {
		if ch < '0' || ch > '9' {
			return 0, validationError("Invalid integer", nil)
		}
		value = value*10 + int(ch-'0')
	}
	if value < 1 {
		return 0, validationError("Invalid integer", nil)
	}
	return value, nil
}

func ageFromDOB(dob time.Time, now time.Time) int {
	dob = time.Date(dob.UTC().Year(), dob.UTC().Month(), dob.UTC().Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	age := today.Year() - dob.Year()
	if today.Month() < dob.Month() || (today.Month() == dob.Month() && today.Day() < dob.Day()) {
		age--
	}
	return age
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

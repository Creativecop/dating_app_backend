package discovery

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Service) CheckTargetDiscoverable(ctx context.Context, viewerUserID uint64, targetUserUUID string, mode DiscoverabilityMode) (*DiscoverableTargetSnapshot, error) {
	targetUUID, err := uuid.Parse(strings.TrimSpace(targetUserUUID))
	if err != nil {
		return nil, validationError("Target user UUID is invalid", map[string]any{"field": "targetUserUuid"})
	}

	readiness, err := s.Readiness(ctx, viewerUserID)
	if err != nil {
		return nil, err
	}
	if !readiness.DiscoveryEligible {
		return nil, discoveryNotReadyError(readinessErrorDetails(readiness))
	}

	freshSince := time.Now().UTC().AddDate(0, 0, -s.cfg.LocationMaxAgeDays)
	var snapshot DiscoverableTargetSnapshot
	err = s.db.WithContext(ctx).Raw(baseDiscoverableCandidatesSQL(mode)+`
SELECT
  candidate_user_id AS target_user_id,
  candidate_user_uuid AS target_user_uuid,
  profile_uuid AS target_profile_uuid,
  distance_meters
FROM candidates
WHERE candidate_user_uuid = ?
LIMIT 1
`, viewerUserID, freshSince, freshSince, targetUUID.String()).Scan(&snapshot).Error
	if err != nil {
		return nil, err
	}
	if snapshot.TargetUserID == 0 {
		return nil, targetNotDiscoverableError()
	}
	return &snapshot, nil
}

func (s *Service) CanViewProfileMedia(ctx context.Context, viewerID uint64, ownerID uint64, mediaUUID string, variant string) bool {
	if viewerID == ownerID {
		return true
	}
	if variant != "DISPLAY" && variant != "THUMBNAIL" {
		return false
	}
	parsedMediaUUID, err := uuid.Parse(strings.TrimSpace(mediaUUID))
	if err != nil {
		return false
	}

	var ownerUUID string
	err = s.db.WithContext(ctx).Raw(`
		SELECT u.uuid::text
		FROM user_media m
		JOIN users u ON u.id = m.user_id
		WHERE m.user_id = ?
		  AND m.uuid = ?
		  AND m.media_purpose = 'PROFILE_PHOTO'
		  AND m.processing_status = 'READY'
		  AND m.moderation_status = 'APPROVED'
		  AND m.deleted_at IS NULL
	`, ownerID, parsedMediaUUID).Scan(&ownerUUID).Error
	if err != nil || ownerUUID == "" {
		return false
	}
	_, err = s.CheckTargetDiscoverable(ctx, viewerID, ownerUUID, DiscoverabilityModeProfileDetail)
	return err == nil
}

func readinessErrorDetails(readiness *ReadinessResponse) map[string]any {
	return map[string]any{
		"missing": readinessReasonsForError(readiness.Missing),
		"blocked": readinessReasonsForError(readiness.Blocked),
	}
}

func readinessReasonsForError(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		switch value {
		case "approvedPrimaryPhoto":
			result = append(result, "primary_photo")
		case "freshLocation":
			result = append(result, "fresh_location")
		case "showMeInDiscoveryDisabled":
			result = append(result, "show_me_in_discovery_disabled")
		case "userBanned":
			result = append(result, "user_banned")
		case "profilePaused":
			result = append(result, "profile_paused")
		case "profileUnderReview":
			result = append(result, "profile_under_review")
		case "preferences":
			result = append(result, "discovery_preferences")
		default:
			result = append(result, camelToSnake(value))
		}
	}
	return result
}

func camelToSnake(value string) string {
	var builder strings.Builder
	for i, ch := range value {
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 {
				builder.WriteByte('_')
			}
			ch = ch + ('a' - 'A')
		}
		builder.WriteRune(ch)
	}
	return builder.String()
}

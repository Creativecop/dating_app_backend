package safety

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	adminpkg "github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/restriction"
)

const auditResourceReport = "REPORT"

type AdminOperations interface {
	CreateUserRestrictionForUserTx(ctx context.Context, tx *gorm.DB, input adminpkg.CreateUserRestrictionTxInput) (*adminpkg.UserRestrictionResponse, error)
	InsertAuditLogTx(ctx context.Context, tx *gorm.DB, adminID *uint64, actorType string, action string, resourceType string, resourceUUID *uuid.UUID, reason *string, before any, after any, meta adminpkg.RequestMeta) error
	DisconnectRestrictedUser(userID uint64)
}

type adminReportRow struct {
	ID                  uint64
	ReportUUID          string
	CaseUUID            *string
	ReporterUUID        string
	ReportedUserID      *uint64
	ReportedUserUUID    *string
	TargetType          string
	TargetUUID          string
	ReasonCode          string
	ReasonTitle         string
	Note                *string
	EvidenceRaw         string
	Status              string
	Severity            string
	ReviewedAt          *time.Time
	ReviewedByAdminUUID *string
	ReviewReason        *string
	ReviewNote          *string
	ReviewActionType    *string
	ReviewMetadataRaw   string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type normalizedReviewReportRequest struct {
	Decision        string
	Reason          string
	Note            *string
	ActionType      string
	RestrictionType string
	ExpiresAt       *time.Time
}

func (s *Service) AdminListReports(ctx context.Context, query AdminReportListQuery) (*AdminReportListResponse, error) {
	limit, err := normalizeAdminReportLimit(query.Limit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeAdminReportCursor(query.Cursor)
	if err != nil {
		return nil, err
	}

	where := []string{"1 = 1"}
	args := make([]any, 0)
	if strings.TrimSpace(query.Status) != "" {
		status := strings.ToUpper(strings.TrimSpace(query.Status))
		if !validReportStatus(status) {
			return nil, validationError("status is invalid", map[string]any{"field": "status"})
		}
		where = append(where, "r.status = ?")
		args = append(args, status)
	}
	if strings.TrimSpace(query.TargetType) != "" {
		targetType := strings.ToUpper(strings.TrimSpace(query.TargetType))
		if !validTargetType(targetType) {
			return nil, validationError("targetType is invalid", map[string]any{"field": "targetType"})
		}
		where = append(where, "r.target_type = ?")
		args = append(args, targetType)
	}
	if strings.TrimSpace(query.Severity) != "" {
		severity := strings.ToUpper(strings.TrimSpace(query.Severity))
		if !validSeverity(severity) {
			return nil, validationError("severity is invalid", map[string]any{"field": "severity"})
		}
		where = append(where, "r.severity = ?")
		args = append(args, severity)
	}
	if strings.TrimSpace(query.CreatedFrom) != "" {
		createdFrom, err := time.Parse(time.RFC3339, strings.TrimSpace(query.CreatedFrom))
		if err != nil {
			return nil, validationError("createdFrom must be RFC3339", map[string]any{"field": "createdFrom"})
		}
		where = append(where, "r.created_at >= ?")
		args = append(args, createdFrom)
	}
	if strings.TrimSpace(query.CreatedTo) != "" {
		createdTo, err := time.Parse(time.RFC3339, strings.TrimSpace(query.CreatedTo))
		if err != nil {
			return nil, validationError("createdTo must be RFC3339", map[string]any{"field": "createdTo"})
		}
		where = append(where, "r.created_at <= ?")
		args = append(args, createdTo)
	}
	if cursor != nil {
		where = append(where, "(r.created_at < ? OR (r.created_at = ? AND r.id < ?))")
		args = append(args, cursor.CreatedAt, cursor.CreatedAt, cursor.ReportID)
	}
	args = append(args, limit+1)

	rows, err := s.adminReportRows(ctx, strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, err
	}
	response := &AdminReportListResponse{Items: make([]AdminReportListItem, 0, minInt(limit, len(rows)))}
	for i, row := range rows {
		if i == limit {
			nextCursor, err := encodeAdminReportCursor(adminReportCursor{CreatedAt: row.CreatedAt, ReportID: row.ID})
			if err != nil {
				return nil, err
			}
			response.NextCursor = &nextCursor
			break
		}
		response.Items = append(response.Items, row.toListItem())
	}
	return response, nil
}

func (s *Service) AdminReportDetail(ctx context.Context, rawReportUUID string) (*AdminReportDetailResponse, error) {
	reportUUID, err := parseUUID(rawReportUUID, "reportId")
	if err != nil {
		return nil, err
	}
	row, err := s.adminReportByUUIDTx(ctx, s.db, reportUUID, false)
	if err != nil {
		return nil, err
	}
	response, err := row.toDetailResponse()
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) AdminReviewReport(ctx context.Context, adminID uint64, rawReportUUID string, req ReviewReportRequest, meta adminpkg.RequestMeta) (*AdminReportDetailResponse, error) {
	if s.adminOps == nil {
		return nil, reportActionNotAllowedError("Admin operations are not available")
	}
	reportUUID, err := parseUUID(rawReportUUID, "reportId")
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeReviewReportRequest(req, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var response AdminReportDetailResponse
	var disconnectUserID uint64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.adminReportByUUIDTx(ctx, tx, reportUUID, true)
		if err != nil {
			return err
		}
		if row.Status != ReportPending {
			return reportAlreadyReviewedError()
		}

		before, err := row.toSnapshot()
		if err != nil {
			return err
		}
		actionMetadata := map[string]any{}
		if normalized.ActionType == ReportActionRestrictUser {
			if row.ReportedUserID == nil {
				return reportActionNotAllowedError("Report has no target user to restrict")
			}
			created, err := s.adminOps.CreateUserRestrictionForUserTx(ctx, tx, adminpkg.CreateUserRestrictionTxInput{
				ActorID:         adminID,
				UserID:          *row.ReportedUserID,
				RestrictionType: normalized.RestrictionType,
				Reason:          normalized.Reason,
				ExpiresAt:       normalized.ExpiresAt,
				Meta:            meta,
				Now:             now,
			})
			if err != nil {
				return err
			}
			actionMetadata["restrictionUuid"] = created.RestrictionUUID
			actionMetadata["restrictionType"] = created.RestrictionType
			actionMetadata["expiresAt"] = created.ExpiresAt
			if normalized.RestrictionType == restriction.TypeFullPlatformBan {
				disconnectUserID = *row.ReportedUserID
			}
		}

		reviewMetadata := map[string]any{
			"decision": normalized.Decision,
			"action":   normalized.ActionType,
		}
		for key, value := range actionMetadata {
			reviewMetadata[key] = value
		}
		metadataJSON, err := json.Marshal(reviewMetadata)
		if err != nil {
			return err
		}

		if err := tx.WithContext(ctx).Model(&Report{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":                    normalized.Decision,
			"reviewed_at":               now,
			"reviewed_by_admin_user_id": adminID,
			"review_reason":             normalized.Reason,
			"review_note":               normalized.Note,
			"review_action_type":        normalized.ActionType,
			"review_metadata":           datatypes.JSON(metadataJSON),
			"updated_at":                now,
		}).Error; err != nil {
			return err
		}

		updated, err := s.adminReportByUUIDTx(ctx, tx, reportUUID, false)
		if err != nil {
			return err
		}
		after, err := updated.toSnapshot()
		if err != nil {
			return err
		}
		if err := s.adminOps.InsertAuditLogTx(ctx, tx, &adminID, adminpkg.AuditActorAdmin, "REPORT_REVIEWED", auditResourceReport, &reportUUID, &normalized.Reason, before, after, meta); err != nil {
			return err
		}
		response, err = updated.toDetailResponse()
		return err
	})
	if err != nil {
		return nil, err
	}
	if disconnectUserID != 0 {
		s.adminOps.DisconnectRestrictedUser(disconnectUserID)
	}
	return &response, nil
}

func (s *Service) adminReportRows(ctx context.Context, where string, args ...any) ([]adminReportRow, error) {
	query := adminReportSelectSQL() + `
		WHERE ` + where + `
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT ?
	`
	var rows []adminReportRow
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) adminReportByUUIDTx(ctx context.Context, tx *gorm.DB, reportUUID uuid.UUID, lock bool) (adminReportRow, error) {
	lockClause := ""
	if lock {
		lockClause = " FOR UPDATE OF r"
	}
	var rows []adminReportRow
	if err := tx.WithContext(ctx).Raw(adminReportSelectSQL()+`
		WHERE r.uuid = ?
		LIMIT 1`+lockClause+`
	`, reportUUID).Scan(&rows).Error; err != nil {
		return adminReportRow{}, err
	}
	if len(rows) == 0 {
		return adminReportRow{}, reportNotFoundError()
	}
	return rows[0], nil
}

func adminReportSelectSQL() string {
	return `
		SELECT
		  r.id,
		  r.uuid::text AS report_uuid,
		  mc.uuid::text AS case_uuid,
		  reporter.uuid::text AS reporter_uuid,
		  r.reported_user_id,
		  reported.uuid::text AS reported_user_uuid,
		  r.target_type,
		  r.target_uuid::text AS target_uuid,
		  rr.reason_code,
		  rr.title AS reason_title,
		  r.note,
		  r.evidence_snapshot::text AS evidence_raw,
		  r.status,
		  r.severity,
		  r.reviewed_at,
		  reviewer.uuid::text AS reviewed_by_admin_uuid,
		  r.review_reason,
		  r.review_note,
		  r.review_action_type,
		  r.review_metadata::text AS review_metadata_raw,
		  r.created_at,
		  r.updated_at
		FROM reports r
		JOIN users reporter ON reporter.id = r.reporter_user_id
		LEFT JOIN users reported ON reported.id = r.reported_user_id
		JOIN report_reasons rr ON rr.id = r.reason_id
		LEFT JOIN moderation_cases mc ON mc.id = r.moderation_case_id
		LEFT JOIN admin_users reviewer ON reviewer.id = r.reviewed_by_admin_user_id
	`
}

func normalizeReviewReportRequest(req ReviewReportRequest, now time.Time) (normalizedReviewReportRequest, error) {
	decision := strings.ToUpper(strings.TrimSpace(req.Decision))
	if !validFinalReportDecision(decision) {
		return normalizedReviewReportRequest{}, validationError("decision is invalid", map[string]any{"field": "decision"})
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return normalizedReviewReportRequest{}, validationError("reason is required", map[string]any{"field": "reason"})
	}
	if len(reason) > 1000 {
		return normalizedReviewReportRequest{}, validationError("reason must be 1000 characters or less", map[string]any{"field": "reason"})
	}
	note, err := normalizeNote(req.Note)
	if err != nil {
		return normalizedReviewReportRequest{}, err
	}
	actionType := ReportActionNone
	restrictionType := ""
	var expiresAt *time.Time
	if req.Action != nil {
		actionType = strings.ToUpper(strings.TrimSpace(req.Action.Type))
		if actionType == "" {
			actionType = ReportActionNone
		}
		restrictionType = strings.ToUpper(strings.TrimSpace(req.Action.RestrictionType))
		expiresAt = req.Action.ExpiresAt
	}
	if actionType == ReportActionHideComment || actionType == ReportActionForceEndLive || actionType == ReportActionHideChatMessage {
		return normalizedReviewReportRequest{}, reportActionNotAllowedError("Report action is not available in this deployment")
	}
	switch decision {
	case ReportReviewed, ReportDismissed:
		if actionType != ReportActionNone {
			return normalizedReviewReportRequest{}, reportActionNotAllowedError("Review decision does not allow enforcement action")
		}
	case ReportActioned:
		if actionType != ReportActionRestrictUser {
			return normalizedReviewReportRequest{}, reportActionNotAllowedError("ACTIONED requires a supported enforcement action")
		}
		if !restriction.IsValidRestrictionType(restrictionType) {
			return normalizedReviewReportRequest{}, validationError("restrictionType is invalid", map[string]any{"field": "action.restrictionType"})
		}
	default:
		return normalizedReviewReportRequest{}, validationError("decision is invalid", map[string]any{"field": "decision"})
	}
	if expiresAt != nil && !expiresAt.After(now) {
		return normalizedReviewReportRequest{}, validationError("expiresAt must be in the future", map[string]any{"field": "action.expiresAt"})
	}
	return normalizedReviewReportRequest{
		Decision:        decision,
		Reason:          reason,
		Note:            note,
		ActionType:      actionType,
		RestrictionType: restrictionType,
		ExpiresAt:       expiresAt,
	}, nil
}

func (row adminReportRow) toListItem() AdminReportListItem {
	return AdminReportListItem{
		ReportUUID:       row.ReportUUID,
		CaseUUID:         row.CaseUUID,
		ReporterUUID:     row.ReporterUUID,
		ReportedUserUUID: row.ReportedUserUUID,
		TargetType:       row.TargetType,
		TargetUUID:       row.TargetUUID,
		ReasonCode:       row.ReasonCode,
		Status:           row.Status,
		Severity:         row.Severity,
		ReviewedAt:       row.ReviewedAt,
		CreatedAt:        row.CreatedAt,
	}
}

func (row adminReportRow) toDetailResponse() (AdminReportDetailResponse, error) {
	evidence, err := decodeJSONMap(row.EvidenceRaw)
	if err != nil {
		return AdminReportDetailResponse{}, err
	}
	metadata, err := decodeJSONMap(row.ReviewMetadataRaw)
	if err != nil {
		return AdminReportDetailResponse{}, err
	}
	return AdminReportDetailResponse{
		ReportUUID:          row.ReportUUID,
		CaseUUID:            row.CaseUUID,
		ReporterUUID:        row.ReporterUUID,
		ReportedUserUUID:    row.ReportedUserUUID,
		TargetType:          row.TargetType,
		TargetUUID:          row.TargetUUID,
		ReasonCode:          row.ReasonCode,
		ReasonTitle:         row.ReasonTitle,
		Note:                row.Note,
		EvidenceSnapshot:    evidence,
		Status:              row.Status,
		Severity:            row.Severity,
		ReviewedAt:          row.ReviewedAt,
		ReviewedByAdminUUID: row.ReviewedByAdminUUID,
		ReviewReason:        row.ReviewReason,
		ReviewNote:          row.ReviewNote,
		ReviewActionType:    row.ReviewActionType,
		ReviewMetadata:      metadata,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func (row adminReportRow) toSnapshot() (map[string]any, error) {
	evidence, err := decodeJSONMap(row.EvidenceRaw)
	if err != nil {
		return nil, err
	}
	metadata, err := decodeJSONMap(row.ReviewMetadataRaw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"reportUuid":          row.ReportUUID,
		"status":              row.Status,
		"targetType":          row.TargetType,
		"targetUuid":          row.TargetUUID,
		"reportedUserUuid":    row.ReportedUserUUID,
		"reasonCode":          row.ReasonCode,
		"severity":            row.Severity,
		"reviewedAt":          row.ReviewedAt,
		"reviewedByAdminUuid": row.ReviewedByAdminUUID,
		"reviewReason":        row.ReviewReason,
		"reviewNote":          row.ReviewNote,
		"reviewActionType":    row.ReviewActionType,
		"reviewMetadata":      metadata,
		"evidenceSnapshot":    evidence,
	}, nil
}

func decodeJSONMap(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, err
	}
	if decoded == nil {
		return map[string]any{}, nil
	}
	return decoded, nil
}

func parseUUID(raw string, field string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, validationError(field+" is invalid", map[string]any{"field": field})
	}
	return parsed, nil
}

func validReportStatus(status string) bool {
	return status == ReportPending || status == ReportReviewed || status == ReportDismissed || status == ReportActioned
}

func validFinalReportDecision(decision string) bool {
	return decision == ReportReviewed || decision == ReportDismissed || decision == ReportActioned
}

func validSeverity(severity string) bool {
	return severity == SeverityLow || severity == SeverityMedium || severity == SeverityHigh || severity == SeverityCritical
}

func reportNotFoundError() *ServiceError {
	return &ServiceError{Status: 404, Code: CodeReportNotFound, Message: "Report not found"}
}

func reportAlreadyReviewedError() *ServiceError {
	return &ServiceError{Status: 409, Code: CodeReportAlreadyReviewed, Message: "Report is already reviewed"}
}

func reportActionNotAllowedError(message string) *ServiceError {
	return &ServiceError{Status: 409, Code: CodeReportActionNotAllowed, Message: message}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

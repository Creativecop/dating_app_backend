package admin

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type auditLogRow struct {
	ID                uint64
	AuditLogUUID      string
	AdminUserUUID     *string
	AdminEmail        *string
	ActorType         string
	Action            string
	ResourceType      string
	ResourceUUID      *string
	Reason            *string
	BeforeSnapshotRaw string
	AfterSnapshotRaw  string
	RequestID         *string
	IPAddress         *string
	UserAgent         *string
	CreatedAt         time.Time
}

func (s *Service) ListAuditLogs(ctx context.Context, query AuditLogListQuery) (*AuditLogListResponse, error) {
	normalized, err := normalizeAuditLogListQuery(query)
	if err != nil {
		return nil, err
	}
	query = normalized
	limit, err := normalizeAuditLogLimit(query.Limit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeAuditLogCursor(query.Cursor)
	if err != nil {
		return nil, err
	}

	where := []string{"1 = 1"}
	args := make([]any, 0)
	var createdFromValue *time.Time
	var createdToValue *time.Time
	if strings.TrimSpace(query.AdminUserUUID) != "" {
		adminUUID, err := uuid.Parse(strings.TrimSpace(query.AdminUserUUID))
		if err != nil {
			return nil, validationError("adminUserUuid is invalid", map[string]any{"field": "adminUserUuid"})
		}
		where = append(where, "au.uuid = ?")
		args = append(args, adminUUID)
	}
	if strings.TrimSpace(query.Action) != "" {
		where = append(where, "al.action = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(query.Action)))
	}
	if strings.TrimSpace(query.ResourceType) != "" {
		where = append(where, "al.resource_type = ?")
		args = append(args, strings.ToUpper(strings.TrimSpace(query.ResourceType)))
	}
	if strings.TrimSpace(query.ResourceUUID) != "" {
		resourceUUID, err := uuid.Parse(strings.TrimSpace(query.ResourceUUID))
		if err != nil {
			return nil, validationError("resourceUuid is invalid", map[string]any{"field": "resourceUuid"})
		}
		where = append(where, "al.resource_uuid = ?")
		args = append(args, resourceUUID)
	}
	if strings.TrimSpace(query.CreatedFrom) != "" {
		createdFrom, err := time.Parse(time.RFC3339, strings.TrimSpace(query.CreatedFrom))
		if err != nil {
			return nil, validationError("createdFrom must be RFC3339", map[string]any{"field": "createdFrom"})
		}
		createdFromValue = &createdFrom
		where = append(where, "al.created_at >= ?")
		args = append(args, createdFrom)
	}
	if strings.TrimSpace(query.CreatedTo) != "" {
		createdTo, err := time.Parse(time.RFC3339, strings.TrimSpace(query.CreatedTo))
		if err != nil {
			return nil, validationError("createdTo must be RFC3339", map[string]any{"field": "createdTo"})
		}
		createdToValue = &createdTo
		where = append(where, "al.created_at <= ?")
		args = append(args, createdTo)
	}
	if err := validateRFC3339DateRange(createdFromValue, createdToValue); err != nil {
		return nil, err
	}
	if cursor != nil {
		where = append(where, "(al.created_at < ? OR (al.created_at = ? AND al.id < ?))")
		args = append(args, cursor.CreatedAt, cursor.CreatedAt, cursor.AuditLogID)
	}
	args = append(args, limit+1)

	rows, err := s.auditLogRows(ctx, strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, err
	}
	response := &AuditLogListResponse{Items: make([]AuditLogResponse, 0, minInt(limit, len(rows)))}
	for i, row := range rows {
		if i == limit {
			nextCursor, err := encodeAuditLogCursor(auditLogCursor{CreatedAt: row.CreatedAt, AuditLogID: row.ID})
			if err != nil {
				return nil, err
			}
			response.NextCursor = &nextCursor
			break
		}
		item, err := row.toResponse()
		if err != nil {
			return nil, err
		}
		response.Items = append(response.Items, item)
	}
	return response, nil
}

func (s *Service) AuditLogDetail(ctx context.Context, auditLogUUID string) (*AuditLogResponse, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(auditLogUUID))
	if err != nil {
		return nil, validationError("auditLogUuid is invalid", map[string]any{"field": "auditLogUuid"})
	}
	rows, err := s.auditLogRows(ctx, "al.uuid = ?", parsed, 1)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, &ServiceError{Status: 404, Code: CodeAdminAuditLogNotFound, Message: "Audit log not found"}
	}
	response, err := rows[0].toResponse()
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func normalizeAuditLogListQuery(query AuditLogListQuery) (AuditLogListQuery, error) {
	adminUserUUID, err := mergeAuditFilter("adminUserUuid", query.AdminUserUUID, "actorAdminUserId", query.ActorAdminUserID)
	if err != nil {
		return AuditLogListQuery{}, err
	}
	adminUserUUID, err = mergeAuditFilter("adminUserUuid", adminUserUUID, "actorAdminUserUuid", query.ActorAdminUserUUID)
	if err != nil {
		return AuditLogListQuery{}, err
	}
	action, err := mergeAuditFilter("action", query.Action, "actionType", query.ActionType)
	if err != nil {
		return AuditLogListQuery{}, err
	}
	createdFrom, err := mergeAuditFilter("createdFrom", query.CreatedFrom, "from", query.From)
	if err != nil {
		return AuditLogListQuery{}, err
	}
	createdTo, err := mergeAuditFilter("createdTo", query.CreatedTo, "to", query.To)
	if err != nil {
		return AuditLogListQuery{}, err
	}
	query.AdminUserUUID = adminUserUUID
	query.Action = action
	query.CreatedFrom = createdFrom
	query.CreatedTo = createdTo
	return query, nil
}

func mergeAuditFilter(canonicalName string, canonicalValue string, aliasName string, aliasValue string) (string, error) {
	canonicalValue = strings.TrimSpace(canonicalValue)
	aliasValue = strings.TrimSpace(aliasValue)
	if canonicalValue == "" {
		return aliasValue, nil
	}
	if aliasValue == "" || canonicalValue == aliasValue {
		return canonicalValue, nil
	}
	return "", &ServiceError{
		Status:  400,
		Code:    CodeAdminAuditFilterConflict,
		Message: "Conflicting audit log filters",
		Details: map[string]any{
			"canonical": canonicalName,
			"alias":     aliasName,
		},
	}
}

func (s *Service) auditLogRows(ctx context.Context, where string, args ...any) ([]auditLogRow, error) {
	query := `
		SELECT
		  al.id,
		  al.uuid::text AS audit_log_uuid,
		  au.uuid::text AS admin_user_uuid,
		  au.email AS admin_email,
		  al.actor_type,
		  al.action,
		  al.resource_type,
		  al.resource_uuid::text AS resource_uuid,
		  al.reason,
		  al.before_snapshot::text AS before_snapshot_raw,
		  al.after_snapshot::text AS after_snapshot_raw,
		  al.request_id,
		  al.ip_address,
		  al.user_agent,
		  al.created_at
		FROM admin_audit_logs al
		LEFT JOIN admin_users au ON au.id = al.admin_user_id
		WHERE ` + where + `
		ORDER BY al.created_at DESC, al.id DESC
		LIMIT ?
	`
	var rows []auditLogRow
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (row auditLogRow) toResponse() (AuditLogResponse, error) {
	before := json.RawMessage(row.BeforeSnapshotRaw)
	if len(before) == 0 {
		before = json.RawMessage(`{}`)
	}
	after := json.RawMessage(row.AfterSnapshotRaw)
	if len(after) == 0 {
		after = json.RawMessage(`{}`)
	}
	return AuditLogResponse{
		AuditLogUUID:   row.AuditLogUUID,
		AdminUserUUID:  row.AdminUserUUID,
		AdminEmail:     row.AdminEmail,
		ActorType:      row.ActorType,
		Action:         row.Action,
		ResourceType:   row.ResourceType,
		ResourceUUID:   row.ResourceUUID,
		Reason:         row.Reason,
		BeforeSnapshot: before,
		AfterSnapshot:  after,
		RequestID:      row.RequestID,
		IPAddress:      row.IPAddress,
		UserAgent:      row.UserAgent,
		CreatedAt:      row.CreatedAt,
	}, nil
}

func insertAdminAuditLogTx(ctx context.Context, tx *gorm.DB, adminID *uint64, actorType string, action string, resourceType string, resourceUUID *uuid.UUID, reason *string, before any, after any, meta RequestMeta) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(`
		INSERT INTO admin_audit_logs (
		  admin_user_id,
		  actor_type,
		  action,
		  resource_type,
		  resource_uuid,
		  reason,
		  before_snapshot,
		  after_snapshot,
		  request_id,
		  ip_address,
		  user_agent
		)
		VALUES (?, ?, ?, ?, ?, ?, ?::jsonb, ?::jsonb, ?, ?, ?)
	`, adminID, actorType, action, resourceType, resourceUUID, reason, string(beforeJSON), string(afterJSON), nullableString(meta.RequestID), meta.IPAddress, meta.UserAgent).Error
}

func (s *Service) InsertAuditLogTx(ctx context.Context, tx *gorm.DB, adminID *uint64, actorType string, action string, resourceType string, resourceUUID *uuid.UUID, reason *string, before any, after any, meta RequestMeta) error {
	_ = s
	return insertAdminAuditLogTx(ctx, tx, adminID, actorType, action, resourceType, resourceUUID, reason, before, after, meta)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

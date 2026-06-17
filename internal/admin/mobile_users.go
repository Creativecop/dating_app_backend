package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neoscoder/aura-backend/internal/restriction"
)

const (
	auditResourceUser = "USER"
	recentReportLimit = 10
	userAuditLimit    = 10
)

type adminMobileUserRow struct {
	UserID                 uint64
	UserUUID               string
	Phone                  *string
	Email                  *string
	Status                 string
	OnboardingStatus       string
	LastLoginAt            *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
	ProfileUUID            *string
	DisplayName            *string
	Gender                 *string
	City                   *string
	Country                *string
	ProfileStatus          *string
	CompletedAt            *time.Time
	ActiveRestrictionCount int
}

type userRestrictionRow struct {
	ID                     uint64
	RestrictionUUID        string
	RestrictionType        string
	Status                 string
	Reason                 string
	CreatedByAdminUserUUID *string
	CreatedByAdminEmail    *string
	RevokedByAdminUserUUID *string
	RevokedByAdminEmail    *string
	RevokedAt              *time.Time
	RevocationReason       *string
	ExpiresAt              *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type recentReportRow struct {
	ReportUUID string
	TargetType string
	ReasonCode string
	Status     string
	Severity   string
	CreatedAt  time.Time
}

func (s *Service) ListUsers(ctx context.Context, query AdminMobileUserListQuery) (*AdminMobileUserListResponse, error) {
	limit, err := normalizeAuditLogLimit(query.Limit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeAdminUserListCursor(query.Cursor)
	if err != nil {
		return nil, err
	}

	where := []string{"1 = 1"}
	args := make([]any, 0)
	search := strings.TrimSpace(query.Search)
	if search != "" {
		where = append(where, `(u.uuid::text ILIKE ? OR u.phone ILIKE ? OR u.email::text ILIKE ? OR p.display_name ILIKE ?)`)
		pattern := "%" + search + "%"
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if strings.TrimSpace(query.Status) != "" {
		status := strings.ToUpper(strings.TrimSpace(query.Status))
		if !isValidMobileUserStatus(status) {
			return nil, validationError("status is invalid", map[string]any{"field": "status"})
		}
		where = append(where, "u.status = ?")
		args = append(args, status)
	}
	if strings.TrimSpace(query.CreatedFrom) != "" {
		createdFrom, err := time.Parse(time.RFC3339, strings.TrimSpace(query.CreatedFrom))
		if err != nil {
			return nil, validationError("createdFrom must be RFC3339", map[string]any{"field": "createdFrom"})
		}
		where = append(where, "u.created_at >= ?")
		args = append(args, createdFrom)
	}
	if strings.TrimSpace(query.CreatedTo) != "" {
		createdTo, err := time.Parse(time.RFC3339, strings.TrimSpace(query.CreatedTo))
		if err != nil {
			return nil, validationError("createdTo must be RFC3339", map[string]any{"field": "createdTo"})
		}
		where = append(where, "u.created_at <= ?")
		args = append(args, createdTo)
	}
	if cursor != nil {
		where = append(where, "(u.created_at < ? OR (u.created_at = ? AND u.id < ?))")
		args = append(args, cursor.CreatedAt, cursor.CreatedAt, cursor.UserID)
	}
	args = append(args, limit+1)

	rows, err := s.mobileUserRows(ctx, strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, err
	}
	response := &AdminMobileUserListResponse{Items: make([]AdminMobileUserListItem, 0, minInt(limit, len(rows)))}
	for i, row := range rows {
		if i == limit {
			nextCursor, err := encodeAdminUserListCursor(adminUserListCursor{CreatedAt: row.CreatedAt, UserID: row.UserID})
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

func (s *Service) UserDetail(ctx context.Context, rawUserUUID string) (*AdminMobileUserDetailResponse, error) {
	userUUID, err := parseUUID(rawUserUUID, "userId")
	if err != nil {
		return nil, err
	}
	rows, err := s.mobileUserRows(ctx, "u.uuid = ?", userUUID, 1)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, userNotFoundError()
	}
	row := rows[0]
	restrictions, err := s.userRestrictionRows(ctx, row.UserID, restriction.StatusActive)
	if err != nil {
		return nil, err
	}
	reports, err := s.recentReports(ctx, row.UserID)
	if err != nil {
		return nil, err
	}
	auditRows, err := s.auditLogRows(ctx, "al.resource_type = ? AND al.resource_uuid = ?", auditResourceUser, userUUID, userAuditLimit)
	if err != nil {
		return nil, err
	}
	auditHistory := make([]AuditLogResponse, 0, len(auditRows))
	for _, auditRow := range auditRows {
		item, err := auditRow.toResponse()
		if err != nil {
			return nil, err
		}
		auditHistory = append(auditHistory, item)
	}
	response := row.toDetailResponse()
	response.Restrictions = restrictions
	response.RecentReports = reports
	response.AuditHistory = auditHistory
	return &response, nil
}

func (s *Service) ListUserRestrictions(ctx context.Context, rawUserUUID string, rawStatus string) (*UserRestrictionListResponse, error) {
	user, err := s.mobileUserByUUID(ctx, rawUserUUID)
	if err != nil {
		return nil, err
	}
	status := normalizeRestrictionStatus(rawStatus)
	if !restriction.IsValidStatus(status) {
		return nil, validationError("status is invalid", map[string]any{"field": "status"})
	}
	rows, err := s.userRestrictionRows(ctx, user.ID, status)
	if err != nil {
		return nil, err
	}
	return &UserRestrictionListResponse{Items: rows}, nil
}

func (s *Service) CreateUserRestriction(ctx context.Context, actorID uint64, rawUserUUID string, req CreateUserRestrictionRequest, meta RequestMeta) (*UserRestrictionResponse, error) {
	restrictionType := strings.ToUpper(strings.TrimSpace(req.RestrictionType))
	if !restriction.IsValidRestrictionType(restrictionType) {
		return nil, validationError("restrictionType is invalid", map[string]any{"field": "restrictionType"})
	}
	reason, err := requireRestrictionReason(req.Reason)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if req.ExpiresAt != nil && !req.ExpiresAt.After(now) {
		return nil, &ServiceError{Status: 409, Code: CodeUserRestrictionExpired, Message: "expiresAt must be in the future", Details: map[string]any{"field": "expiresAt"}}
	}

	var created UserRestrictionResponse
	disconnectAfterCommit := false
	var disconnectUserID uint64
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := s.mobileUserByUUIDTx(ctx, tx, rawUserUUID, true)
		if err != nil {
			return err
		}
		createdRow, fullBanCreated, err := s.createUserRestrictionTx(ctx, tx, actorID, user, restrictionType, reason, req.ExpiresAt, meta, now)
		if err != nil {
			return err
		}
		created = *createdRow
		disconnectAfterCommit = fullBanCreated
		disconnectUserID = user.ID
		return nil
	}); err != nil {
		return nil, err
	}
	if disconnectAfterCommit && s.socketDisconnecter != nil {
		s.socketDisconnecter.DisconnectUser(disconnectUserID)
	}
	return &created, nil
}

type CreateUserRestrictionTxInput struct {
	ActorID         uint64
	UserID          uint64
	RestrictionType string
	Reason          string
	ExpiresAt       *time.Time
	Meta            RequestMeta
	Now             time.Time
}

func (s *Service) CreateUserRestrictionForUserTx(ctx context.Context, tx *gorm.DB, input CreateUserRestrictionTxInput) (*UserRestrictionResponse, error) {
	restrictionType := strings.ToUpper(strings.TrimSpace(input.RestrictionType))
	if !restriction.IsValidRestrictionType(restrictionType) {
		return nil, validationError("restrictionType is invalid", map[string]any{"field": "restrictionType"})
	}
	reason, err := requireRestrictionReason(input.Reason)
	if err != nil {
		return nil, err
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(now) {
		return nil, &ServiceError{Status: 409, Code: CodeUserRestrictionExpired, Message: "expiresAt must be in the future", Details: map[string]any{"field": "expiresAt"}}
	}
	user, err := s.mobileUserByIDTx(ctx, tx, input.UserID, true)
	if err != nil {
		return nil, err
	}
	created, _, err := s.createUserRestrictionTx(ctx, tx, input.ActorID, user, restrictionType, reason, input.ExpiresAt, input.Meta, now)
	return created, err
}

func (s *Service) DisconnectRestrictedUser(userID uint64) {
	if s.socketDisconnecter != nil {
		s.socketDisconnecter.DisconnectUser(userID)
	}
}

func (s *Service) createUserRestrictionTx(ctx context.Context, tx *gorm.DB, actorID uint64, user mobileUserIdentity, restrictionType string, reason string, expiresAt *time.Time, meta RequestMeta, now time.Time) (*UserRestrictionResponse, bool, error) {
	if err := s.expireActiveRestrictionsTx(ctx, tx, user.ID, user.UUID, restrictionType, now); err != nil {
		return nil, false, err
	}
	var exists bool
	if err := tx.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM user_restrictions
		  WHERE user_id = ?
		    AND restriction_type = ?
		    AND status = ?
		    AND (expires_at IS NULL OR expires_at > ?)
		)
	`, user.ID, restrictionType, restriction.StatusActive, now).Scan(&exists).Error; err != nil {
		return nil, false, err
	}
	if exists {
		return nil, false, conflictCodeError(CodeUserAlreadyRestricted, "User already has an active restriction")
	}

	row := UserRestriction{
		UUID:                 uuid.New(),
		UserID:               user.ID,
		RestrictionType:      restrictionType,
		Status:               restriction.StatusActive,
		Reason:               reason,
		CreatedByAdminUserID: actorID,
		ExpiresAt:            expiresAt,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, false, err
	}
	sessionsRevoked := int64(0)
	fullPlatformBan := restrictionType == restriction.TypeFullPlatformBan
	if fullPlatformBan {
		result := tx.WithContext(ctx).Exec(`
			UPDATE user_sessions
			SET revoked_at = ?, updated_at = ?
			WHERE user_id = ?
			  AND revoked_at IS NULL
		`, now, now, user.ID)
		if result.Error != nil {
			return nil, false, result.Error
		}
		sessionsRevoked = result.RowsAffected
	}
	created, err := s.userRestrictionRowsByID(ctx, tx, row.ID)
	if err != nil {
		return nil, false, err
	}
	if err := insertAdminAuditLogTx(ctx, tx, &actorID, AuditActorAdmin, "USER_RESTRICTION_CREATED", auditResourceUser, &user.UUID, &reason, map[string]any{}, map[string]any{
		"restrictionUuid":  created.RestrictionUUID,
		"restrictionType":  created.RestrictionType,
		"status":           created.Status,
		"expiresAt":        created.ExpiresAt,
		"sessionsRevoked":  sessionsRevoked,
		"socketDisconnect": fullPlatformBan,
	}, meta); err != nil {
		return nil, false, err
	}
	return &created, fullPlatformBan, nil
}

func (s *Service) RevokeUserRestriction(ctx context.Context, actorID uint64, rawUserUUID string, rawRestrictionUUID string, req RevokeUserRestrictionRequest, meta RequestMeta) (*UserRestrictionResponse, error) {
	reason, err := requireRestrictionReason(req.Reason)
	if err != nil {
		return nil, err
	}
	restrictionUUID, err := parseUUID(rawRestrictionUUID, "restrictionId")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var response UserRestrictionResponse
	var postCommitErr error
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := s.mobileUserByUUIDTx(ctx, tx, rawUserUUID, true)
		if err != nil {
			return err
		}
		var row UserRestriction
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND uuid = ?", user.ID, restrictionUUID).
			First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return notFoundCodeError(CodeUserRestrictionNotFound, "User restriction not found")
			}
			return err
		}
		if row.Status == restriction.StatusRevoked {
			return conflictCodeError(CodeUserNotRestricted, "User restriction is not active")
		}
		if row.Status == restriction.StatusExpired || (row.Status == restriction.StatusActive && row.ExpiresAt != nil && !row.ExpiresAt.After(now)) {
			if row.Status == restriction.StatusActive {
				if err := s.expireRestrictionRowTx(ctx, tx, user.UUID, row, now); err != nil {
					return err
				}
			}
			postCommitErr = &ServiceError{Status: 409, Code: CodeUserRestrictionExpired, Message: "User restriction is expired"}
			return nil
		}

		before := restrictionSnapshot(row)
		if err := tx.WithContext(ctx).Model(&UserRestriction{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":                   restriction.StatusRevoked,
			"revoked_by_admin_user_id": actorID,
			"revoked_at":               now,
			"revocation_reason":        reason,
			"updated_at":               now,
		}).Error; err != nil {
			return err
		}
		updated, err := s.userRestrictionRowsByID(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		response = updated
		return insertAdminAuditLogTx(ctx, tx, &actorID, AuditActorAdmin, "USER_RESTRICTION_REVOKED", auditResourceUser, &user.UUID, &reason, before, restrictionResponseSnapshot(updated), meta)
	}); err != nil {
		return nil, err
	}
	if postCommitErr != nil {
		return nil, postCommitErr
	}
	return &response, nil
}

func (s *Service) mobileUserRows(ctx context.Context, where string, args ...any) ([]adminMobileUserRow, error) {
	query := `
		SELECT
		  u.id AS user_id,
		  u.uuid::text AS user_uuid,
		  u.phone,
		  u.email::text AS email,
		  u.status,
		  u.onboarding_status,
		  u.last_login_at,
		  u.created_at,
		  u.updated_at,
		  p.uuid::text AS profile_uuid,
		  p.display_name,
		  p.gender,
		  p.city,
		  p.country,
		  p.profile_status,
		  p.completed_at,
		  (
		    SELECT COUNT(*)
		    FROM user_restrictions ur
		    WHERE ur.user_id = u.id
		      AND ur.status = 'ACTIVE'
		      AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
		  ) AS active_restriction_count
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE ` + where + `
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT ?
	`
	var rows []adminMobileUserRow
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) mobileUserByUUID(ctx context.Context, rawUserUUID string) (mobileUserIdentity, error) {
	return s.mobileUserByUUIDTx(ctx, s.db, rawUserUUID, false)
}

func (s *Service) mobileUserByUUIDTx(ctx context.Context, tx *gorm.DB, rawUserUUID string, lock bool) (mobileUserIdentity, error) {
	parsed, err := parseUUID(rawUserUUID, "userId")
	if err != nil {
		return mobileUserIdentity{}, err
	}
	return s.mobileUserByWhereTx(ctx, tx, "uuid = ?", parsed, lock)
}

func (s *Service) mobileUserByIDTx(ctx context.Context, tx *gorm.DB, userID uint64, lock bool) (mobileUserIdentity, error) {
	if userID == 0 {
		return mobileUserIdentity{}, userNotFoundError()
	}
	return s.mobileUserByWhereTx(ctx, tx, "id = ?", userID, lock)
}

func (s *Service) mobileUserByWhereTx(ctx context.Context, tx *gorm.DB, where string, arg any, lock bool) (mobileUserIdentity, error) {
	query := tx.WithContext(ctx)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var user mobileUserIdentity
	if err := query.Table("users").Select("id, uuid").Where(where, arg).Scan(&user).Error; err != nil {
		return mobileUserIdentity{}, err
	}
	if user.ID == 0 {
		return mobileUserIdentity{}, userNotFoundError()
	}
	return user, nil
}

type mobileUserIdentity struct {
	ID   uint64
	UUID uuid.UUID `gorm:"type:uuid"`
}

func (s *Service) userRestrictionRows(ctx context.Context, userID uint64, status string) ([]UserRestrictionResponse, error) {
	rows, err := s.userRestrictionRowData(ctx, s.db, userID, status)
	if err != nil {
		return nil, err
	}
	response := make([]UserRestrictionResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, row.toResponse())
	}
	return response, nil
}

func (s *Service) userRestrictionRowsByID(ctx context.Context, tx *gorm.DB, restrictionID uint64) (UserRestrictionResponse, error) {
	var rows []userRestrictionRow
	err := tx.WithContext(ctx).Raw(userRestrictionSelectSQL()+`
		WHERE ur.id = ?
	`, restrictionID).Scan(&rows).Error
	if err != nil {
		return UserRestrictionResponse{}, err
	}
	if len(rows) == 0 {
		return UserRestrictionResponse{}, notFoundCodeError(CodeUserRestrictionNotFound, "User restriction not found")
	}
	return rows[0].toResponse(), nil
}

func (s *Service) userRestrictionRowData(ctx context.Context, db *gorm.DB, userID uint64, status string) ([]userRestrictionRow, error) {
	where := "ur.user_id = ?"
	args := []any{userID}
	now := time.Now().UTC()
	switch status {
	case restriction.StatusActive:
		where += " AND ur.status = ? AND (ur.expires_at IS NULL OR ur.expires_at > ?)"
		args = append(args, restriction.StatusActive, now)
	case restriction.StatusExpired:
		where += " AND (ur.status = ? OR (ur.status = ? AND ur.expires_at IS NOT NULL AND ur.expires_at <= ?))"
		args = append(args, restriction.StatusExpired, restriction.StatusActive, now)
	default:
		where += " AND ur.status = ?"
		args = append(args, status)
	}
	var rows []userRestrictionRow
	err := db.WithContext(ctx).Raw(userRestrictionSelectSQL()+`
		WHERE `+where+`
		ORDER BY ur.created_at DESC, ur.id DESC
	`, args...).Scan(&rows).Error
	return rows, err
}

func userRestrictionSelectSQL() string {
	return `
		SELECT
		  ur.id,
		  ur.uuid::text AS restriction_uuid,
		  ur.restriction_type,
		  CASE
		    WHEN ur.status = 'ACTIVE' AND ur.expires_at IS NOT NULL AND ur.expires_at <= NOW() THEN 'EXPIRED'
		    ELSE ur.status
		  END AS status,
		  ur.reason,
		  creator.uuid::text AS created_by_admin_user_uuid,
		  creator.email AS created_by_admin_email,
		  revoker.uuid::text AS revoked_by_admin_user_uuid,
		  revoker.email AS revoked_by_admin_email,
		  ur.revoked_at,
		  ur.revocation_reason,
		  ur.expires_at,
		  ur.created_at,
		  ur.updated_at
		FROM user_restrictions ur
		LEFT JOIN admin_users creator ON creator.id = ur.created_by_admin_user_id
		LEFT JOIN admin_users revoker ON revoker.id = ur.revoked_by_admin_user_id
	`
}

func (s *Service) expireActiveRestrictionsTx(ctx context.Context, tx *gorm.DB, userID uint64, userUUID uuid.UUID, restrictionType string, now time.Time) error {
	var rows []UserRestriction
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND restriction_type = ? AND status = ? AND expires_at IS NOT NULL AND expires_at <= ?", userID, restrictionType, restriction.StatusActive, now).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := s.expireRestrictionRowTx(ctx, tx, userUUID, row, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) expireRestrictionRowTx(ctx context.Context, tx *gorm.DB, userUUID uuid.UUID, row UserRestriction, now time.Time) error {
	before := restrictionSnapshot(row)
	if err := tx.WithContext(ctx).Model(&UserRestriction{}).Where("id = ?", row.ID).Updates(map[string]any{
		"status":     restriction.StatusExpired,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}
	reason := "restriction_expired"
	after := map[string]any{}
	for key, value := range before {
		after[key] = value
	}
	after["status"] = restriction.StatusExpired
	return insertAdminAuditLogTx(ctx, tx, nil, AuditActorSystem, "USER_RESTRICTION_EXPIRED", auditResourceUser, &userUUID, &reason, before, after, RequestMeta{IPAddress: "system", UserAgent: "restriction-expiry"})
}

func (s *Service) recentReports(ctx context.Context, userID uint64) ([]AdminRecentReport, error) {
	var rows []recentReportRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  r.uuid::text AS report_uuid,
		  r.target_type,
		  rr.reason_code,
		  r.status,
		  r.severity,
		  r.created_at
		FROM reports r
		JOIN report_reasons rr ON rr.id = r.reason_id
		WHERE r.reported_user_id = ?
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT ?
	`, userID, recentReportLimit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]AdminRecentReport, 0, len(rows))
	for _, row := range rows {
		items = append(items, AdminRecentReport{
			ReportUUID: row.ReportUUID,
			TargetType: row.TargetType,
			ReasonCode: row.ReasonCode,
			Status:     row.Status,
			Severity:   row.Severity,
			CreatedAt:  row.CreatedAt,
		})
	}
	return items, nil
}

func (row adminMobileUserRow) toListItem() AdminMobileUserListItem {
	return AdminMobileUserListItem{
		UserUUID:               row.UserUUID,
		Phone:                  row.Phone,
		Email:                  row.Email,
		Status:                 row.Status,
		OnboardingStatus:       row.OnboardingStatus,
		Profile:                row.profileSummary(),
		ActiveRestrictionCount: row.ActiveRestrictionCount,
		LastLoginAt:            row.LastLoginAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func (row adminMobileUserRow) toDetailResponse() AdminMobileUserDetailResponse {
	return AdminMobileUserDetailResponse{
		UserUUID:         row.UserUUID,
		Phone:            row.Phone,
		Email:            row.Email,
		Status:           row.Status,
		OnboardingStatus: row.OnboardingStatus,
		Profile:          row.profileSummary(),
		Restrictions:     []UserRestrictionResponse{},
		RecentReports:    []AdminRecentReport{},
		AuditHistory:     []AuditLogResponse{},
		WalletSummary:    nil,
		LiveSummary:      nil,
		LastLoginAt:      row.LastLoginAt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (row adminMobileUserRow) profileSummary() *AdminUserProfileSummary {
	if row.ProfileUUID == nil {
		return nil
	}
	return &AdminUserProfileSummary{
		ProfileUUID:   row.ProfileUUID,
		DisplayName:   row.DisplayName,
		Gender:        row.Gender,
		City:          row.City,
		Country:       row.Country,
		ProfileStatus: row.ProfileStatus,
		CompletedAt:   row.CompletedAt,
	}
}

func (row userRestrictionRow) toResponse() UserRestrictionResponse {
	return UserRestrictionResponse{
		RestrictionUUID:        row.RestrictionUUID,
		RestrictionType:        row.RestrictionType,
		Status:                 row.Status,
		Reason:                 row.Reason,
		CreatedByAdminUserUUID: row.CreatedByAdminUserUUID,
		CreatedByAdminEmail:    row.CreatedByAdminEmail,
		RevokedByAdminUserUUID: row.RevokedByAdminUserUUID,
		RevokedByAdminEmail:    row.RevokedByAdminEmail,
		RevokedAt:              row.RevokedAt,
		RevocationReason:       row.RevocationReason,
		ExpiresAt:              row.ExpiresAt,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func restrictionSnapshot(row UserRestriction) map[string]any {
	return map[string]any{
		"restrictionUuid": row.UUID.String(),
		"restrictionType": row.RestrictionType,
		"status":          row.Status,
		"reason":          row.Reason,
		"expiresAt":       row.ExpiresAt,
		"revokedAt":       row.RevokedAt,
	}
}

func restrictionResponseSnapshot(row UserRestrictionResponse) map[string]any {
	return map[string]any{
		"restrictionUuid": row.RestrictionUUID,
		"restrictionType": row.RestrictionType,
		"status":          row.Status,
		"reason":          row.Reason,
		"expiresAt":       row.ExpiresAt,
		"revokedAt":       row.RevokedAt,
	}
}

func parseUUID(raw string, field string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, validationError(field+" is invalid", map[string]any{"field": field})
	}
	return parsed, nil
}

func normalizeRestrictionStatus(raw string) string {
	status := strings.ToUpper(strings.TrimSpace(raw))
	if status == "" {
		return restriction.StatusActive
	}
	return status
}

func requireRestrictionReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if reason == "" {
		return "", &ServiceError{Status: 400, Code: CodeUserRestrictionReasonRequired, Message: "reason is required", Details: map[string]any{"field": "reason"}}
	}
	return reason, nil
}

func userNotFoundError() *ServiceError {
	return &ServiceError{Status: 404, Code: CodeUserNotFound, Message: "User not found"}
}

func notFoundCodeError(code string, message string) *ServiceError {
	return &ServiceError{Status: 404, Code: code, Message: message}
}

func isValidMobileUserStatus(status string) bool {
	return status == "ACTIVE" || status == "SUSPENDED" || status == "BANNED" || status == "DELETED"
}

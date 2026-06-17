package subscription

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db *gorm.DB
}

type paymentRequestRow struct {
	ID                   uint64
	UUID                 string
	UserID               uint64
	UserUUID             string
	PlanID               *uint64
	PlanCodeSnapshot     string
	PlanNameSnapshot     string
	PriceAmountSnapshot  int
	CurrencySnapshot     string
	DurationDaysSnapshot int
	EntitlementsRaw      string
	PaymentProvider      string
	PaymentReference     *string
	PayerPhone           *string
	Note                 *string
	Status               string
	SubmittedAt          time.Time
	ReviewedAt           *time.Time
	RejectionReason      *string
	SubscriptionUUID     *string
	CreatedAt            time.Time
}

type subscriptionRow struct {
	ID              uint64
	UUID            string
	PlanCode        string
	PlanName        string
	Status          string
	StartsAt        time.Time
	ExpiresAt       time.Time
	EntitlementsRaw string
}

type entitlementSnapshot struct {
	IsPremium    bool
	PlanCode     *string
	ExpiresAt    *time.Time
	Entitlements Entitlements
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListPlans(ctx context.Context) (*PlanListResponse, error) {
	var plans []SubscriptionPlan
	if err := s.db.WithContext(ctx).
		Where("is_active = TRUE").
		Order("sort_order ASC, id ASC").
		Find(&plans).Error; err != nil {
		return nil, err
	}
	response := &PlanListResponse{Items: make([]PlanResponse, 0, len(plans))}
	for _, plan := range plans {
		item, err := planToResponse(plan)
		if err != nil {
			return nil, validationError("Plan entitlements are invalid", map[string]any{"planCode": plan.PlanCode, "error": err.Error()})
		}
		response.Items = append(response.Items, item)
	}
	return response, nil
}

func (s *Service) CurrentSubscription(ctx context.Context, userID uint64) (*CurrentSubscriptionResponse, error) {
	row, err := s.activeSubscription(ctx, s.db.WithContext(ctx), userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &CurrentSubscriptionResponse{}, nil
	}
	subscription, err := row.toResponse()
	if err != nil {
		return nil, err
	}
	return &CurrentSubscriptionResponse{Subscription: &subscription}, nil
}

func (s *Service) GetEntitlements(ctx context.Context, userID uint64) (*EntitlementsResponse, error) {
	snapshot, err := s.entitlements(ctx, s.db.WithContext(ctx), userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &EntitlementsResponse{
		IsPremium:    snapshot.IsPremium,
		PlanCode:     snapshot.PlanCode,
		ExpiresAt:    snapshot.ExpiresAt,
		Entitlements: snapshot.Entitlements,
	}, nil
}

func (s *Service) PremiumStatus(ctx context.Context, userID uint64) (*PremiumStatusResponse, error) {
	snapshot, err := s.entitlements(ctx, s.db.WithContext(ctx), userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &PremiumStatusResponse{
		IsPremium: snapshot.IsPremium,
		PlanCode:  snapshot.PlanCode,
		ExpiresAt: snapshot.ExpiresAt,
	}, nil
}

func (s *Service) Usage(ctx context.Context, userID uint64) (*UsageResponse, error) {
	now := time.Now().UTC()
	usageDate := utcDate(now)
	snapshot, err := s.entitlements(ctx, s.db.WithContext(ctx), userID, now)
	if err != nil {
		return nil, err
	}
	counts, err := s.usageCounts(ctx, s.db.WithContext(ctx), userID, usageDate)
	if err != nil {
		return nil, err
	}
	likes := counts[FeatureLikeDaily].UsedCount
	superLikes := counts[FeatureSuperLikeDaily].UsedCount
	audioSeconds := counts[FeatureAudioCallDailySeconds].UsedSeconds
	videoSeconds := counts[FeatureVideoCallDailySeconds].UsedSeconds

	return &UsageResponse{
		Date: usageDate.Format("2006-01-02"),
		Usage: UsageUsedResponse{
			LikesUsedToday:            likes,
			SuperLikesUsedToday:       superLikes,
			AudioCallSecondsUsedToday: audioSeconds,
			VideoCallSecondsUsedToday: videoSeconds,
		},
		Remaining: UsageRemainingResponse{
			LikesRemainingToday:            remaining(snapshot.Entitlements.DailyLikeLimit, likes),
			SuperLikesRemainingToday:       remaining(snapshot.Entitlements.DailySuperLikeLimit, superLikes),
			AudioCallSecondsRemainingToday: remaining(snapshot.Entitlements.DailyCallLimitSeconds, audioSeconds),
			VideoCallSecondsRemainingToday: remaining(snapshot.Entitlements.DailyCallLimitSeconds, videoSeconds),
		},
	}, nil
}

func (s *Service) CreateManualPaymentRequest(ctx context.Context, userID uint64, req CreateManualPaymentRequest) (*PaymentRequestResponse, error) {
	normalized, err := normalizeCreatePaymentRequest(req)
	if err != nil {
		return nil, err
	}
	var plan SubscriptionPlan
	if err := s.db.WithContext(ctx).
		Where("plan_code = ? AND is_active = TRUE", normalized.PlanCode).
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFoundError("Subscription plan not found")
		}
		return nil, err
	}
	if _, err := DecodeEntitlements(plan.Entitlements); err != nil {
		return nil, validationError("Plan entitlements are invalid", map[string]any{"planCode": plan.PlanCode, "error": err.Error()})
	}

	now := time.Now().UTC()
	planID := plan.ID
	reference := normalized.PaymentReference
	request := ManualPaymentRequest{
		UUID:                 uuid.New(),
		UserID:               userID,
		PlanID:               &planID,
		PlanCodeSnapshot:     plan.PlanCode,
		PlanNameSnapshot:     plan.Name,
		PriceAmountSnapshot:  plan.PriceAmount,
		CurrencySnapshot:     plan.Currency,
		DurationDaysSnapshot: plan.DurationDays,
		EntitlementsSnapshot: datatypes.JSON(plan.Entitlements),
		PaymentProvider:      normalized.PaymentProvider,
		PaymentReference:     &reference,
		PayerPhone:           normalized.PayerPhone,
		Note:                 normalized.Note,
		Status:               PaymentStatusPending,
		SubmittedAt:          now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := s.db.WithContext(ctx).Create(&request).Error; err != nil {
		return nil, mapPaymentRequestCreateError(err)
	}
	return s.userPaymentRequestByID(ctx, userID, request.ID)
}

func (s *Service) ListManualPaymentRequests(ctx context.Context, userID uint64) (*PaymentRequestListResponse, error) {
	rows, err := s.paymentRequestRows(ctx, `
		WHERE mpr.user_id = ?
		ORDER BY mpr.created_at DESC, mpr.id DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, err
	}
	response := &PaymentRequestListResponse{Items: make([]PaymentRequestResponse, 0, len(rows))}
	for _, row := range rows {
		item, err := row.toUserResponse()
		if err != nil {
			return nil, err
		}
		response.Items = append(response.Items, item)
	}
	return response, nil
}

func (s *Service) AdminListPaymentRequests(ctx context.Context, status string) (*AdminPaymentRequestListResponse, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	where := `
		WHERE (? = '' OR mpr.status = ?)
		ORDER BY mpr.created_at DESC, mpr.id DESC
		LIMIT 100
	`
	rows, err := s.paymentRequestRows(ctx, where, status, status)
	if err != nil {
		return nil, err
	}
	response := &AdminPaymentRequestListResponse{Items: make([]AdminPaymentRequestResponse, 0, len(rows))}
	for _, row := range rows {
		item, err := row.toAdminResponse()
		if err != nil {
			return nil, err
		}
		response.Items = append(response.Items, item)
	}
	return response, nil
}

func (s *Service) AdminPaymentRequestDetail(ctx context.Context, paymentRequestUUID string) (*AdminPaymentRequestResponse, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(paymentRequestUUID))
	if err != nil {
		return nil, validationError("paymentRequestUuid is invalid", map[string]any{"field": "paymentRequestUuid"})
	}
	rows, err := s.paymentRequestRows(ctx, `WHERE mpr.uuid = ?`, parsed)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, notFoundError("Payment request not found")
	}
	response, err := rows[0].toAdminResponse()
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) ApprovePaymentRequest(ctx context.Context, adminID uint64, paymentRequestUUID string, req ReviewPaymentRequest, meta AdminMeta) (*AdminPaymentRequestResponse, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(paymentRequestUUID))
	if err != nil {
		return nil, validationError("paymentRequestUuid is invalid", map[string]any{"field": "paymentRequestUuid"})
	}
	note := trimOptional(req.Note, 500)
	if len(strings.TrimSpace(req.Note)) > 500 {
		return nil, validationError("note must be at most 500 characters", map[string]any{"field": "note"})
	}

	var requestID uint64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.lockPaymentRequestTx(ctx, tx, parsed)
		if err != nil {
			return err
		}
		requestID = row.ID
		if row.Status == PaymentStatusApproved {
			return nil
		}
		if row.Status == PaymentStatusRejected {
			return paymentRejectedError()
		}
		if row.Status != PaymentStatusPending && row.Status != PaymentStatusUnderReview {
			return conflictError("Payment request cannot be approved")
		}
		if _, err := DecodeEntitlements([]byte(row.EntitlementsRaw)); err != nil {
			return validationError("Payment request entitlements are invalid", map[string]any{"error": err.Error()})
		}
		now := time.Now().UTC()
		if err := s.lockActiveSubscriptionsTx(ctx, tx, row.UserID, now); err != nil {
			return err
		}
		startsAt, err := s.nextSubscriptionStartTx(ctx, tx, row.UserID, now)
		if err != nil {
			return err
		}
		expiresAt := startsAt.AddDate(0, 0, row.DurationDaysSnapshot)
		planID := row.PlanID
		paymentRequestID := row.ID
		subscription := UserSubscription{
			UUID:                 uuid.New(),
			UserID:               row.UserID,
			PlanID:               planID,
			PaymentRequestID:     &paymentRequestID,
			PlanCode:             row.PlanCodeSnapshot,
			PlanName:             row.PlanNameSnapshot,
			Source:               SubscriptionSourceManualPayment,
			Status:               SubscriptionStatusActive,
			StartsAt:             startsAt,
			ExpiresAt:            expiresAt,
			EntitlementsSnapshot: datatypes.JSON([]byte(row.EntitlementsRaw)),
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if err := tx.WithContext(ctx).Create(&subscription).Error; err != nil {
			return err
		}
		before := snapshotMap(row)
		if err := tx.WithContext(ctx).Model(&ManualPaymentRequest{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":               PaymentStatusApproved,
			"reviewed_at":          now,
			"reviewed_by_admin_id": adminID,
			"subscription_id":      subscription.ID,
			"updated_at":           now,
		}).Error; err != nil {
			return err
		}
		after := map[string]any{
			"status":           PaymentStatusApproved,
			"subscriptionUuid": subscription.UUID.String(),
			"reviewedAt":       now,
		}
		return s.insertReviewAndAuditTx(ctx, tx, row.ID, adminID, ReviewActionApproved, note, before, after, parsed, meta)
	})
	if err != nil {
		return nil, err
	}
	if requestID == 0 {
		return nil, notFoundError("Payment request not found")
	}
	return s.AdminPaymentRequestDetail(ctx, paymentRequestUUID)
}

func (s *Service) RejectPaymentRequest(ctx context.Context, adminID uint64, paymentRequestUUID string, req RejectPaymentRequest, meta AdminMeta) (*AdminPaymentRequestResponse, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(paymentRequestUUID))
	if err != nil {
		return nil, validationError("paymentRequestUuid is invalid", map[string]any{"field": "paymentRequestUuid"})
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, validationError("reason is required", map[string]any{"field": "reason"})
	}
	if len(reason) > 500 {
		return nil, validationError("reason must be at most 500 characters", map[string]any{"field": "reason"})
	}

	var requestID uint64
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := s.lockPaymentRequestTx(ctx, tx, parsed)
		if err != nil {
			return err
		}
		requestID = row.ID
		if row.Status == PaymentStatusRejected {
			return nil
		}
		if row.Status == PaymentStatusApproved {
			return paymentApprovedError()
		}
		if row.Status != PaymentStatusPending && row.Status != PaymentStatusUnderReview {
			return conflictError("Payment request cannot be rejected")
		}
		now := time.Now().UTC()
		before := snapshotMap(row)
		if err := tx.WithContext(ctx).Model(&ManualPaymentRequest{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":               PaymentStatusRejected,
			"reviewed_at":          now,
			"reviewed_by_admin_id": adminID,
			"rejection_reason":     reason,
			"updated_at":           now,
		}).Error; err != nil {
			return err
		}
		after := map[string]any{"status": PaymentStatusRejected, "rejectionReason": reason, "reviewedAt": now}
		return s.insertReviewAndAuditTx(ctx, tx, row.ID, adminID, ReviewActionRejected, &reason, before, after, parsed, meta)
	})
	if err != nil {
		return nil, err
	}
	if requestID == 0 {
		return nil, notFoundError("Payment request not found")
	}
	return s.AdminPaymentRequestDetail(ctx, paymentRequestUUID)
}

func (s *Service) ConsumeActionUsageTx(ctx context.Context, tx *gorm.DB, userID uint64, actionType string, usageDate time.Time) error {
	var featureKey string
	var limit int
	var limitErr *ServiceError
	snapshot, err := s.entitlements(ctx, tx, userID, time.Now().UTC())
	if err != nil {
		return err
	}
	switch strings.ToUpper(strings.TrimSpace(actionType)) {
	case "LIKE":
		featureKey = FeatureLikeDaily
		limit = snapshot.Entitlements.DailyLikeLimit
		limitErr = likeLimitReachedError()
	case "SUPER_LIKE":
		featureKey = FeatureSuperLikeDaily
		limit = snapshot.Entitlements.DailySuperLikeLimit
		limitErr = superLikeLimitReachedError()
	default:
		return nil
	}
	if limit <= 0 {
		return limitErr
	}
	date := utcDate(usageDate)
	now := time.Now().UTC()
	if err := tx.WithContext(ctx).Exec(`
		INSERT INTO user_feature_usage (user_id, feature_key, usage_date, used_count, used_seconds, created_at, updated_at)
		VALUES (?, ?, ?, 0, 0, ?, ?)
		ON CONFLICT (user_id, feature_key, usage_date) DO NOTHING
	`, userID, featureKey, date, now, now).Error; err != nil {
		return err
	}
	var usage UserFeatureUsage
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND feature_key = ? AND usage_date = ?", userID, featureKey, date).
		First(&usage).Error; err != nil {
		return err
	}
	if usage.UsedCount >= limit {
		return limitErr
	}
	return tx.WithContext(ctx).Model(&UserFeatureUsage{}).Where("id = ?", usage.ID).Updates(map[string]any{
		"used_count": usage.UsedCount + 1,
		"updated_at": now,
	}).Error
}

type normalizedPaymentRequest struct {
	PlanCode         string
	PaymentProvider  string
	PaymentReference string
	PayerPhone       *string
	Note             *string
}

type AdminMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

func normalizeCreatePaymentRequest(req CreateManualPaymentRequest) (normalizedPaymentRequest, error) {
	planCode := strings.ToUpper(strings.TrimSpace(req.PlanCode))
	provider := strings.ToUpper(strings.TrimSpace(req.PaymentProvider))
	reference := strings.TrimSpace(req.PaymentReference)
	if planCode == "" {
		return normalizedPaymentRequest{}, validationError("planCode is required", map[string]any{"field": "planCode"})
	}
	if provider == "" || len(provider) > 50 {
		return normalizedPaymentRequest{}, validationError("paymentProvider is invalid", map[string]any{"field": "paymentProvider"})
	}
	if reference == "" || len(reference) > 120 {
		return normalizedPaymentRequest{}, validationError("paymentReference is invalid", map[string]any{"field": "paymentReference"})
	}
	var payerPhone *string
	if req.PayerPhone != nil {
		value := strings.TrimSpace(*req.PayerPhone)
		if len(value) > 30 {
			return normalizedPaymentRequest{}, validationError("payerPhone must be at most 30 characters", map[string]any{"field": "payerPhone"})
		}
		if value != "" {
			payerPhone = &value
		}
	}
	note := trimOptionalPtr(req.Note, 500)
	if req.Note != nil && len(strings.TrimSpace(*req.Note)) > 500 {
		return normalizedPaymentRequest{}, validationError("note must be at most 500 characters", map[string]any{"field": "note"})
	}
	return normalizedPaymentRequest{
		PlanCode:         planCode,
		PaymentProvider:  provider,
		PaymentReference: reference,
		PayerPhone:       payerPhone,
		Note:             note,
	}, nil
}

func (s *Service) activeSubscription(ctx context.Context, db *gorm.DB, userID uint64, now time.Time) (*subscriptionRow, error) {
	var row subscriptionRow
	err := db.WithContext(ctx).Raw(`
		SELECT
		  id,
		  uuid::text AS uuid,
		  plan_code,
		  plan_name,
		  status,
		  starts_at,
		  expires_at,
		  entitlements_snapshot::text AS entitlements_raw
		FROM user_subscriptions
		WHERE user_id = ?
		  AND status = 'ACTIVE'
		  AND starts_at <= ?
		  AND expires_at > ?
		ORDER BY expires_at DESC, id DESC
		LIMIT 1
	`, userID, now, now).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) entitlements(ctx context.Context, db *gorm.DB, userID uint64, now time.Time) (entitlementSnapshot, error) {
	row, err := s.activeSubscription(ctx, db, userID, now)
	if err != nil {
		return entitlementSnapshot{}, err
	}
	if row == nil {
		return entitlementSnapshot{Entitlements: FreeEntitlements()}, nil
	}
	entitlements, err := DecodeEntitlements([]byte(row.EntitlementsRaw))
	if err != nil {
		return entitlementSnapshot{}, validationError("Subscription entitlements are invalid", map[string]any{"subscriptionUuid": row.UUID, "error": err.Error()})
	}
	planCode := row.PlanCode
	expiresAt := row.ExpiresAt
	return entitlementSnapshot{IsPremium: true, PlanCode: &planCode, ExpiresAt: &expiresAt, Entitlements: entitlements}, nil
}

func (s *Service) usageCounts(ctx context.Context, db *gorm.DB, userID uint64, usageDate time.Time) (map[string]UserFeatureUsage, error) {
	var rows []UserFeatureUsage
	if err := db.WithContext(ctx).Where("user_id = ? AND usage_date = ?", userID, usageDate).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := map[string]UserFeatureUsage{
		FeatureLikeDaily:             {},
		FeatureSuperLikeDaily:        {},
		FeatureAudioCallDailySeconds: {},
		FeatureVideoCallDailySeconds: {},
	}
	for _, row := range rows {
		result[row.FeatureKey] = row
	}
	return result, nil
}

func (s *Service) userPaymentRequestByID(ctx context.Context, userID uint64, id uint64) (*PaymentRequestResponse, error) {
	rows, err := s.paymentRequestRows(ctx, `WHERE mpr.user_id = ? AND mpr.id = ?`, userID, id)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, notFoundError("Payment request not found")
	}
	response, err := rows[0].toUserResponse()
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) paymentRequestRows(ctx context.Context, where string, args ...any) ([]paymentRequestRow, error) {
	query := `
		SELECT
		  mpr.id,
		  mpr.uuid::text AS uuid,
		  mpr.user_id,
		  u.uuid::text AS user_uuid,
		  mpr.plan_id,
		  mpr.plan_code_snapshot,
		  mpr.plan_name_snapshot,
		  mpr.price_amount_snapshot,
		  mpr.currency_snapshot,
		  mpr.duration_days_snapshot,
		  mpr.entitlements_snapshot::text AS entitlements_raw,
		  mpr.payment_provider,
		  mpr.payment_reference,
		  mpr.payer_phone,
		  mpr.note,
		  mpr.status,
		  mpr.submitted_at,
		  mpr.reviewed_at,
		  mpr.rejection_reason,
		  us.uuid::text AS subscription_uuid,
		  mpr.created_at
		FROM manual_payment_requests mpr
		JOIN users u ON u.id = mpr.user_id
		LEFT JOIN user_subscriptions us ON us.id = mpr.subscription_id
	` + "\n" + where
	var rows []paymentRequestRow
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) lockPaymentRequestTx(ctx context.Context, tx *gorm.DB, paymentRequestUUID uuid.UUID) (*paymentRequestRow, error) {
	query := `
		SELECT
		  mpr.id,
		  mpr.uuid::text AS uuid,
		  mpr.user_id,
		  u.uuid::text AS user_uuid,
		  mpr.plan_id,
		  mpr.plan_code_snapshot,
		  mpr.plan_name_snapshot,
		  mpr.price_amount_snapshot,
		  mpr.currency_snapshot,
		  mpr.duration_days_snapshot,
		  mpr.entitlements_snapshot::text AS entitlements_raw,
		  mpr.payment_provider,
		  mpr.payment_reference,
		  mpr.payer_phone,
		  mpr.note,
		  mpr.status,
		  mpr.submitted_at,
		  mpr.reviewed_at,
		  mpr.rejection_reason,
		  us.uuid::text AS subscription_uuid,
		  mpr.created_at
		FROM manual_payment_requests mpr
		JOIN users u ON u.id = mpr.user_id
		LEFT JOIN user_subscriptions us ON us.id = mpr.subscription_id
		WHERE mpr.uuid = ?
		FOR UPDATE OF mpr
	`
	var row paymentRequestRow
	if err := tx.WithContext(ctx).Raw(query, paymentRequestUUID).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, notFoundError("Payment request not found")
	}
	return &row, nil
}

func (s *Service) lockActiveSubscriptionsTx(ctx context.Context, tx *gorm.DB, userID uint64, now time.Time) error {
	return tx.WithContext(ctx).Exec(`
		SELECT id
		FROM user_subscriptions
		WHERE user_id = ?
		  AND status = 'ACTIVE'
		  AND expires_at > ?
		FOR UPDATE
	`, userID, now).Error
}

func (s *Service) nextSubscriptionStartTx(ctx context.Context, tx *gorm.DB, userID uint64, now time.Time) (time.Time, error) {
	var latest *time.Time
	if err := tx.WithContext(ctx).Raw(`
		SELECT MAX(expires_at)
		FROM user_subscriptions
		WHERE user_id = ?
		  AND status = 'ACTIVE'
		  AND expires_at > ?
	`, userID, now).Scan(&latest).Error; err != nil {
		return time.Time{}, err
	}
	if latest != nil && latest.After(now) {
		return *latest, nil
	}
	return now, nil
}

func (s *Service) insertReviewAndAuditTx(ctx context.Context, tx *gorm.DB, requestID uint64, adminID uint64, action string, note *string, before any, after any, resourceUUID uuid.UUID, meta AdminMeta) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(`
		INSERT INTO payment_review_actions (
		  payment_request_id,
		  admin_user_id,
		  action,
		  note,
		  before_snapshot,
		  after_snapshot
		)
		VALUES (?, ?, ?, ?, ?::jsonb, ?::jsonb)
	`, requestID, adminID, action, note, string(beforeJSON), string(afterJSON)).Error; err != nil {
		return err
	}
	auditAction := "SUBSCRIPTION_PAYMENT_" + action
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
		VALUES (?, 'ADMIN', ?, 'MANUAL_PAYMENT_REQUEST', ?, ?, ?::jsonb, ?::jsonb, ?, ?, ?)
	`, adminID, auditAction, resourceUUID, note, string(beforeJSON), string(afterJSON), nullableString(meta.RequestID), meta.IPAddress, meta.UserAgent).Error
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func planToResponse(plan SubscriptionPlan) (PlanResponse, error) {
	entitlements, err := DecodeEntitlements(plan.Entitlements)
	if err != nil {
		return PlanResponse{}, err
	}
	return PlanResponse{
		PlanUUID:     plan.UUID.String(),
		PlanCode:     plan.PlanCode,
		Name:         plan.Name,
		Description:  plan.Description,
		PriceAmount:  plan.PriceAmount,
		Currency:     plan.Currency,
		DurationDays: plan.DurationDays,
		Entitlements: entitlements,
	}, nil
}

func (row subscriptionRow) toResponse() (SubscriptionResponse, error) {
	entitlements, err := DecodeEntitlements([]byte(row.EntitlementsRaw))
	if err != nil {
		return SubscriptionResponse{}, err
	}
	return SubscriptionResponse{
		SubscriptionUUID: row.UUID,
		PlanCode:         row.PlanCode,
		PlanName:         row.PlanName,
		Status:           row.Status,
		StartsAt:         row.StartsAt,
		ExpiresAt:        row.ExpiresAt,
		Entitlements:     entitlements,
	}, nil
}

func (row paymentRequestRow) toUserResponse() (PaymentRequestResponse, error) {
	return PaymentRequestResponse{
		PaymentRequestUUID: row.UUID,
		PlanCode:           row.PlanCodeSnapshot,
		PlanName:           row.PlanNameSnapshot,
		PriceAmount:        row.PriceAmountSnapshot,
		Currency:           row.CurrencySnapshot,
		DurationDays:       row.DurationDaysSnapshot,
		PaymentProvider:    row.PaymentProvider,
		PaymentReference:   row.PaymentReference,
		PayerPhone:         row.PayerPhone,
		Note:               row.Note,
		Status:             row.Status,
		SubmittedAt:        row.SubmittedAt,
		ReviewedAt:         row.ReviewedAt,
		RejectionReason:    row.RejectionReason,
		SubscriptionUUID:   row.SubscriptionUUID,
		CreatedAt:          row.CreatedAt,
	}, nil
}

func (row paymentRequestRow) toAdminResponse() (AdminPaymentRequestResponse, error) {
	entitlements, err := DecodeEntitlements([]byte(row.EntitlementsRaw))
	if err != nil {
		return AdminPaymentRequestResponse{}, err
	}
	return AdminPaymentRequestResponse{
		PaymentRequestUUID: row.UUID,
		UserUUID:           row.UserUUID,
		PlanCode:           row.PlanCodeSnapshot,
		PlanName:           row.PlanNameSnapshot,
		PriceAmount:        row.PriceAmountSnapshot,
		Currency:           row.CurrencySnapshot,
		DurationDays:       row.DurationDaysSnapshot,
		Entitlements:       entitlements,
		PaymentProvider:    row.PaymentProvider,
		PaymentReference:   row.PaymentReference,
		PayerPhone:         row.PayerPhone,
		Note:               row.Note,
		Status:             row.Status,
		SubmittedAt:        row.SubmittedAt,
		ReviewedAt:         row.ReviewedAt,
		RejectionReason:    row.RejectionReason,
		SubscriptionUUID:   row.SubscriptionUUID,
		CreatedAt:          row.CreatedAt,
	}, nil
}

func snapshotMap(row *paymentRequestRow) map[string]any {
	return map[string]any{
		"paymentRequestUuid": row.UUID,
		"userUuid":           row.UserUUID,
		"planCode":           row.PlanCodeSnapshot,
		"status":             row.Status,
		"priceAmount":        row.PriceAmountSnapshot,
		"currency":           row.CurrencySnapshot,
	}
}

func mapPaymentRequestCreateError(err error) error {
	message := err.Error()
	if strings.Contains(message, "manual_payment_requests_reference_unique") {
		return conflictError("Payment reference is already used")
	}
	if strings.Contains(message, "manual_payment_requests_one_open_per_user") {
		return conflictError("User already has an open payment request")
	}
	return err
}

func trimOptional(value string, max int) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > max {
		trimmed = trimmed[:max]
	}
	return &trimmed
}

func trimOptionalPtr(value *string, max int) *string {
	if value == nil {
		return nil
	}
	return trimOptional(*value, max)
}

func remaining(limit int, used int) int {
	value := limit - used
	if value < 0 {
		return 0
	}
	return value
}

func utcDate(now time.Time) time.Time {
	return time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

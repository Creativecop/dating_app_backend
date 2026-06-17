package admin

import (
	"context"
	"database/sql"
	"time"
)

func (s *Service) DashboardSummary(ctx context.Context, adminID uint64) (*DashboardSummaryResponse, error) {
	broad, err := s.hasAnyPermission(ctx, adminID, PermissionAnalyticsRead)
	if err != nil {
		return nil, err
	}
	reportsAllowed, err := s.hasAnyPermission(ctx, adminID, PermissionAnalyticsReportsRead, PermissionAnalyticsTrustSafetyRead)
	if err != nil {
		return nil, err
	}
	restrictionsAllowed, err := s.hasAnyPermission(ctx, adminID, PermissionAnalyticsRestrictionsRead, PermissionAnalyticsTrustSafetyRead)
	if err != nil {
		return nil, err
	}
	adminActivityAllowed, err := s.hasAnyPermission(ctx, adminID, PermissionAnalyticsAdminActivityRead)
	if err != nil {
		return nil, err
	}
	subscriptionAllowed, err := s.hasAnyPermission(ctx, adminID, PermissionAnalyticsSubscriptionPaymentsRead)
	if err != nil {
		return nil, err
	}
	if !broad && !reportsAllowed && !restrictionsAllowed && !adminActivityAllowed && !subscriptionAllowed {
		return nil, forbiddenError(CodeAdminAnalyticsAccessDenied, "Analytics access denied", nil)
	}

	now := time.Now().UTC()
	todayStart := startOfUTCDay(now)
	tomorrowStart := todayStart.AddDate(0, 0, 1)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	response := &DashboardSummaryResponse{Modules: s.capabilities}
	if broad {
		users, err := s.dashboardUsersSummary(ctx, todayStart, tomorrowStart, monthStart)
		if err != nil {
			return nil, err
		}
		response.Users = users
	}
	if broad || reportsAllowed {
		reports, err := s.dashboardReportsSummary(ctx, todayStart, tomorrowStart)
		if err != nil {
			return nil, err
		}
		response.Reports = reports
	}
	if broad || restrictionsAllowed {
		restrictions, err := s.dashboardRestrictionsSummary(ctx)
		if err != nil {
			return nil, err
		}
		response.Restrictions = restrictions
	}
	if broad || adminActivityAllowed {
		adminSummary, err := s.dashboardAdminSummary(ctx, todayStart, tomorrowStart)
		if err != nil {
			return nil, err
		}
		response.Admin = adminSummary
	}
	if subscriptionAllowed && s.capabilities.SubscriptionPaymentAnalytics {
		summary, err := s.dashboardSubscriptionPaymentsSummary(ctx, todayStart, tomorrowStart)
		if err != nil {
			return nil, err
		}
		response.SubscriptionPayments = summary
	}
	return response, nil
}

func (s *Service) UserAnalytics(ctx context.Context, query AnalyticsQuery) (*UserAnalyticsResponse, error) {
	period, err := normalizeAnalyticsRange(query.From, query.To, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	total, err := s.countRaw(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	newUsers, err := s.countRaw(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	restrictedUsers, err := s.countRaw(ctx, `
		SELECT COUNT(DISTINCT user_id)
		FROM user_restrictions
		WHERE status = 'ACTIVE'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		return nil, err
	}
	fullPlatformBans, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM user_restrictions
		WHERE status = 'ACTIVE'
		  AND restriction_type = 'FULL_PLATFORM_BAN'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		return nil, err
	}
	byStatus, err := s.countRows(ctx, `
		SELECT status AS key, COUNT(*) AS count
		FROM users
		WHERE deleted_at IS NULL
		GROUP BY status
		ORDER BY status ASC
	`)
	if err != nil {
		return nil, err
	}
	note := "Activity tracking is not enabled"
	return &UserAnalyticsResponse{
		Period:           period,
		TotalUsers:       total,
		NewUsers:         newUsers,
		ActiveUsers:      nil,
		ActivityNote:     &note,
		RestrictedUsers:  restrictedUsers,
		FullPlatformBans: fullPlatformBans,
		UsersByStatus:    byStatus,
	}, nil
}

func (s *Service) ReportAnalytics(ctx context.Context, query AnalyticsQuery) (*ReportAnalyticsResponse, error) {
	period, err := normalizeAnalyticsRange(query.From, query.To, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.reportAnalyticsForPeriod(ctx, period)
}

func (s *Service) RestrictionAnalytics(ctx context.Context, query AnalyticsQuery) (*RestrictionAnalyticsResponse, error) {
	period, err := normalizeAnalyticsRange(query.From, query.To, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return s.restrictionAnalyticsForPeriod(ctx, period)
}

func (s *Service) TrustSafetyAnalytics(ctx context.Context, query AnalyticsQuery) (*TrustSafetyAnalyticsResponse, error) {
	period, err := normalizeAnalyticsRange(query.From, query.To, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	reports, err := s.reportAnalyticsForPeriod(ctx, period)
	if err != nil {
		return nil, err
	}
	restrictions, err := s.restrictionAnalyticsForPeriod(ctx, period)
	if err != nil {
		return nil, err
	}
	return &TrustSafetyAnalyticsResponse{Period: period, Reports: *reports, Restrictions: *restrictions}, nil
}

func (s *Service) AdminActivityAnalytics(ctx context.Context, query AnalyticsQuery) (*AdminActivityAnalyticsResponse, error) {
	period, err := normalizeAnalyticsRange(query.From, query.To, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	byType, err := s.countRows(ctx, `
		SELECT action AS key, COUNT(*) AS count
		FROM admin_audit_logs
		WHERE created_at >= ? AND created_at < ?
		GROUP BY action
		ORDER BY count DESC, action ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	byAdmin, err := s.adminActorRows(ctx, `
		SELECT
		  au.uuid::text AS admin_user_uuid,
		  au.email AS admin_email,
		  al.actor_type,
		  COUNT(*) AS count
		FROM admin_audit_logs al
		LEFT JOIN admin_users au ON au.id = al.admin_user_id
		WHERE al.created_at >= ? AND al.created_at < ?
		GROUP BY au.uuid, au.email, al.actor_type
		ORDER BY count DESC, admin_email ASC NULLS LAST
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	roleChanges, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM admin_audit_logs
		WHERE created_at >= ? AND created_at < ?
		  AND action IN ('ADMIN_ROLE_ASSIGNED', 'ADMIN_ROLE_REMOVED')
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	restrictionActions, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM admin_audit_logs
		WHERE created_at >= ? AND created_at < ?
		  AND action IN ('USER_RESTRICTION_CREATED', 'USER_RESTRICTION_REVOKED', 'USER_RESTRICTION_EXPIRED')
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	reportReviewActions, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM admin_audit_logs
		WHERE created_at >= ? AND created_at < ?
		  AND action = 'REPORT_REVIEWED'
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	return &AdminActivityAnalyticsResponse{
		Period:              period,
		AuditActionsByType:  byType,
		AuditActionsByAdmin: byAdmin,
		AdminLogins:         nil,
		RoleChanges:         roleChanges,
		RestrictionActions:  restrictionActions,
		ReportReviewActions: reportReviewActions,
	}, nil
}

func (s *Service) SubscriptionPaymentAnalytics(ctx context.Context, query AnalyticsQuery) (*SubscriptionPaymentAnalyticsResponse, error) {
	period, err := normalizeAnalyticsRange(query.From, query.To, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	var totals SubscriptionPaymentTotals
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  COUNT(*) AS payment_count,
		  COUNT(*) FILTER (WHERE status = 'PENDING') AS pending_count,
		  COUNT(*) FILTER (WHERE status = 'UNDER_REVIEW') AS under_review_count,
		  COUNT(*) FILTER (WHERE status = 'APPROVED') AS approved_count,
		  COUNT(*) FILTER (WHERE status = 'REJECTED') AS rejected_count,
		  COUNT(*) FILTER (WHERE status = 'CANCELED') AS canceled_count,
		  COALESCE(SUM(price_amount_snapshot), 0) AS total_amount,
		  COALESCE(SUM(price_amount_snapshot) FILTER (WHERE status = 'APPROVED'), 0) AS approved_amount,
		  COALESCE(SUM(price_amount_snapshot) FILTER (WHERE status = 'REJECTED'), 0) AS rejected_amount
		FROM manual_payment_requests
		WHERE created_at >= ? AND created_at < ?
	`, period.From, period.To).Scan(&totals).Error; err != nil {
		return nil, err
	}
	statusBreakdown, err := s.paymentBreakdownRows(ctx, `
		SELECT status AS key, COUNT(*) AS count, COALESCE(SUM(price_amount_snapshot), 0) AS amount
		FROM manual_payment_requests
		WHERE created_at >= ? AND created_at < ?
		GROUP BY status
		ORDER BY status ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	providerBreakdown, err := s.paymentBreakdownRows(ctx, `
		SELECT payment_provider AS key, COUNT(*) AS count, COALESCE(SUM(price_amount_snapshot), 0) AS amount
		FROM manual_payment_requests
		WHERE created_at >= ? AND created_at < ?
		GROUP BY payment_provider
		ORDER BY count DESC, payment_provider ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	planBreakdown, err := s.paymentPlanBreakdownRows(ctx, `
		SELECT
		  plan_id::text AS plan_id,
		  plan_code_snapshot AS plan_code,
		  plan_name_snapshot AS plan_name,
		  COUNT(*) AS count,
		  COALESCE(SUM(price_amount_snapshot), 0) AS amount
		FROM manual_payment_requests
		WHERE created_at >= ? AND created_at < ?
		GROUP BY plan_id, plan_code_snapshot, plan_name_snapshot
		ORDER BY count DESC, plan_name_snapshot ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	reviewSLA, err := s.paymentReviewSLA(ctx, period)
	if err != nil {
		return nil, err
	}
	return &SubscriptionPaymentAnalyticsResponse{
		Period:            period,
		Totals:            totals,
		StatusBreakdown:   statusBreakdown,
		ProviderBreakdown: providerBreakdown,
		PlanBreakdown:     planBreakdown,
		ReviewSLA:         reviewSLA,
	}, nil
}

func (s *Service) reportAnalyticsForPeriod(ctx context.Context, period AnalyticsPeriod) (*ReportAnalyticsResponse, error) {
	created, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE created_at >= ? AND created_at < ?`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	pending, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'PENDING'`)
	if err != nil {
		return nil, err
	}
	reviewed, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'REVIEWED' AND reviewed_at >= ? AND reviewed_at < ?`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	dismissed, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'DISMISSED' AND reviewed_at >= ? AND reviewed_at < ?`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	actioned, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'ACTIONED' AND reviewed_at >= ? AND reviewed_at < ?`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	byTargetType, err := s.countRows(ctx, `
		SELECT target_type AS key, COUNT(*) AS count
		FROM reports
		WHERE created_at >= ? AND created_at < ?
		GROUP BY target_type
		ORDER BY target_type ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	byReason, err := s.reportReasonRows(ctx, `
		SELECT rr.reason_code, rr.title AS reason_title, COUNT(*) AS count
		FROM reports r
		JOIN report_reasons rr ON rr.id = r.reason_id
		WHERE r.created_at >= ? AND r.created_at < ?
		GROUP BY rr.reason_code, rr.title
		ORDER BY count DESC, rr.reason_code ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	avgReview, err := s.floatPtrRaw(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (reviewed_at - created_at)) / 60.0)
		FROM reports
		WHERE reviewed_at IS NOT NULL
		  AND reviewed_at >= ? AND reviewed_at < ?
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	return &ReportAnalyticsResponse{
		Period:               period,
		ReportsCreated:       created,
		PendingReports:       pending,
		ReviewedReports:      reviewed,
		DismissedReports:     dismissed,
		ActionedReports:      actioned,
		ReportsByTargetType:  byTargetType,
		ReportsByReason:      byReason,
		AverageReviewMinutes: avgReview,
	}, nil
}

func (s *Service) restrictionAnalyticsForPeriod(ctx context.Context, period AnalyticsPeriod) (*RestrictionAnalyticsResponse, error) {
	created, err := s.countRaw(ctx, `SELECT COUNT(*) FROM user_restrictions WHERE created_at >= ? AND created_at < ?`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	revoked, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM user_restrictions
		WHERE status = 'REVOKED'
		  AND revoked_at >= ? AND revoked_at < ?
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	active, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM user_restrictions
		WHERE status = 'ACTIVE'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		return nil, err
	}
	expired, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM user_restrictions
		WHERE status = 'EXPIRED'
		  AND updated_at >= ? AND updated_at < ?
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	byType, err := s.countRows(ctx, `
		SELECT restriction_type AS key, COUNT(*) AS count
		FROM user_restrictions
		WHERE created_at >= ? AND created_at < ?
		GROUP BY restriction_type
		ORDER BY count DESC, restriction_type ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	byAdmin, err := s.adminActorRows(ctx, `
		SELECT
		  au.uuid::text AS admin_user_uuid,
		  au.email AS admin_email,
		  'ADMIN' AS actor_type,
		  COUNT(*) AS count
		FROM user_restrictions ur
		JOIN admin_users au ON au.id = ur.created_by_admin_user_id
		WHERE ur.created_at >= ? AND ur.created_at < ?
		GROUP BY au.uuid, au.email
		ORDER BY count DESC, au.email ASC
	`, period.From, period.To)
	if err != nil {
		return nil, err
	}
	return &RestrictionAnalyticsResponse{
		Period:              period,
		RestrictionsCreated: created,
		RestrictionsRevoked: revoked,
		ActiveRestrictions:  active,
		ExpiredRestrictions: expired,
		RestrictionsByType:  byType,
		RestrictionsByAdmin: byAdmin,
	}, nil
}

func (s *Service) dashboardUsersSummary(ctx context.Context, todayStart time.Time, tomorrowStart time.Time, monthStart time.Time) (*DashboardUsersSummary, error) {
	total, err := s.countRaw(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}
	newToday, err := s.countRaw(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?`, todayStart, tomorrowStart)
	if err != nil {
		return nil, err
	}
	newThisMonth, err := s.countRaw(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?`, monthStart, tomorrowStart)
	if err != nil {
		return nil, err
	}
	return &DashboardUsersSummary{Total: total, NewToday: newToday, NewThisMonth: newThisMonth}, nil
}

func (s *Service) dashboardReportsSummary(ctx context.Context, todayStart time.Time, tomorrowStart time.Time) (*DashboardReportsSummary, error) {
	pending, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'PENDING'`)
	if err != nil {
		return nil, err
	}
	reviewedToday, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'REVIEWED' AND reviewed_at >= ? AND reviewed_at < ?`, todayStart, tomorrowStart)
	if err != nil {
		return nil, err
	}
	actionedToday, err := s.countRaw(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'ACTIONED' AND reviewed_at >= ? AND reviewed_at < ?`, todayStart, tomorrowStart)
	if err != nil {
		return nil, err
	}
	return &DashboardReportsSummary{Pending: pending, ReviewedToday: reviewedToday, ActionedToday: actionedToday}, nil
}

func (s *Service) dashboardRestrictionsSummary(ctx context.Context) (*DashboardRestrictionsSummary, error) {
	active, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM user_restrictions
		WHERE status = 'ACTIVE'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		return nil, err
	}
	fullPlatformBans, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM user_restrictions
		WHERE status = 'ACTIVE'
		  AND restriction_type = 'FULL_PLATFORM_BAN'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		return nil, err
	}
	commentBans, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM user_restrictions
		WHERE status = 'ACTIVE'
		  AND restriction_type = 'COMMENT_BAN'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`)
	if err != nil {
		return nil, err
	}
	return &DashboardRestrictionsSummary{Active: active, FullPlatformBans: fullPlatformBans, CommentBans: commentBans}, nil
}

func (s *Service) dashboardAdminSummary(ctx context.Context, todayStart time.Time, tomorrowStart time.Time) (*DashboardAdminSummary, error) {
	activeAdmins, err := s.countRaw(ctx, `SELECT COUNT(*) FROM admin_users WHERE status = 'ACTIVE'`)
	if err != nil {
		return nil, err
	}
	auditActionsToday, err := s.countRaw(ctx, `SELECT COUNT(*) FROM admin_audit_logs WHERE created_at >= ? AND created_at < ?`, todayStart, tomorrowStart)
	if err != nil {
		return nil, err
	}
	return &DashboardAdminSummary{ActiveAdmins: activeAdmins, AuditActionsToday: auditActionsToday}, nil
}

func (s *Service) dashboardSubscriptionPaymentsSummary(ctx context.Context, todayStart time.Time, tomorrowStart time.Time) (*DashboardSubscriptionPaymentsSummary, error) {
	var summary DashboardSubscriptionPaymentsSummary
	err := s.db.WithContext(ctx).Raw(`
		SELECT
		  COUNT(*) FILTER (WHERE status = 'PENDING') AS pending,
		  COUNT(*) FILTER (WHERE status = 'APPROVED' AND reviewed_at >= ? AND reviewed_at < ?) AS approved_today,
		  COUNT(*) FILTER (WHERE status = 'REJECTED' AND reviewed_at >= ? AND reviewed_at < ?) AS rejected_today,
		  COALESCE(SUM(price_amount_snapshot) FILTER (WHERE status = 'APPROVED' AND reviewed_at >= ? AND reviewed_at < ?), 0) AS approved_amount_today
		FROM manual_payment_requests
	`, todayStart, tomorrowStart, todayStart, tomorrowStart, todayStart, tomorrowStart).Scan(&summary).Error
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (s *Service) paymentReviewSLA(ctx context.Context, period AnalyticsPeriod) (PaymentReviewSLA, error) {
	averageReviewMinutes, err := s.floatPtrRaw(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (reviewed_at - submitted_at)) / 60.0)
		FROM manual_payment_requests
		WHERE reviewed_at IS NOT NULL
		  AND reviewed_at >= ? AND reviewed_at < ?
	`, period.From, period.To)
	if err != nil {
		return PaymentReviewSLA{}, err
	}
	pendingOlderThan24h, err := s.countRaw(ctx, `
		SELECT COUNT(*)
		FROM manual_payment_requests
		WHERE status IN ('PENDING', 'UNDER_REVIEW')
		  AND submitted_at < ?
	`, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		return PaymentReviewSLA{}, err
	}
	return PaymentReviewSLA{AverageReviewMinutes: averageReviewMinutes, PendingOlderThan24h: pendingOlderThan24h}, nil
}

func (s *Service) countRaw(ctx context.Context, query string, args ...any) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&count).Error
	return count, err
}

func (s *Service) floatPtrRaw(ctx context.Context, query string, args ...any) (*float64, error) {
	var value sql.NullFloat64
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&value).Error; err != nil {
		return nil, err
	}
	if !value.Valid {
		return nil, nil
	}
	return &value.Float64, nil
}

func (s *Service) countRows(ctx context.Context, query string, args ...any) ([]AnalyticsCount, error) {
	rows := make([]AnalyticsCount, 0)
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) reportReasonRows(ctx context.Context, query string, args ...any) ([]ReportReasonCount, error) {
	rows := make([]ReportReasonCount, 0)
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) adminActorRows(ctx context.Context, query string, args ...any) ([]AdminActorCount, error) {
	rows := make([]AdminActorCount, 0)
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) paymentBreakdownRows(ctx context.Context, query string, args ...any) ([]PaymentBreakdown, error) {
	rows := make([]PaymentBreakdown, 0)
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) paymentPlanBreakdownRows(ctx context.Context, query string, args ...any) ([]PaymentPlanBreakdown, error) {
	rows := make([]PaymentPlanBreakdown, 0)
	err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (s *Service) hasAnyPermission(ctx context.Context, adminID uint64, permissions ...string) (bool, error) {
	for _, permission := range permissions {
		allowed, err := s.HasPermission(ctx, adminID, permission)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func startOfUTCDay(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

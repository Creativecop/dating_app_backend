package admin

import (
	"errors"
	"net/http"
)

var (
	ErrUnauthorized           = errors.New("unauthorized")
	ErrInvalidRefresh         = errors.New("invalid refresh token")
	ErrInactiveAdmin          = errors.New("inactive admin")
	ErrPermissionDenied       = errors.New("permission denied")
	ErrAdminUserNotFound      = errors.New("admin user not found")
	ErrPasswordMismatch       = errors.New("password mismatch")
	ErrPasswordChangeRequired = errors.New("password change required")
)

const (
	CodeValidation                    = "VALIDATION_ERROR"
	CodeUnauthorized                  = "ADMIN_UNAUTHORIZED"
	CodePermissionDenied              = "ADMIN_PERMISSION_DENIED"
	CodeRoleRequired                  = "ADMIN_ROLE_REQUIRED"
	CodeRoleNotFound                  = "ADMIN_ROLE_NOT_FOUND"
	CodeRoleAlreadyAssigned           = "ADMIN_ROLE_ALREADY_ASSIGNED"
	CodeRoleAssignmentNotAllowed      = "ADMIN_ROLE_ASSIGNMENT_NOT_ALLOWED"
	CodeReasonRequired                = "ADMIN_REASON_REQUIRED"
	CodeIdempotencyKeyRequired        = "ADMIN_IDEMPOTENCY_KEY_REQUIRED"
	CodeAuditWriteFailed              = "ADMIN_AUDIT_WRITE_FAILED"
	CodeTargetNotFound                = "ADMIN_TARGET_NOT_FOUND"
	CodeActionNotAllowed              = "ADMIN_ACTION_NOT_ALLOWED"
	CodeCannotModifySuperAdmin        = "ADMIN_CANNOT_MODIFY_SUPER_ADMIN"
	CodeAdminModuleNotAvailable       = "ADMIN_MODULE_NOT_AVAILABLE"
	CodeUserRestrictionNotFound       = "USER_RESTRICTION_NOT_FOUND"
	CodeUserAlreadyRestricted         = "USER_ALREADY_RESTRICTED"
	CodeUserNotRestricted             = "USER_NOT_RESTRICTED"
	CodeLiveForceEndNotAllowed        = "LIVE_FORCE_END_NOT_ALLOWED"
	CodeReportNotFound                = "REPORT_NOT_FOUND"
	CodeReportAlreadyReviewed         = "REPORT_ALREADY_REVIEWED"
	CodeReportActionNotAllowed        = "REPORT_ACTION_NOT_ALLOWED"
	CodeCommentNotFound               = "COMMENT_NOT_FOUND"
	CodeCommentHideNotAllowed         = "COMMENT_HIDE_NOT_ALLOWED"
	CodeWalletAuditNotAllowed         = "WALLET_AUDIT_NOT_ALLOWED"
	CodeWalletAdjustmentNotAllowed    = "WALLET_ADJUSTMENT_NOT_ALLOWED"
	CodeWalletReversalNotAllowed      = "WALLET_REVERSAL_NOT_ALLOWED"
	CodeGiftManagementNotAllowed      = "GIFT_MANAGEMENT_NOT_ALLOWED"
	CodeAgencyAdminActionNotAllowed   = "AGENCY_ADMIN_ACTION_NOT_ALLOWED"
	CodeResellerAdminActionNotAllowed = "RESELLER_ADMIN_ACTION_NOT_ALLOWED"
	CodeSuspiciousFlagNotFound        = "SUSPICIOUS_FLAG_NOT_FOUND"
	CodeSuspiciousFlagAlreadyReviewed = "SUSPICIOUS_FLAG_ALREADY_REVIEWED"
	CodeAnalyticsAccessDenied         = "ANALYTICS_ACCESS_DENIED"
	CodePasswordChangeRequired        = "ADMIN_PASSWORD_CHANGE_REQUIRED"
	CodeUserNotFound                  = "USER_NOT_FOUND"
	CodeNotFound                      = "ADMIN_NOT_FOUND"
	CodeConflict                      = "ADMIN_CONFLICT"
	CodeUserRestrictionExpired        = "USER_RESTRICTION_EXPIRED"
	CodeUserRestrictionReasonRequired = "USER_RESTRICTION_REASON_REQUIRED"
	CodeUserActionRestricted          = "USER_ACTION_RESTRICTED"
	CodeLiveNotFound                  = "LIVE_NOT_FOUND"
	CodeLiveAlreadyEnded              = "LIVE_ALREADY_ENDED"
)

type ServiceError struct {
	Status  int
	Code    string
	Message string
	Details any
}

func (e *ServiceError) Error() string {
	return e.Message
}

func validationError(message string, details any) *ServiceError {
	return &ServiceError{Status: http.StatusBadRequest, Code: CodeValidation, Message: message, Details: details}
}

func unauthorizedError() *ServiceError {
	return &ServiceError{Status: http.StatusUnauthorized, Code: CodeUnauthorized, Message: "Unauthorized"}
}

func permissionDeniedError() *ServiceError {
	return &ServiceError{Status: http.StatusForbidden, Code: CodePermissionDenied, Message: "Permission denied"}
}

func forbiddenError(code string, message string, details any) *ServiceError {
	return &ServiceError{Status: http.StatusForbidden, Code: code, Message: message, Details: details}
}

func notFoundError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeNotFound, Message: message}
}

func targetNotFoundError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeTargetNotFound, Message: message}
}

func conflictError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodeConflict, Message: message}
}

func conflictCodeError(code string, message string) *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: code, Message: message}
}

func actionNotAllowedError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodeActionNotAllowed, Message: message}
}

func passwordChangeRequiredError() *ServiceError {
	return &ServiceError{Status: http.StatusForbidden, Code: CodePasswordChangeRequired, Message: "Password change required"}
}

func AsServiceError(err error) (*ServiceError, bool) {
	serviceErr, ok := err.(*ServiceError)
	return serviceErr, ok
}

func PublicErrorMessage(err error) string {
	if errors.Is(err, ErrInactiveAdmin) {
		return "Admin account is inactive."
	}
	return "Unauthorized."
}

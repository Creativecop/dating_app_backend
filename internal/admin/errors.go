package admin

import (
	"errors"
	"net/http"
)

var (
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInvalidRefresh    = errors.New("invalid refresh token")
	ErrInactiveAdmin     = errors.New("inactive admin")
	ErrPermissionDenied  = errors.New("permission denied")
	ErrAdminUserNotFound = errors.New("admin user not found")
	ErrPasswordMismatch  = errors.New("password mismatch")
)

const (
	CodeValidation       = "VALIDATION_ERROR"
	CodeUnauthorized     = "ADMIN_UNAUTHORIZED"
	CodePermissionDenied = "ADMIN_PERMISSION_DENIED"
	CodeNotFound         = "ADMIN_NOT_FOUND"
	CodeConflict         = "ADMIN_CONFLICT"
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

func notFoundError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeNotFound, Message: message}
}

func conflictError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodeConflict, Message: message}
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

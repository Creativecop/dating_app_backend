package chat

import "net/http"

const (
	CodeValidation             = "VALIDATION_ERROR"
	CodeNotFound               = "CHAT_NOT_FOUND"
	CodeForbidden              = "CHAT_FORBIDDEN"
	CodeIdempotencyKeyConflict = "IDEMPOTENCY_KEY_CONFLICT"
	CodeUnsupportedEvent       = "UNSUPPORTED_EVENT"
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

func forbiddenError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusForbidden, Code: CodeForbidden, Message: message}
}

func notFoundError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeNotFound, Message: message}
}

func idempotencyConflictError() *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodeIdempotencyKeyConflict, Message: "Idempotency key was reused with different payload"}
}

func AsServiceError(err error) (*ServiceError, bool) {
	serviceErr, ok := err.(*ServiceError)
	return serviceErr, ok
}

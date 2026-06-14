package discovery

import "net/http"

const (
	CodeValidation             = "VALIDATION_ERROR"
	CodeNotFound               = "DISCOVERY_NOT_FOUND"
	CodeDiscoveryNotReady      = "DISCOVERY_NOT_READY"
	CodeTargetNotDiscoverable  = "TARGET_NOT_DISCOVERABLE"
	CodeActionNotAllowed       = "ACTION_NOT_ALLOWED"
	CodeActionAlreadyExists    = "ACTION_ALREADY_EXISTS"
	CodeIdempotencyKeyConflict = "IDEMPOTENCY_KEY_CONFLICT"
	CodeSuperLikeLimitReached  = "SUPER_LIKE_LIMIT_REACHED"
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

func (e *ServiceError) StatusCode() int {
	return e.Status
}

func (e *ServiceError) ErrorCode() string {
	return e.Code
}

func (e *ServiceError) ErrorDetails() any {
	return e.Details
}

func validationError(message string, details any) *ServiceError {
	return &ServiceError{Status: http.StatusBadRequest, Code: CodeValidation, Message: message, Details: details}
}

func forbiddenError(message string, code string, details any) *ServiceError {
	return &ServiceError{Status: http.StatusForbidden, Code: code, Message: message, Details: details}
}

func discoveryNotReadyError(details any) *ServiceError {
	return forbiddenError("Discovery is not ready", CodeDiscoveryNotReady, details)
}

func targetNotDiscoverableError() *ServiceError {
	return forbiddenError("Target is not available for discovery", CodeTargetNotDiscoverable, nil)
}

func actionNotAllowedError() *ServiceError {
	return forbiddenError("Action is not allowed", CodeActionNotAllowed, nil)
}

func actionAlreadyExistsError() *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodeActionAlreadyExists, Message: "Action already exists for this user"}
}

func idempotencyKeyConflictError() *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodeIdempotencyKeyConflict, Message: "Idempotency key was reused with different payload"}
}

func superLikeLimitReachedError() *ServiceError {
	return &ServiceError{Status: http.StatusTooManyRequests, Code: CodeSuperLikeLimitReached, Message: "Super Like limit reached"}
}

func notFoundError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeNotFound, Message: message}
}

func AsServiceError(err error) (*ServiceError, bool) {
	serviceErr, ok := err.(*ServiceError)
	return serviceErr, ok
}

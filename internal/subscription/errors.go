package subscription

import "net/http"

const (
	CodeValidation                    = "VALIDATION_ERROR"
	CodeNotFound                      = "SUBSCRIPTION_NOT_FOUND"
	CodeConflict                      = "SUBSCRIPTION_CONFLICT"
	CodePaymentRequestAlreadyRejected = "PAYMENT_REQUEST_ALREADY_REJECTED"
	CodePaymentRequestAlreadyApproved = "PAYMENT_REQUEST_ALREADY_APPROVED"
	CodeLikeLimitReached              = "LIKE_LIMIT_REACHED"
	CodeSuperLikeLimitReached         = "SUPER_LIKE_LIMIT_REACHED"
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

func notFoundError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeNotFound, Message: message}
}

func conflictError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodeConflict, Message: message}
}

func paymentRejectedError() *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodePaymentRequestAlreadyRejected, Message: "Payment request is already rejected"}
}

func paymentApprovedError() *ServiceError {
	return &ServiceError{Status: http.StatusConflict, Code: CodePaymentRequestAlreadyApproved, Message: "Payment request is already approved"}
}

func likeLimitReachedError() *ServiceError {
	return &ServiceError{Status: http.StatusTooManyRequests, Code: CodeLikeLimitReached, Message: "Daily like limit reached"}
}

func superLikeLimitReachedError() *ServiceError {
	return &ServiceError{Status: http.StatusTooManyRequests, Code: CodeSuperLikeLimitReached, Message: "Super Like limit reached"}
}

func AsServiceError(err error) (*ServiceError, bool) {
	serviceErr, ok := err.(*ServiceError)
	return serviceErr, ok
}

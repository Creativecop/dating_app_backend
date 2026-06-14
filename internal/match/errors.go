package match

import "net/http"

const (
	CodeValidation = "VALIDATION_ERROR"
	CodeNotFound   = "MATCH_NOT_FOUND"
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

func notFoundError() *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeNotFound, Message: "Match not found"}
}

func AsServiceError(err error) (*ServiceError, bool) {
	serviceErr, ok := err.(*ServiceError)
	return serviceErr, ok
}

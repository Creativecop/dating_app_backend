package profile

import "net/http"

const (
	CodeValidation        = "VALIDATION_ERROR"
	CodeProfileIncomplete = "PROFILE_INCOMPLETE"
	CodeNotFound          = "NOT_FOUND"
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
	return &ServiceError{
		Status:  http.StatusBadRequest,
		Code:    CodeValidation,
		Message: message,
		Details: details,
	}
}

func incompleteError(missing []string) *ServiceError {
	return &ServiceError{
		Status:  http.StatusBadRequest,
		Code:    CodeProfileIncomplete,
		Message: "Profile is incomplete",
		Details: map[string]any{"missing": missing},
	}
}

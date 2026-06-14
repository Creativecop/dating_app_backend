package media

import "net/http"

const (
	CodeValidation    = "VALIDATION_ERROR"
	CodeNotFound      = "MEDIA_NOT_FOUND"
	CodeForbidden     = "MEDIA_FORBIDDEN"
	CodeUploadFailed  = "MEDIA_UPLOAD_FAILED"
	CodeProcessFailed = "MEDIA_PROCESSING_FAILED"
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

func notFoundError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusNotFound, Code: CodeNotFound, Message: message}
}

func forbiddenError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusForbidden, Code: CodeForbidden, Message: message}
}

func uploadFailedError(message string) *ServiceError {
	return &ServiceError{Status: http.StatusInternalServerError, Code: CodeUploadFailed, Message: message}
}

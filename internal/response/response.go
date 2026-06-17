package response

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Body struct {
	Success    bool       `json:"success"`
	StatusCode int        `json:"statusCode"`
	Message    string     `json:"message"`
	Data       any        `json:"data,omitempty"`
	Error      *ErrorBody `json:"error,omitempty"`
}

type RateLimitBody struct {
	Success          bool       `json:"success"`
	StatusCode       int        `json:"statusCode"`
	StatusCodeCompat int        `json:"status_code"`
	Message          string     `json:"message"`
	Error            *ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code              string `json:"code"`
	RetryAfterSeconds *int64 `json:"retryAfterSeconds,omitempty"`
	Details           any    `json:"details,omitempty"`
}

func Success(c *gin.Context, status int, message string, data any) {
	if message == "" {
		message = http.StatusText(status)
	}
	c.JSON(status, Body{
		Success:    true,
		StatusCode: status,
		Message:    message,
		Data:       data,
	})
}

func Error(c *gin.Context, status int, message string, code string, details any) {
	if message == "" {
		message = http.StatusText(status)
	}
	if code == "" {
		code = "ERROR"
	}
	c.JSON(status, Body{
		Success:    false,
		StatusCode: status,
		Message:    message,
		Error: &ErrorBody{
			Code:    code,
			Details: details,
		},
	})
}

func BadRequest(c *gin.Context, message string, details any) {
	Error(c, http.StatusBadRequest, message, "BAD_REQUEST", details)
}

func Validation(c *gin.Context, details any) {
	Error(c, http.StatusBadRequest, "Validation failed", "VALIDATION_ERROR", details)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message, "UNAUTHORIZED", nil)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message, "FORBIDDEN", nil)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message, "NOT_FOUND", nil)
}

func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, message, "RATE_LIMITED", nil)
}

func RateLimited(c *gin.Context, retryAfterSeconds int64) {
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}
	c.Header("Retry-After", strconv.FormatInt(retryAfterSeconds, 10))
	c.JSON(http.StatusTooManyRequests, RateLimitBody{
		Success:          false,
		StatusCode:       http.StatusTooManyRequests,
		StatusCodeCompat: http.StatusTooManyRequests,
		Message:          "Too many requests",
		Error: &ErrorBody{
			Code:              "RATE_LIMITED",
			RetryAfterSeconds: &retryAfterSeconds,
		},
	})
}

func Internal(c *gin.Context) {
	Error(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_SERVER_ERROR", nil)
}

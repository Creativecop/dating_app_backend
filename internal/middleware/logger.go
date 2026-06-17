package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxLoggedBodyBytes int64 = 32 * 1024

type responseCaptureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *responseCaptureWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *responseCaptureWriter) WriteString(data string) (int, error) {
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *responseCaptureWriter) capture(data []byte) {
	remaining := int(maxLoggedBodyBytes) - w.body.Len()
	if remaining <= 0 {
		return
	}
	if len(data) > remaining {
		data = data[:remaining]
	}
	_, _ = w.body.Write(data)
}

func Logger(appEnv ...string) gin.HandlerFunc {
	env := ""
	if len(appEnv) > 0 {
		env = strings.ToLower(strings.TrimSpace(appEnv[0]))
	}
	logBodies := env != "production"
	return func(c *gin.Context) {
		start := time.Now()
		requestBody := ""
		if logBodies {
			requestBody = safeRequestBody(c)
		}

		capture := &responseCaptureWriter{ResponseWriter: c.Writer}
		c.Writer = capture

		c.Next()

		latency := time.Since(start)
		responseBody := ""
		if logBodies {
			responseBody = safeResponseBody(c, capture)
		}

		log.Printf("%s", prettyHTTPLog(
			GetRequestID(c),
			c.Request.Method,
			safeRequestPath(c),
			c.Writer.Status(),
			latency,
			c.ClientIP(),
			c.FullPath(),
			contextUintField(c, "auth_user", "UserID"),
			contextUintField(c, "admin_user", "AdminUserID"),
			responseErrorCode(capture.body.Bytes()),
			requestBody,
			responseBody,
		))
	}
}

func safeRequestPath(c *gin.Context) string {
	if c.Request == nil || c.Request.URL == nil {
		return ""
	}
	if c.Request.URL.RawQuery == "" {
		return c.Request.URL.Path
	}
	values, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		return c.Request.URL.Path
	}
	if _, ok := values["token"]; ok {
		values.Set("token", "***")
	}
	return c.Request.URL.Path + "?" + values.Encode()
}

func safeRequestBody(c *gin.Context) string {
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}
	contentType := c.GetHeader("Content-Type")
	if !isLoggableContentType(contentType) {
		return skippedBodyMessage(contentType, c.Request.ContentLength)
	}
	if c.Request.ContentLength < 0 || c.Request.ContentLength > maxLoggedBodyBytes {
		return skippedBodyMessage(contentType, c.Request.ContentLength)
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "<failed to read request body>"
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(bytes.TrimSpace(body)) == 0 {
		return ""
	}
	return sanitizeBody(body, true)
}

func safeResponseBody(c *gin.Context, writer *responseCaptureWriter) string {
	contentType := writer.Header().Get("Content-Type")
	if !isLoggableContentType(contentType) {
		return skippedBodyMessage(contentType, int64(writer.Size()))
	}
	if writer.body.Len() == 0 {
		return ""
	}
	value := sanitizeBody(writer.body.Bytes(), false)
	if writer.Size() > int(maxLoggedBodyBytes) {
		value += " <truncated>"
	}
	return value
}

func isLoggableContentType(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "application/problem+json") ||
		strings.Contains(contentType, "text/plain")
}

func skippedBodyMessage(contentType string, size int64) string {
	if strings.TrimSpace(contentType) == "" && size <= 0 {
		return ""
	}
	if size > 0 {
		return fmt.Sprintf("<skipped body content_type=%q size=%d>", contentType, size)
	}
	return fmt.Sprintf("<skipped body content_type=%q>", contentType)
}

func sanitizeBody(body []byte, request bool) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}
	if !json.Valid(trimmed) {
		return string(trimmed)
	}
	var value any
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return string(trimmed)
	}
	redactJSON(value, request)
	encoded, err := json.Marshal(value)
	if err != nil {
		return string(trimmed)
	}
	return string(encoded)
}

func redactJSON(value any, request bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if isSensitiveLogField(key, request) {
				typed[key] = "***"
				continue
			}
			redactJSON(item, request)
		}
	case []any:
		for _, item := range typed {
			redactJSON(item, request)
		}
	}
}

func isSensitiveLogField(field string, request bool) bool {
	key := strings.ToLower(strings.TrimSpace(field))
	if key == "" {
		return false
	}
	sensitiveContains := []string{
		"token",
		"password",
		"secret",
		"authorization",
		"fcm",
		"phone",
		"email",
		"latitude",
		"longitude",
		"payment_reference",
		"paymentreference",
		"screenshot",
		"internal_ops",
		"internalops",
		"ops_key",
		"opskey",
	}
	for _, part := range sensitiveContains {
		if strings.Contains(key, part) {
			return true
		}
	}
	if request && (key == "code" || strings.Contains(key, "otp")) {
		return true
	}
	return false
}

func prettyHTTPLog(requestID string, method string, path string, status int, latency time.Duration, ip string, route string, userID *uint64, adminUserID *uint64, errorCode string, requestBody string, responseBody string) string {
	record := map[string]any{
		"event":       "http_request",
		"request_id":  requestID,
		"method":      method,
		"path":        path,
		"route":       route,
		"status":      status,
		"duration_ms": latency.Milliseconds(),
		"ip":          ip,
	}
	if errorCode != "" {
		record["error_code"] = errorCode
	}
	if userID != nil {
		record["user_id"] = *userID
	}
	if adminUserID != nil {
		record["admin_user_id"] = *adminUserID
	}
	if requestBody != "" {
		record["request"] = requestBody
	}
	if responseBody != "" {
		record["response"] = responseBody
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return fmt.Sprintf(`{"event":"http_request","request_id":%q,"method":%q,"path":%q,"status":%d}`, requestID, method, path, status)
	}
	return string(encoded)
}

func responseErrorCode(body []byte) string {
	if len(bytes.TrimSpace(body)) == 0 || !json.Valid(bytes.TrimSpace(body)) {
		return ""
	}
	var payload struct {
		Error *struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(body), &payload); err != nil || payload.Error == nil {
		return ""
	}
	return payload.Error.Code
}

func contextUintField(c *gin.Context, key string, field string) *uint64 {
	value, ok := c.Get(key)
	if !ok {
		return nil
	}
	reflected := reflect.ValueOf(value)
	if reflected.Kind() == reflect.Pointer {
		if reflected.IsNil() {
			return nil
		}
		reflected = reflected.Elem()
	}
	if reflected.Kind() != reflect.Struct {
		return nil
	}
	fieldValue := reflected.FieldByName(field)
	if !fieldValue.IsValid() {
		return nil
	}
	switch fieldValue.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value := fieldValue.Uint()
		return &value
	default:
		return nil
	}
}

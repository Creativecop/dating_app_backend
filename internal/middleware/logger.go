package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
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

func prettyHTTPLog(requestID string, method string, path string, status int, latency time.Duration, ip string, requestBody string, responseBody string) string {
	var builder strings.Builder
	builder.WriteString("\n")
	builder.WriteString("========== HTTP ==========\n")
	builder.WriteString(fmt.Sprintf("--> %s %s\n", method, path))
	builder.WriteString(fmt.Sprintf("    request_id: %s\n", requestID))
	builder.WriteString(fmt.Sprintf("    ip: %s\n", ip))
	if requestBody != "" {
		builder.WriteString(fmt.Sprintf("    request: %s\n", requestBody))
	}
	builder.WriteString(fmt.Sprintf("<-- %d %s %s\n", status, method, path))
	builder.WriteString(fmt.Sprintf("    latency: %s\n", latency.String()))
	if responseBody != "" {
		builder.WriteString(fmt.Sprintf("    response: %s\n", responseBody))
	}
	builder.WriteString("==========================")
	return builder.String()
}

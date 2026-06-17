package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/response"
)

type RateLimiter struct {
	client  *goredis.Client
	enabled bool
}

type RateLimitRule struct {
	Scope      string
	Limit      int
	Window     time.Duration
	Identifier RateLimitIdentifier
	FailClosed bool
}

type RateLimitIdentifier func(*gin.Context) string

type rateLimitResult struct {
	Allowed           bool
	RetryAfterSeconds int64
}

var rateLimitScript = goredis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
local ttl = redis.call("PTTL", KEYS[1])
return {current, ttl}
`)

func NewRateLimiter(client *goredis.Client, enabled bool) *RateLimiter {
	return &RateLimiter{client: client, enabled: enabled}
}

func (l *RateLimiter) Limit(rules ...RateLimitRule) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil || !l.enabled {
			c.Next()
			return
		}
		for _, rule := range rules {
			if rule.Limit <= 0 || rule.Window <= 0 || rule.Identifier == nil || strings.TrimSpace(rule.Scope) == "" {
				continue
			}
			identifier := strings.TrimSpace(rule.Identifier(c))
			if identifier == "" {
				continue
			}
			result, err := l.allow(c.Request.Context(), rule, identifier)
			if err != nil {
				if rule.FailClosed {
					log.Printf("rate_limit_unavailable request_id=%s scope=%s fail_closed=true error=%v", GetRequestID(c), rule.Scope, err)
					response.RateLimited(c, int64(rule.Window.Seconds()))
					c.Abort()
					return
				}
				log.Printf("rate_limit_unavailable request_id=%s scope=%s fail_closed=false error=%v", GetRequestID(c), rule.Scope, err)
				continue
			}
			if !result.Allowed {
				response.RateLimited(c, result.RetryAfterSeconds)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func (l *RateLimiter) allow(ctx context.Context, rule RateLimitRule, identifier string) (rateLimitResult, error) {
	if l.client == nil {
		return rateLimitResult{}, fmt.Errorf("redis client is nil")
	}
	key := rateLimitKey(rule.Scope, identifier, rule.Window)
	result, err := rateLimitScript.Run(ctx, l.client, []string{key}, rule.Window.Milliseconds()).Result()
	if err != nil {
		return rateLimitResult{}, err
	}
	values, ok := result.([]any)
	if !ok || len(values) != 2 {
		return rateLimitResult{}, fmt.Errorf("unexpected redis rate limit result")
	}
	count, err := redisInt64(values[0])
	if err != nil {
		return rateLimitResult{}, err
	}
	ttlMillis, err := redisInt64(values[1])
	if err != nil {
		return rateLimitResult{}, err
	}
	retryAfterSeconds := int64(1)
	if ttlMillis > 0 {
		retryAfterSeconds = (ttlMillis + 999) / 1000
	}
	return rateLimitResult{Allowed: count <= int64(rule.Limit), RetryAfterSeconds: retryAfterSeconds}, nil
}

func rateLimitKey(scope string, identifier string, window time.Duration) string {
	return "rate:" + strings.TrimSpace(scope) + ":" + hashRateLimitIdentifier(identifier) + ":" + strconv.FormatInt(int64(window.Seconds()), 10)
}

func hashRateLimitIdentifier(identifier string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(identifier))))
	return hex.EncodeToString(sum[:])
}

func redisInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, fmt.Errorf("unexpected redis integer type %T", value)
	}
}

func IPIdentifier() RateLimitIdentifier {
	return func(c *gin.Context) string {
		return c.ClientIP()
	}
}

func ConstantIdentifier(value string) RateLimitIdentifier {
	return func(c *gin.Context) string {
		return value
	}
}

func AuthenticatedUserIDIdentifier() RateLimitIdentifier {
	return func(c *gin.Context) string {
		user, ok := auth.CurrentUser(c)
		if !ok {
			return ""
		}
		return strconv.FormatUint(user.UserID, 10)
	}
}

func BodyFieldIdentifier(fields ...string) RateLimitIdentifier {
	return func(c *gin.Context) string {
		values := requestJSONBody(c)
		for _, field := range fields {
			value, ok := values[field]
			if !ok {
				continue
			}
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return text
			}
		}
		return ""
	}
}

func requestJSONBody(c *gin.Context) map[string]any {
	if c.Request == nil || c.Request.Body == nil {
		return nil
	}
	if cached, ok := c.Get("rate_limit_json_body"); ok {
		values, _ := cached.(map[string]any)
		return values
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Request.Body = io.NopCloser(bytes.NewBuffer(nil))
		return nil
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	values := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &values)
	}
	c.Set("rate_limit_json_body", values)
	return values
}

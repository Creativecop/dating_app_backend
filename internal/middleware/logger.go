package middleware

import (
	"log"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		log.Printf(
			"request_id=%s method=%s path=%s status=%d latency=%s ip=%s",
			GetRequestID(c),
			c.Request.Method,
			safeRequestPath(c),
			c.Writer.Status(),
			time.Since(start).String(),
			c.ClientIP(),
		)
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

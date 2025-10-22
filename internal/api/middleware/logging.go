package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

// Logging returns a logging middleware that logs HTTP requests
func Logging() gin.HandlerFunc {
	logger := utils.NewLogger(false)

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get request method
		method := c.Request.Method

		// Build log message
		logMsg := fmt.Sprintf("method=%s path=%s status=%d latency_ms=%d client_ip=%s",
			method, path, statusCode, latency.Milliseconds(), clientIP)

		if query != "" {
			logMsg += fmt.Sprintf(" query=%s", query)
		}

		// Get error if any
		if len(c.Errors) > 0 {
			logMsg += fmt.Sprintf(" errors=%s", c.Errors.String())
		}

		// Log based on status code
		if statusCode >= 500 {
			logger.Error("%s", logMsg)
		} else if statusCode >= 400 {
			logger.Warn("%s", logMsg)
		} else {
			logger.Info("%s", logMsg)
		}
	}
}

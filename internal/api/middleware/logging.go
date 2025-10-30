package middleware

import (
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

		// Build structured log fields
		fields := []utils.Field{
			{Key: "method", Value: method},
			{Key: "path", Value: path},
			{Key: "status", Value: statusCode},
			{Key: "latency", Value: latency},
			{Key: "client_ip", Value: clientIP},
		}

		if query != "" {
			fields = append(fields, utils.Field{Key: "query", Value: query})
		}

		// Get error if any
		var errMsg error
		if len(c.Errors) > 0 {
			errMsg = c.Errors.Last()
			fields = append(fields, utils.Field{Key: "error_count", Value: len(c.Errors)})
		}

		// Log based on status code
		if statusCode >= 500 {
			logger.ErrorWithFields("HTTP request failed", errMsg, fields...)
		} else if statusCode >= 400 {
			logger.WarnWithFields("HTTP request client error", fields...)
		} else {
			logger.InfoWithFields("HTTP request", fields...)
		}
	}
}

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled bool
	Tokens  map[string]bool // Valid API tokens
}

// NewAuthConfig creates a new auth configuration
func NewAuthConfig(enabled bool, tokens []string) *AuthConfig {
	tokenMap := make(map[string]bool)
	for _, token := range tokens {
		if token != "" {
			tokenMap[token] = true
		}
	}
	return &AuthConfig{
		Enabled: enabled,
		Tokens:  tokenMap,
	}
}

// Auth returns an authentication middleware
func Auth(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication if disabled
		if !config.Enabled {
			c.Next()
			return
		}

		// Skip authentication for health check
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		if !config.Tokens[token] {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Token is valid, continue
		c.Next()
	}
}

package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	AllowAll       bool
}

// NewCORSConfig creates a new CORS configuration
func NewCORSConfig(origins []string) *CORSConfig {
	// Default configuration
	config := &CORSConfig{
		AllowedOrigins: origins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowAll:       false,
	}

	// Check if wildcard is present
	for _, origin := range origins {
		if origin == "*" {
			config.AllowAll = true
			break
		}
	}

	return config
}

// CORS returns a CORS middleware
func CORS(config *CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Determine if origin is allowed
		allowed := config.AllowAll
		if !allowed && origin != "" {
			for _, allowedOrigin := range config.AllowedOrigins {
				if allowedOrigin == origin {
					allowed = true
					break
				}
			}
		}

		// Set CORS headers if origin is allowed
		if allowed {
			if config.AllowAll {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}

			c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

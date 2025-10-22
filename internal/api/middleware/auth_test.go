package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuth_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewAuthConfig(false, []string{})
	router := gin.New()
	router.Use(Auth(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuth_HealthCheckBypass(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewAuthConfig(true, []string{"valid-token"})
	router := gin.New()
	router.Use(Auth(config))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewAuthConfig(true, []string{"valid-token"})
	router := gin.New()
	router.Use(Auth(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuth_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewAuthConfig(true, []string{"valid-token"})
	router := gin.New()
	router.Use(Auth(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewAuthConfig(true, []string{"valid-token"})
	router := gin.New()
	router.Use(Auth(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewAuthConfig(true, []string{"valid-token"})
	router := gin.New()
	router.Use(Auth(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestNewAuthConfig(t *testing.T) {
	tokens := []string{"token1", "token2", ""}
	config := NewAuthConfig(true, tokens)

	if !config.Enabled {
		t.Error("Expected auth to be enabled")
	}

	if len(config.Tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(config.Tokens))
	}

	if !config.Tokens["token1"] {
		t.Error("Expected token1 to be in map")
	}

	if !config.Tokens["token2"] {
		t.Error("Expected token2 to be in map")
	}

	if config.Tokens[""] {
		t.Error("Empty token should not be in map")
	}
}

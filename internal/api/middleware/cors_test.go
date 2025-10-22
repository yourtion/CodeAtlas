package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_AllowAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewCORSConfig([]string{"*"})
	router := gin.New()
	router.Use(CORS(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_SpecificOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewCORSConfig([]string{"http://example.com", "http://test.com"})
	router := gin.New()
	router.Use(CORS(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin: http://example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewCORSConfig([]string{"http://example.com"})
	router := gin.New()
	router.Use(CORS(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://malicious.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no Access-Control-Allow-Origin header, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewCORSConfig([]string{"*"})
	router := gin.New()
	router.Use(CORS(config))
	router.OPTIONS("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header to be set")
	}
}

func TestNewCORSConfig(t *testing.T) {
	config := NewCORSConfig([]string{"http://example.com", "http://test.com"})

	if config.AllowAll {
		t.Error("Expected AllowAll to be false")
	}

	if len(config.AllowedOrigins) != 2 {
		t.Errorf("Expected 2 origins, got %d", len(config.AllowedOrigins))
	}

	if len(config.AllowedMethods) == 0 {
		t.Error("Expected default methods to be set")
	}

	if len(config.AllowedHeaders) == 0 {
		t.Error("Expected default headers to be set")
	}
}

func TestNewCORSConfig_Wildcard(t *testing.T) {
	config := NewCORSConfig([]string{"*"})

	if !config.AllowAll {
		t.Error("Expected AllowAll to be true")
	}
}

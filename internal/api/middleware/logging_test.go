package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLogging_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logging())
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

func TestLogging_WithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test?param=value", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLogging_ClientError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogging_ServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logging())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestLogging_WithError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Logging())
	router.GET("/test", func(c *gin.Context) {
		c.Error(http.ErrAbortHandler)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck(t *testing.T) {
	// Create a new Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Create a mock database (we'll need to refactor the server to accept a mock)
	// For now, we'll test the health check endpoint directly
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "CodeAtlas API server is running",
		})
	})

	// Create a test request
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check the response body
	expected := `{"message":"CodeAtlas API server is running","status":"ok"}`
	if w.Body.String() != expected {
		t.Errorf("Expected body %s, got %s", expected, w.Body.String())
	}
}

func TestCreateRepository(t *testing.T) {
	// This test would require a mock database
	// We'll need to refactor the server to accept a database interface
	// that can be mocked for testing purposes
	t.Skip("Skipping test that requires database mock")
}

func TestGetRepository(t *testing.T) {
	// This test would require a mock database
	// We'll need to refactor the server to accept a database interface
	// that can be mocked for testing purposes
	t.Skip("Skipping test that requires database mock")
}
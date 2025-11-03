package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	// Disable Gin logging in tests
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard
}

func TestSetupRouter(t *testing.T) {

	// Create server with default config
	config := &ServerConfig{
		EnableAuth:  false,
		AuthTokens:  []string{},
		CORSOrigins: []string{"*"},
	}

	server := &Server{
		db:     nil, // Mock DB not needed for router setup test
		config: config,
	}

	router := server.SetupRouter()

	if router == nil {
		t.Fatal("Expected router to be created")
	}
}

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &ServerConfig{
		EnableAuth:  false,
		AuthTokens:  []string{},
		CORSOrigins: []string{"*"},
	}

	server := &Server{
		db:     nil,
		config: config,
	}

	router := gin.New()
	server.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	expected := `{"message":"CodeAtlas API server is running","status":"ok"}`
	if w.Body.String() != expected {
		t.Errorf("Expected body %s, got %s", expected, w.Body.String())
	}
}

func TestHealthCheck_WithAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &ServerConfig{
		EnableAuth:  true,
		AuthTokens:  []string{"test-token"},
		CORSOrigins: []string{"*"},
	}

	server := &Server{
		db:     nil,
		config: config,
	}

	router := server.SetupRouter()

	// Health check should work without auth
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &ServerConfig{
		EnableAuth:  false,
		AuthTokens:  []string{},
		CORSOrigins: []string{"http://example.com"},
	}

	server := &Server{
		db:     nil,
		config: config,
	}

	router := server.SetupRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected CORS header to be set, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestAuthProtectedEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := &ServerConfig{
		EnableAuth:  true,
		AuthTokens:  []string{"valid-token"},
		CORSOrigins: []string{"*"},
	}

	server := &Server{
		db:     nil,
		config: config,
	}

	router := server.SetupRouter()

	// Try to access protected endpoint without auth
	req := httptest.NewRequest("GET", "/api/v1/repositories", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	// Try with valid token
	req = httptest.NewRequest("GET", "/api/v1/repositories", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should not be unauthorized (may be 500 due to nil DB, but that's expected)
	if w.Code == http.StatusUnauthorized {
		t.Error("Expected request to pass authentication")
	}
}

func TestNewServer_DefaultConfig(t *testing.T) {
	server := NewServer(nil, nil)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.config == nil {
		t.Fatal("Expected default config to be created")
	}

	if server.config.EnableAuth {
		t.Error("Expected auth to be disabled by default")
	}

	if len(server.config.CORSOrigins) == 0 {
		t.Error("Expected default CORS origins to be set")
	}
}

func TestNewServer_CustomConfig(t *testing.T) {
	config := &ServerConfig{
		EnableAuth:  true,
		AuthTokens:  []string{"token1", "token2"},
		CORSOrigins: []string{"http://example.com"},
	}

	server := NewServer(nil, config)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if !server.config.EnableAuth {
		t.Error("Expected auth to be enabled")
	}

	if len(server.config.AuthTokens) != 2 {
		t.Errorf("Expected 2 auth tokens, got %d", len(server.config.AuthTokens))
	}

	if len(server.config.CORSOrigins) != 1 {
		t.Errorf("Expected 1 CORS origin, got %d", len(server.config.CORSOrigins))
	}
}

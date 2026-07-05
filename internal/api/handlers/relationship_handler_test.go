package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

func TestRelationshipHandler_GetCallers_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		symbolID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty symbol ID",
			symbolID:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Symbol ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with nil DB (won't be used for validation errors)
			handler := NewRelationshipHandler(nil)

			// Create test router
			router := gin.New()
			router.GET("/api/v1/symbols/:id/callers", handler.GetCallers)

			// Create request
			req, _ := http.NewRequest("GET", "/api/v1/symbols/"+tt.symbolID+"/callers", nil)
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRelationshipHandler_GetCallees_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		symbolID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty symbol ID",
			symbolID:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Symbol ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with nil DB
			handler := NewRelationshipHandler(nil)

			// Create test router
			router := gin.New()
			router.GET("/api/v1/symbols/:id/callees", handler.GetCallees)

			// Create request
			req, _ := http.NewRequest("GET", "/api/v1/symbols/"+tt.symbolID+"/callees", nil)
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRelationshipHandler_GetDependencies_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		symbolID       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty symbol ID",
			symbolID:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Symbol ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with nil DB
			handler := NewRelationshipHandler(nil)

			// Create test router
			router := gin.New()
			router.GET("/api/v1/symbols/:id/dependencies", handler.GetDependencies)

			// Create request
			req, _ := http.NewRequest("GET", "/api/v1/symbols/"+tt.symbolID+"/dependencies", nil)
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRelationshipHandler_GetFileSymbols_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		fileID         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty file ID",
			fileID:         "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "File ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler with nil DB
			handler := NewRelationshipHandler(nil)

			// Create test router
			router := gin.New()
			router.GET("/api/v1/files/:id/symbols", handler.GetFileSymbols)

			// Create request
			req, _ := http.NewRequest("GET", "/api/v1/files/"+tt.fileID+"/symbols", nil)
			w := httptest.NewRecorder()

			// Perform request
			router.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestRelationshipResponse_Structure(t *testing.T) {
	response := RelationshipResponse{
		Symbols: []RelatedSymbol{
			{
				SymbolID:  "symbol-1",
				Name:      "testFunc",
				Kind:      "function",
				FilePath:  "test.go",
				Signature: "func testFunc()",
			},
		},
		Total: 1,
	}

	if len(response.Symbols) != response.Total {
		t.Errorf("Expected symbols length %d to match total %d", len(response.Symbols), response.Total)
	}

	symbol := response.Symbols[0]
	if symbol.SymbolID == "" {
		t.Error("Expected symbol_id to be set")
	}
	if symbol.Name == "" {
		t.Error("Expected name to be set")
	}
	if symbol.Kind == "" {
		t.Error("Expected kind to be set")
	}
}

func TestDependencyResponse_Structure(t *testing.T) {
	response := DependencyResponse{
		Dependencies: []Dependency{
			{
				SymbolID:  "symbol-1",
				Name:      "testModule",
				Kind:      "module",
				FilePath:  "test.go",
				EdgeType:  "import",
				Signature: "",
			},
		},
		Total: 1,
	}

	if len(response.Dependencies) != response.Total {
		t.Errorf("Expected dependencies length %d to match total %d", len(response.Dependencies), response.Total)
	}

	dep := response.Dependencies[0]
	if dep.Name == "" {
		t.Error("Expected name to be set")
	}
	if dep.EdgeType == "" {
		t.Error("Expected edge_type to be set")
	}
}

func TestSymbolsResponse_Structure(t *testing.T) {
	response := SymbolsResponse{
		Symbols: []SymbolInfo{
			{
				SymbolID:        "symbol-1",
				Name:            "testFunc",
				Kind:            "function",
				Signature:       "func testFunc()",
				StartLine:       10,
				EndLine:         20,
				Docstring:       "Test function",
				SemanticSummary: "A test function",
			},
		},
		Total: 1,
	}

	if len(response.Symbols) != response.Total {
		t.Errorf("Expected symbols length %d to match total %d", len(response.Symbols), response.Total)
	}

	symbol := response.Symbols[0]
	if symbol.SymbolID == "" {
		t.Error("Expected symbol_id to be set")
	}
	if symbol.Name == "" {
		t.Error("Expected name to be set")
	}
	if symbol.Kind == "" {
		t.Error("Expected kind to be set")
	}
	if symbol.StartLine <= 0 {
		t.Error("Expected start_line to be positive")
	}
	if symbol.EndLine <= symbol.StartLine {
		t.Error("Expected end_line to be greater than start_line")
	}
}

// TestRelationshipHandler_GetTransitiveCallees_InvalidRequest 验证空 symbol ID 返回 400。
func TestRelationshipHandler_GetTransitiveCallees_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewRelationshipHandler(nil)
	router := gin.New()
	router.GET("/api/v1/symbols/:id/transitive-callees", handler.GetTransitiveCallees)

	req, _ := http.NewRequest("GET", "/api/v1/symbols//transitive-callees", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for empty symbol ID, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestRelationshipHandler_GetTransitiveCallers_InvalidRequest 验证空 symbol ID 返回 400。
func TestRelationshipHandler_GetTransitiveCallers_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewRelationshipHandler(nil)
	router := gin.New()
	router.GET("/api/v1/symbols/:id/transitive-callers", handler.GetTransitiveCallers)

	req, _ := http.NewRequest("GET", "/api/v1/symbols//transitive-callers", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for empty symbol ID, got %d", http.StatusBadRequest, w.Code)
	}
}

// TestParseDepthParam 验证 depth 查询参数解析的边界。
func TestParseDepthParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		query   string
		want    int
		name    string
	}{
		{"", models.DefaultTransitiveDepth, "missing uses default"},
		{"?depth=10", 10, "valid depth"},
		{"?depth=0", models.DefaultTransitiveDepth, "zero falls back to default"},
		{"?depth=-3", models.DefaultTransitiveDepth, "negative falls back to default"},
		{"?depth=abc", models.DefaultTransitiveDepth, "invalid falls back to default"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/symbols/x/transitive-callees"+tc.query, nil)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = req
			if got := parseDepthParam(c); got != tc.want {
				t.Errorf("parseDepthParam(%q) = %d, want %d", tc.query, got, tc.want)
			}
		})
	}
}

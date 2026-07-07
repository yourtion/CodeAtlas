package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/qa"
)

var errSentinel = errors.New("boom")

// mockQAService 记录收到的 AskRequest 并返回预设响应。
type mockQAService struct {
	lastReq *qa.AskRequest
	resp    *qa.AskResponse
	err     error
}

func (m *mockQAService) Ask(ctx context.Context, req qa.AskRequest) (*qa.AskResponse, error) {
	m.lastReq = &req
	if m.err != nil {
		return nil, m.err
	}
	if m.resp != nil {
		return m.resp, nil
	}
	// 默认返回一个非空响应
	return &qa.AskResponse{
		Query:    req.Query,
		Prompt:   "# Context\nsample prompt",
		ChunkIDs: []string{"chunk-1"},
		Blocks:   []qa.ContextBlockJSON{{ChunkID: "chunk-1"}},
	}, nil
}

func newTestRouter(h *QAHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/qa", h.Ask)
	r.GET("/api/v1/qa/chunks", h.GetChunks)
	return r
}

func doRequest(t *testing.T, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// 用例 1: 空 query → 400（binding required）
func TestQAHandler_Ask_EmptyQuery_400(t *testing.T) {
	svc := &mockQAService{}
	handler := NewQAHandlerWithService(svc, nil)
	router := newTestRouter(handler)

	w := doRequest(t, router, "POST", "/api/v1/qa", `{}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "Invalid request body" {
		t.Errorf("expected 'Invalid request body', got %v", resp["error"])
	}
	// service 不应被调用
	if svc.lastReq != nil {
		t.Errorf("service should not be called on binding failure")
	}
}

// 用例 2: 正常请求 → 200 + 响应含 prompt/blocks/chunk_ids
func TestQAHandler_Ask_OK_200(t *testing.T) {
	svc := &mockQAService{}
	handler := NewQAHandlerWithService(svc, nil)
	router := newTestRouter(handler)

	w := doRequest(t, router, "POST", "/api/v1/qa", `{"query":"how does auth work"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp qa.AskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if len(resp.Blocks) == 0 {
		t.Error("expected non-empty blocks")
	}
	if len(resp.ChunkIDs) == 0 {
		t.Error("expected non-empty chunk_ids")
	}
	if resp.Query != "how does auth work" {
		t.Errorf("unexpected query echo: %q", resp.Query)
	}
}

// 用例 3: 三态处理 —— 不传字段默认 true；传 false 收到 false
func TestQAHandler_Ask_TristateExpand(t *testing.T) {
	t.Run("omitted defaults to true", func(t *testing.T) {
		svc := &mockQAService{}
		handler := NewQAHandlerWithService(svc, nil)
		router := newTestRouter(handler)

		w := doRequest(t, router, "POST", "/api/v1/qa", `{"query":"q"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
		}
		if svc.lastReq == nil {
			t.Fatal("service not called")
		}
		if !svc.lastReq.ExpandCallers {
			t.Error("expected ExpandCallers default true")
		}
		if !svc.lastReq.ExpandCallees {
			t.Error("expected ExpandCallees default true")
		}
	})

	t.Run("explicit false propagates", func(t *testing.T) {
		svc := &mockQAService{}
		handler := NewQAHandlerWithService(svc, nil)
		router := newTestRouter(handler)

		w := doRequest(t, router, "POST", "/api/v1/qa",
			`{"query":"q","expand_callers":false,"expand_callees":false}`)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
		}
		if svc.lastReq == nil {
			t.Fatal("service not called")
		}
		if svc.lastReq.ExpandCallers {
			t.Error("expected ExpandCallers false")
		}
		if svc.lastReq.ExpandCallees {
			t.Error("expected ExpandCallees false")
		}
	})

	t.Run("explicit true propagates", func(t *testing.T) {
		svc := &mockQAService{}
		handler := NewQAHandlerWithService(svc, nil)
		router := newTestRouter(handler)

		w := doRequest(t, router, "POST", "/api/v1/qa",
			`{"query":"q","expand_callers":true,"expand_callees":true}`)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
		}
		if svc.lastReq == nil {
			t.Fatal("service not called")
		}
		if !svc.lastReq.ExpandCallers || !svc.lastReq.ExpandCallees {
			t.Error("expected both expand flags true")
		}
	})
}

// 用例 4: GetChunks ids 为空 → 400
func TestQAHandler_GetChunks_EmptyIDs_400(t *testing.T) {
	handler := NewQAHandlerWithService(&mockQAService{}, nil)
	router := newTestRouter(handler)

	w := doRequest(t, router, "GET", "/api/v1/qa/chunks", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "ids parameter required" {
		t.Errorf("expected 'ids parameter required', got %v", resp["error"])
	}
}

// 用例 5: GetChunks ids > 50 → 400
func TestQAHandler_GetChunks_TooManyIDs_400(t *testing.T) {
	handler := NewQAHandlerWithService(&mockQAService{}, nil)
	router := newTestRouter(handler)

	// 构造 51 个 id（逗号分隔）
	var ids []byte
	for i := 0; i < 51; i++ {
		if i > 0 {
			ids = append(ids, ',')
		}
		ids = append(ids, 'i', 'd')
	}
	path := "/api/v1/qa/chunks?ids=" + string(ids)
	req, _ := http.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "too many ids, max 50" {
		t.Errorf("expected 'too many ids, max 50', got %v", resp["error"])
	}
}

// 用例: service 返回 error → 500
func TestQAHandler_Ask_ServiceError_500(t *testing.T) {
	svc := &mockQAService{err: errSentinel}
	handler := NewQAHandlerWithService(svc, nil)
	router := newTestRouter(handler)

	w := doRequest(t, router, "POST", "/api/v1/qa", `{"query":"q"}`)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["error"] != "QA failed" {
		t.Errorf("expected 'QA failed', got %v", resp["error"])
	}
}

// 用例: 字段透传（repo_ids/language/kind/mode/limit/include_source）
func TestQAHandler_Ask_FieldPassThrough(t *testing.T) {
	svc := &mockQAService{}
	handler := NewQAHandlerWithService(svc, nil)
	router := newTestRouter(handler)

	body := `{"query":"q","repo_ids":["r1"],"language":"go","kind":["function"],"mode":"hybrid","limit":5,"include_source":true}`
	w := doRequest(t, router, "POST", "/api/v1/qa", body)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	if svc.lastReq == nil {
		t.Fatal("service not called")
	}
	r := svc.lastReq
	if len(r.RepoIDs) != 1 || r.RepoIDs[0] != "r1" {
		t.Errorf("repo_ids mismatch: %v", r.RepoIDs)
	}
	if r.Language != "go" {
		t.Errorf("language mismatch: %v", r.Language)
	}
	if len(r.Kind) != 1 || r.Kind[0] != "function" {
		t.Errorf("kind mismatch: %v", r.Kind)
	}
	if r.Mode != "hybrid" {
		t.Errorf("mode mismatch: %v", r.Mode)
	}
	if r.Limit != 5 {
		t.Errorf("limit mismatch: %v", r.Limit)
	}
	if !r.IncludeSource {
		t.Error("expected include_source true")
	}
}

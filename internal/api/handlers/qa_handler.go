package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/qa"
	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// QAHandler handles QA endpoints.
type QAHandler struct {
	qaService  qa.Service
	vectorRepo *models.VectorRepository // 给 chunks 端点用
}

// NewQAHandler creates a QA handler. 组装真实依赖（与 NewSearchHandler 的组装方式一致）。
// embedderConfig 用 *EmbedderConfig（handlers 包 alias），与 NewSearchHandler 保持一致。
func NewQAHandler(db *models.DB, embedderConfig *EmbedderConfig) *QAHandler {
	vectorRepo := models.NewVectorRepository(db)
	edgeRepo := models.NewEdgeRepository(db)
	// 与 NewSearchHandler 一致：nil config 退回默认配置
	if embedderConfig == nil {
		embedderConfig = indexer.DefaultEmbedderConfig()
	}
	emb := indexer.NewOpenAIEmbedder(embedderConfig, vectorRepo)
	retriever := retrieval.NewHybridRetriever(vectorRepo, edgeRepo, emb, retrieval.DefaultHybridRetrieverConfig())
	sf := &vectorSourceFetcher{vr: vectorRepo}
	return &QAHandler{
		qaService:  qa.NewService(retriever, sf, qa.DefaultPromptBuildOptions()),
		vectorRepo: vectorRepo,
	}
}

// NewQAHandlerWithService creates a QA handler with an injected qa.Service
// and VectorRepository, 方便测试注入 mock（无需真实 DB）。
func NewQAHandlerWithService(svc qa.Service, vr *models.VectorRepository) *QAHandler {
	return &QAHandler{
		qaService:  svc,
		vectorRepo: vr,
	}
}

// vectorSourceFetcher 把 VectorRepository 适配成 qa.SourceFetcher。
type vectorSourceFetcher struct {
	vr *models.VectorRepository
}

func (f *vectorSourceFetcher) GetByVectorIDs(ctx context.Context, ids []string) (map[string]string, error) {
	vectors, err := f.vr.GetByVectorIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(vectors))
	for _, v := range vectors {
		m[v.VectorID] = v.Content
	}
	return m, nil
}

// askRequestBody 是 POST /api/v1/qa 的请求体。
type askRequestBody struct {
	Query         string   `json:"query" binding:"required"`
	RepoIDs       []string `json:"repo_ids,omitempty"`
	Language      string   `json:"language,omitempty"`
	Kind          []string `json:"kind,omitempty"`
	Mode          string   `json:"mode,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	IncludeSource bool     `json:"include_source,omitempty"`
	ExpandCallers *bool    `json:"expand_callers,omitempty"` // 指针区分"未传"(nil→默认true)和"传false"
	ExpandCallees *bool    `json:"expand_callees,omitempty"`
}

// Ask handles POST /api/v1/qa
func (h *QAHandler) Ask(c *gin.Context) {
	var body askRequestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// 三态处理：ExpandCallers/Callees 未传时默认 true
	expandCallers := true
	if body.ExpandCallers != nil {
		expandCallers = *body.ExpandCallers
	}
	expandCallees := true
	if body.ExpandCallees != nil {
		expandCallees = *body.ExpandCallees
	}

	req := qa.AskRequest{
		Query:         body.Query,
		RepoIDs:       body.RepoIDs,
		Language:      body.Language,
		Kind:          body.Kind,
		Mode:          body.Mode,
		Limit:         body.Limit,
		IncludeSource: body.IncludeSource,
		ExpandCallers: expandCallers,
		ExpandCallees: expandCallees,
	}

	resp, err := h.qaService.Ask(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "QA failed", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetChunks handles GET /api/v1/qa/chunks?ids=id1,id2
func (h *QAHandler) GetChunks(c *gin.Context) {
	idsParam := c.Query("ids")
	if idsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids parameter required"})
		return
	}
	ids := strings.Split(idsParam, ",")
	if len(ids) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many ids, max 50"})
		return
	}

	vectors, err := h.vectorRepo.GetByVectorIDs(c.Request.Context(), ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch chunks", "details": err.Error()})
		return
	}

	chunks := make([]chunkJSON, 0, len(vectors))
	for _, v := range vectors {
		chunks = append(chunks, chunkJSON{
			ChunkID:  v.VectorID,
			SymbolID: v.EntityID,
			Content:  v.Content,
		})
	}
	c.JSON(http.StatusOK, gin.H{"chunks": chunks})
}

type chunkJSON struct {
	ChunkID  string `json:"chunk_id"`
	SymbolID string `json:"symbol_id"`
	Content  string `json:"content"`
}

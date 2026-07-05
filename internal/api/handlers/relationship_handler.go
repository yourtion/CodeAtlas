package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// RelationshipHandler handles relationship query operations
type RelationshipHandler struct {
	db         *models.DB
	symbolRepo *models.SymbolRepository
	fileRepo   *models.FileRepository
	edgeRepo   *models.EdgeRepository
}

// NewRelationshipHandler creates a new relationship handler
func NewRelationshipHandler(db *models.DB) *RelationshipHandler {
	return &RelationshipHandler{
		db:         db,
		symbolRepo: models.NewSymbolRepository(db),
		fileRepo:   models.NewFileRepository(db),
		edgeRepo:   models.NewEdgeRepository(db),
	}
}

// RelatedSymbol represents a symbol in a relationship query result
type RelatedSymbol struct {
	SymbolID  string `json:"symbol_id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"file_path"`
	Signature string `json:"signature"`
}

// RelationshipResponse represents the response for relationship queries
type RelationshipResponse struct {
	Symbols []RelatedSymbol `json:"symbols"`
	Total   int             `json:"total"`
}

// DependencyResponse represents the response for dependency queries
type DependencyResponse struct {
	Dependencies []Dependency `json:"dependencies"`
	Total        int          `json:"total"`
}

// Dependency represents a dependency relationship
type Dependency struct {
	SymbolID     string `json:"symbol_id,omitempty"`
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	FilePath     string `json:"file_path,omitempty"`
	Module       string `json:"module,omitempty"`
	EdgeType     string `json:"edge_type"`
	Signature    string `json:"signature,omitempty"`
}

// SymbolsResponse represents the response for file symbols query
type SymbolsResponse struct {
	Symbols []SymbolInfo `json:"symbols"`
	Total   int          `json:"total"`
}

// SymbolInfo represents symbol information
type SymbolInfo struct {
	SymbolID        string `json:"symbol_id"`
	Name            string `json:"name"`
	Kind            string `json:"kind"`
	Signature       string `json:"signature"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	Docstring       string `json:"docstring,omitempty"`
	SemanticSummary string `json:"semantic_summary,omitempty"`
}

// GetCallers handles GET /api/v1/symbols/:id/callers
// Finds all functions that call the specified symbol via parameterized SQL.
func (h *RelationshipHandler) GetCallers(c *gin.Context) {
	symbolID := c.Param("id")
	if symbolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Symbol ID is required",
		})
		return
	}

	ctx := context.Background()

	// Verify symbol exists
	symbol, err := h.symbolRepo.GetByID(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve symbol",
			"details": err.Error(),
		})
		return
	}
	if symbol == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Symbol not found",
		})
		return
	}

	// 查询关系表（edges JOIN symbols/files），一次 SQL 取回调用方详情。
	h.getCallersSQL(c, symbolID)
}

// getCallersSQL 通过 JOIN 查询返回调用给定符号的所有符号（含详情），
// 一次 SQL 消除原先逐条 GetByID 的 N+1 查询。
func (h *RelationshipHandler) getCallersSQL(c *gin.Context, symbolID string) {
	ctx := context.Background()

	edges, err := h.edgeRepo.GetCallersWithDetails(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve callers",
			"details": err.Error(),
		})
		return
	}

	results := make([]RelatedSymbol, 0, len(edges))
	for _, e := range edges {
		results = append(results, RelatedSymbol{
			SymbolID:  e.SymbolID,
			Name:      e.Name,
			Kind:      e.Kind,
			FilePath:  e.FilePath,
			Signature: e.Signature,
		})
	}

	c.JSON(http.StatusOK, RelationshipResponse{Symbols: results, Total: len(results)})
}

// GetCallees handles GET /api/v1/symbols/:id/callees
// Finds all functions called by the specified symbol using Cypher queries
func (h *RelationshipHandler) GetCallees(c *gin.Context) {
	symbolID := c.Param("id")
	if symbolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Symbol ID is required",
		})
		return
	}

	ctx := context.Background()

	// Verify symbol exists
	symbol, err := h.symbolRepo.GetByID(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve symbol",
			"details": err.Error(),
		})
		return
	}
	if symbol == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Symbol not found",
		})
		return
	}

	// 查询关系表，一次 SQL 取回被调用方详情。
	h.getCalleesSQL(c, symbolID)
}

// getCalleesSQL 通过 JOIN 查询返回给定符号调用的所有符号（含详情），
// 一次 SQL 消除原先逐条 GetByID 的 N+1 查询。
func (h *RelationshipHandler) getCalleesSQL(c *gin.Context, symbolID string) {
	ctx := context.Background()

	edges, err := h.edgeRepo.GetCalleesWithDetails(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve callees",
			"details": err.Error(),
		})
		return
	}

	results := make([]RelatedSymbol, 0, len(edges))
	for _, e := range edges {
		results = append(results, RelatedSymbol{
			SymbolID:  e.SymbolID,
			Name:      e.Name,
			Kind:      e.Kind,
			FilePath:  e.FilePath,
			Signature: e.Signature,
		})
	}

	c.JSON(http.StatusOK, RelationshipResponse{Symbols: results, Total: len(results)})
}

// GetDependencies handles GET /api/v1/symbols/:id/dependencies
// Finds all dependencies of the specified symbol (imports, extends, implements)
func (h *RelationshipHandler) GetDependencies(c *gin.Context) {
	symbolID := c.Param("id")
	if symbolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Symbol ID is required",
		})
		return
	}

	ctx := context.Background()

	// Verify symbol exists
	symbol, err := h.symbolRepo.GetByID(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve symbol",
			"details": err.Error(),
		})
		return
	}
	if symbol == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Symbol not found",
		})
		return
	}

	// 查询关系表，一次 SQL 取回依赖详情（含外部模块依赖）。
	h.getDependenciesSQL(c, symbolID)
}

// getDependenciesSQL 通过 JOIN 查询返回给定符号的依赖（含详情），
// 一次 SQL 消除原先逐条 GetByID 的 N+1 查询。
// 分两类：内部符号依赖（JOIN symbols/files）+ 外部模块依赖（仅 target_module）。
func (h *RelationshipHandler) getDependenciesSQL(c *gin.Context, symbolID string) {
	ctx := context.Background()

	// 1. 内部符号依赖（有 target_id，JOIN 取详情）
	internalDeps, err := h.edgeRepo.GetDependenciesWithDetails(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve dependencies",
			"details": err.Error(),
		})
		return
	}

	// 2. 外部模块依赖（无 target_id，仅有 target_module，如未解析的 import）
	externalDeps, err := h.edgeRepo.GetExternalDependencies(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve external dependencies",
			"details": err.Error(),
		})
		return
	}

	results := make([]Dependency, 0, len(internalDeps)+len(externalDeps))
	for _, e := range internalDeps {
		results = append(results, Dependency{
			SymbolID:  e.SymbolID,
			Name:      e.Name,
			Kind:      e.Kind,
			FilePath:  e.FilePath,
			Signature: e.Signature,
			EdgeType:  e.EdgeType,
		})
	}
	for _, e := range externalDeps {
		module := ""
		if e.TargetModule != nil {
			module = *e.TargetModule
		}
		results = append(results, Dependency{
			Module:   module,
			Name:     module,
			Kind:     "module",
			EdgeType: e.EdgeType,
		})
	}

	c.JSON(http.StatusOK, DependencyResponse{Dependencies: results, Total: len(results)})
}

// GetFileSymbols handles GET /api/v1/files/:id/symbols
// Retrieves all symbols in a file using SQL queries
func (h *RelationshipHandler) GetFileSymbols(c *gin.Context) {
	fileID := c.Param("id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File ID is required",
		})
		return
	}

	ctx := context.Background()

	// Verify file exists
	file, err := h.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve file",
			"details": err.Error(),
		})
		return
	}
	if file == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "File not found",
		})
		return
	}

	// Get all symbols for the file
	symbols, err := h.symbolRepo.GetByFileID(ctx, fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve symbols",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	results := make([]SymbolInfo, len(symbols))
	for i, symbol := range symbols {
		results[i] = SymbolInfo{
			SymbolID:        symbol.SymbolID,
			Name:            symbol.Name,
			Kind:            symbol.Kind,
			Signature:       symbol.Signature,
			StartLine:       symbol.StartLine,
			EndLine:         symbol.EndLine,
			Docstring:       symbol.Docstring,
			SemanticSummary: symbol.SemanticSummary,
		}
	}

	response := SymbolsResponse{
		Symbols: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
}

// ReachableSymbolResponse 是多跳可达性查询的单条结果。
type ReachableSymbolResponse struct {
	SymbolID  string `json:"symbol_id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FilePath  string `json:"file_path"`
	Signature string `json:"signature"`
	// Depth 是该符号相对起始符号的最短跳数（1=直接调用关系，2=隔一跳，依此类推）。
	Depth int `json:"depth"`
}

// TransitiveResponse 是多跳查询响应。
type TransitiveResponse struct {
	Symbols []ReachableSymbolResponse `json:"symbols"`
	Total   int                       `json:"total"`
	Depth   int                       `json:"depth"` // 实际使用的最大深度
}

// parseDepthParam 解析可选的 depth 查询参数，<=0 或非法时返回默认值。
func parseDepthParam(c *gin.Context) int {
	raw := c.Query("depth")
	if raw == "" {
		return models.DefaultTransitiveDepth
	}
	d, err := strconv.Atoi(raw)
	if err != nil || d <= 0 {
		return models.DefaultTransitiveDepth
	}
	return d
}

// verifySymbol 校验符号存在，返回 true 并写入 404/500 响应时返回 false。
func (h *RelationshipHandler) verifySymbol(c *gin.Context) (string, bool) {
	symbolID := c.Param("id")
	if symbolID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol ID is required"})
		return "", false
	}
	ctx := context.Background()
	symbol, err := h.symbolRepo.GetByID(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve symbol", "details": err.Error()})
		return "", false
	}
	if symbol == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Symbol not found"})
		return "", false
	}
	return symbolID, true
}

// toReachableResponse 将 []*ReachableSymbol 转为响应并写入 c。
func toReachableResponse(c *gin.Context, reachable []*models.ReachableSymbol, depth int) {
	results := make([]ReachableSymbolResponse, 0, len(reachable))
	for _, r := range reachable {
		results = append(results, ReachableSymbolResponse{
			SymbolID:  r.SymbolID,
			Name:      r.Name,
			Kind:      r.Kind,
			FilePath:  r.FilePath,
			Signature: r.Signature,
			Depth:     r.Depth,
		})
	}
	c.JSON(http.StatusOK, TransitiveResponse{
		Symbols: results,
		Total:   len(results),
		Depth:   depth,
	})
}

// GetTransitiveCallees handles GET /api/v1/symbols/:id/transitive-callees
// 返回从指定符号出发沿调用边递归可达的全部符号（传递调用链）。
// 可选查询参数 depth 控制最大跳数（默认 5）。
//
// 语义："起始符号的执行会触及哪些代码"——例如查 main 的传递调用链可得到
// 整棵调用子树（去重，每符号取最短跳数）。
func (h *RelationshipHandler) GetTransitiveCallees(c *gin.Context) {
	symbolID, ok := h.verifySymbol(c)
	if !ok {
		return
	}
	depth := parseDepthParam(c)
	reachable, err := h.edgeRepo.GetTransitiveCallees(c.Request.Context(), symbolID, depth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transitive callees", "details": err.Error()})
		return
	}
	toReachableResponse(c, reachable, depth)
}

// GetTransitiveCallers handles GET /api/v1/symbols/:id/transitive-callers
// 返回沿调用边反向递归可达的全部符号（反向影响范围）。
// 可选查询参数 depth 控制最大跳数（默认 5）。
//
// 语义："修改起始符号会影响哪些代码"——例如查某底层函数的传递调用方
// 可得到所有直接/间接依赖它的入口点。
func (h *RelationshipHandler) GetTransitiveCallers(c *gin.Context) {
	symbolID, ok := h.verifySymbol(c)
	if !ok {
		return
	}
	depth := parseDepthParam(c)
	reachable, err := h.edgeRepo.GetTransitiveCallers(c.Request.Context(), symbolID, depth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transitive callers", "details": err.Error()})
		return
	}
	toReachableResponse(c, reachable, depth)
}

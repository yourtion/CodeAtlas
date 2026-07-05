package handlers

import (
	"context"
	"net/http"

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

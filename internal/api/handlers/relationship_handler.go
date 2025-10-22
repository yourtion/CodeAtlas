package handlers

import (
	"context"
	"database/sql"
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
// Finds all functions that call the specified symbol using Cypher queries
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

	// Query using AGE Cypher to find callers
	// This finds all symbols that have a CALLS edge pointing to this symbol
	query := `
		SELECT * FROM cypher('code_graph', $$
			MATCH (caller)-[r:CALLS]->(callee)
			WHERE callee.symbol_id = $symbol_id
			RETURN caller.symbol_id as symbol_id, 
			       caller.name as name,
			       caller.kind as kind,
			       caller.file_path as file_path,
			       caller.signature as signature
		$$) AS (symbol_id agtype, name agtype, kind agtype, file_path agtype, signature agtype)
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		// If AGE is not available or graph doesn't exist, fall back to SQL
		h.getCallersSQL(c, symbolID)
		return
	}
	defer rows.Close()

	results := make([]RelatedSymbol, 0)
	for rows.Next() {
		var result RelatedSymbol
		var symbolIDAgtype, nameAgtype, kindAgtype, filePathAgtype, signatureAgtype sql.NullString
		
		err := rows.Scan(&symbolIDAgtype, &nameAgtype, &kindAgtype, &filePathAgtype, &signatureAgtype)
		if err != nil {
			continue
		}

		// Parse agtype values (they come as JSON strings)
		result.SymbolID = parseAgtypeString(symbolIDAgtype.String)
		result.Name = parseAgtypeString(nameAgtype.String)
		result.Kind = parseAgtypeString(kindAgtype.String)
		result.FilePath = parseAgtypeString(filePathAgtype.String)
		result.Signature = parseAgtypeString(signatureAgtype.String)

		results = append(results, result)
	}

	response := RelationshipResponse{
		Symbols: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
}

// getCallersSQL is a fallback method using SQL when AGE is not available
func (h *RelationshipHandler) getCallersSQL(c *gin.Context, symbolID string) {
	ctx := context.Background()

	// Get edges where this symbol is the target
	edges, err := h.edgeRepo.GetByTargetAndType(ctx, symbolID, "call")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve callers",
			"details": err.Error(),
		})
		return
	}

	results := make([]RelatedSymbol, 0)
	for _, edge := range edges {
		// Get source symbol details
		sourceSymbol, err := h.symbolRepo.GetByID(ctx, edge.SourceID)
		if err != nil || sourceSymbol == nil {
			continue
		}

		// Get file path
		file, err := h.fileRepo.GetByID(ctx, sourceSymbol.FileID)
		if err != nil || file == nil {
			continue
		}

		result := RelatedSymbol{
			SymbolID:  sourceSymbol.SymbolID,
			Name:      sourceSymbol.Name,
			Kind:      sourceSymbol.Kind,
			FilePath:  file.Path,
			Signature: sourceSymbol.Signature,
		}
		results = append(results, result)
	}

	response := RelationshipResponse{
		Symbols: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
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

	// Query using AGE Cypher to find callees
	query := `
		SELECT * FROM cypher('code_graph', $$
			MATCH (caller)-[r:CALLS]->(callee)
			WHERE caller.symbol_id = $symbol_id
			RETURN callee.symbol_id as symbol_id,
			       callee.name as name,
			       callee.kind as kind,
			       callee.file_path as file_path,
			       callee.signature as signature
		$$) AS (symbol_id agtype, name agtype, kind agtype, file_path agtype, signature agtype)
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		// Fall back to SQL
		h.getCalleesSQL(c, symbolID)
		return
	}
	defer rows.Close()

	results := make([]RelatedSymbol, 0)
	for rows.Next() {
		var result RelatedSymbol
		var symbolIDAgtype, nameAgtype, kindAgtype, filePathAgtype, signatureAgtype sql.NullString
		
		err := rows.Scan(&symbolIDAgtype, &nameAgtype, &kindAgtype, &filePathAgtype, &signatureAgtype)
		if err != nil {
			continue
		}

		result.SymbolID = parseAgtypeString(symbolIDAgtype.String)
		result.Name = parseAgtypeString(nameAgtype.String)
		result.Kind = parseAgtypeString(kindAgtype.String)
		result.FilePath = parseAgtypeString(filePathAgtype.String)
		result.Signature = parseAgtypeString(signatureAgtype.String)

		results = append(results, result)
	}

	response := RelationshipResponse{
		Symbols: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
}

// getCalleesSQL is a fallback method using SQL when AGE is not available
func (h *RelationshipHandler) getCalleesSQL(c *gin.Context, symbolID string) {
	ctx := context.Background()

	// Get edges where this symbol is the source
	edges, err := h.edgeRepo.GetBySourceAndType(ctx, symbolID, "call")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve callees",
			"details": err.Error(),
		})
		return
	}

	results := make([]RelatedSymbol, 0)
	for _, edge := range edges {
		if edge.TargetID == nil {
			continue
		}

		// Get target symbol details
		targetSymbol, err := h.symbolRepo.GetByID(ctx, *edge.TargetID)
		if err != nil || targetSymbol == nil {
			continue
		}

		// Get file path
		file, err := h.fileRepo.GetByID(ctx, targetSymbol.FileID)
		if err != nil || file == nil {
			continue
		}

		result := RelatedSymbol{
			SymbolID:  targetSymbol.SymbolID,
			Name:      targetSymbol.Name,
			Kind:      targetSymbol.Kind,
			FilePath:  file.Path,
			Signature: targetSymbol.Signature,
		}
		results = append(results, result)
	}

	response := RelationshipResponse{
		Symbols: results,
		Total:   len(results),
	}

	c.JSON(http.StatusOK, response)
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

	// Query using AGE Cypher to find dependencies
	query := `
		SELECT * FROM cypher('code_graph', $$
			MATCH (source)-[r]->(target)
			WHERE source.symbol_id = $symbol_id
			  AND type(r) IN ['IMPORTS', 'EXTENDS', 'IMPLEMENTS', 'REFERENCES']
			RETURN target.symbol_id as symbol_id,
			       target.name as name,
			       target.kind as kind,
			       target.file_path as file_path,
			       target.signature as signature,
			       type(r) as edge_type
		$$) AS (symbol_id agtype, name agtype, kind agtype, file_path agtype, signature agtype, edge_type agtype)
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		// Fall back to SQL
		h.getDependenciesSQL(c, symbolID)
		return
	}
	defer rows.Close()

	results := make([]Dependency, 0)
	for rows.Next() {
		var dep Dependency
		var symbolIDAgtype, nameAgtype, kindAgtype, filePathAgtype, signatureAgtype, edgeTypeAgtype sql.NullString
		
		err := rows.Scan(&symbolIDAgtype, &nameAgtype, &kindAgtype, &filePathAgtype, &signatureAgtype, &edgeTypeAgtype)
		if err != nil {
			continue
		}

		dep.SymbolID = parseAgtypeString(symbolIDAgtype.String)
		dep.Name = parseAgtypeString(nameAgtype.String)
		dep.Kind = parseAgtypeString(kindAgtype.String)
		dep.FilePath = parseAgtypeString(filePathAgtype.String)
		dep.Signature = parseAgtypeString(signatureAgtype.String)
		dep.EdgeType = parseAgtypeString(edgeTypeAgtype.String)

		results = append(results, dep)
	}

	response := DependencyResponse{
		Dependencies: results,
		Total:        len(results),
	}

	c.JSON(http.StatusOK, response)
}

// getDependenciesSQL is a fallback method using SQL when AGE is not available
func (h *RelationshipHandler) getDependenciesSQL(c *gin.Context, symbolID string) {
	ctx := context.Background()

	// Get all outgoing edges for dependency types
	edges, err := h.edgeRepo.GetBySourceID(ctx, symbolID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to retrieve dependencies",
			"details": err.Error(),
		})
		return
	}

	results := make([]Dependency, 0)
	for _, edge := range edges {
		// Filter for dependency edge types
		if edge.EdgeType != "import" && edge.EdgeType != "extends" && 
		   edge.EdgeType != "implements" && edge.EdgeType != "reference" {
			continue
		}

		dep := Dependency{
			EdgeType: edge.EdgeType,
		}

		// Handle edges with target symbols
		if edge.TargetID != nil {
			targetSymbol, err := h.symbolRepo.GetByID(ctx, *edge.TargetID)
			if err == nil && targetSymbol != nil {
				dep.SymbolID = targetSymbol.SymbolID
				dep.Name = targetSymbol.Name
				dep.Kind = targetSymbol.Kind
				dep.Signature = targetSymbol.Signature

				// Get file path
				file, err := h.fileRepo.GetByID(ctx, targetSymbol.FileID)
				if err == nil && file != nil {
					dep.FilePath = file.Path
				}
			}
		}

		// Handle edges with target modules (imports without resolved symbols)
		if edge.TargetModule != nil && dep.SymbolID == "" {
			dep.Module = *edge.TargetModule
			dep.Name = *edge.TargetModule
			dep.Kind = "module"
		}

		results = append(results, dep)
	}

	response := DependencyResponse{
		Dependencies: results,
		Total:        len(results),
	}

	c.JSON(http.StatusOK, response)
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

// parseAgtypeString parses an agtype JSON string value
// AGE returns values as JSON, so "value" becomes value
func parseAgtypeString(agtype string) string {
	if len(agtype) < 2 {
		return agtype
	}
	// Remove surrounding quotes if present
	if agtype[0] == '"' && agtype[len(agtype)-1] == '"' {
		return agtype[1 : len(agtype)-1]
	}
	return agtype
}

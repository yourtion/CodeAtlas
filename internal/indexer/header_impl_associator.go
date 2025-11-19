package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// HeaderImplAssociator handles header-implementation file pairing and symbol matching
type HeaderImplAssociator struct {
	db     *models.DB
	logger IndexerLogger
}

// NewHeaderImplAssociator creates a new header-implementation associator
func NewHeaderImplAssociator(db *models.DB, logger IndexerLogger) *HeaderImplAssociator {
	if logger == nil {
		logger = &noOpLogger{}
	}
	return &HeaderImplAssociator{
		db:     db,
		logger: logger,
	}
}

// HeaderImplPair represents a matched header-implementation file pair
type HeaderImplPair struct {
	HeaderFile string
	ImplFile   string
	Language   string
}

// AssociationResult contains the results of header-implementation association
type AssociationResult struct {
	PairsFound     int                    `json:"pairs_found"`
	EdgesCreated   int                    `json:"edges_created"`
	Duration       string                 `json:"duration"`
	Errors         []AssociationError     `json:"errors,omitempty"`
}

// AssociationError represents an error during association
type AssociationError struct {
	File    string `json:"file"`
	Message string `json:"message"`
}

// AssociateHeadersAndImplementations performs header-implementation association
// This should be called after all files have been parsed and written to the database
func (a *HeaderImplAssociator) AssociateHeadersAndImplementations(ctx context.Context, files []schema.File) (*AssociationResult, error) {
	result := &AssociationResult{}
	
	a.logger.Info("starting header-implementation association")
	
	// Step 1: Identify header-implementation pairs
	pairs := a.findHeaderImplPairs(files)
	result.PairsFound = len(pairs)
	
	a.logger.InfoWithFields("found header-implementation pairs",
		LogField{Key: "pair_count", Value: len(pairs)},
	)
	
	// Step 2: For each pair, match symbols and create edges
	var allEdges []schema.DependencyEdge
	for _, pair := range pairs {
		edges, err := a.matchSymbolsAndCreateEdges(ctx, pair, files)
		if err != nil {
			result.Errors = append(result.Errors, AssociationError{
				File:    pair.ImplFile,
				Message: err.Error(),
			})
			a.logger.WarnWithFields("failed to match symbols for pair",
				LogField{Key: "header", Value: pair.HeaderFile},
				LogField{Key: "impl", Value: pair.ImplFile},
				LogField{Key: "error", Value: err.Error()},
			)
			continue
		}
		allEdges = append(allEdges, edges...)
	}
	
	// Step 3: Write edges to database
	if len(allEdges) > 0 {
		edgeRepo := models.NewEdgeRepository(a.db)
		modelEdges := a.convertToModelEdges(allEdges)
		
		err := edgeRepo.BatchCreate(ctx, modelEdges)
		if err != nil {
			return result, fmt.Errorf("failed to write header-impl edges: %w", err)
		}
		result.EdgesCreated = len(modelEdges)
		
		a.logger.InfoWithFields("created header-implementation edges",
			LogField{Key: "edge_count", Value: len(modelEdges)},
		)
	}
	
	a.logger.Info("header-implementation association completed")
	return result, nil
}

// findHeaderImplPairs identifies header-implementation file pairs
func (a *HeaderImplAssociator) findHeaderImplPairs(files []schema.File) []HeaderImplPair {
	var pairs []HeaderImplPair
	
	// Create maps for quick lookup
	fileMap := make(map[string]schema.File)
	for _, file := range files {
		fileMap[file.Path] = file
	}
	
	// Process each file
	for _, file := range files {
		// Skip if not a language with header/implementation split
		if !a.isHeaderImplLanguage(file.Language) {
			continue
		}
		
		// Check if this is a header file
		if a.isHeaderFile(file.Path) {
			// Look for corresponding implementation file
			implPath := a.findImplementationFile(file.Path, fileMap)
			if implPath != "" {
				pairs = append(pairs, HeaderImplPair{
					HeaderFile: file.Path,
					ImplFile:   implPath,
					Language:   file.Language,
				})
			}
		}
	}
	
	return pairs
}

// isHeaderImplLanguage checks if a language uses header/implementation split
func (a *HeaderImplAssociator) isHeaderImplLanguage(language string) bool {
	switch strings.ToLower(language) {
	case "c", "cpp", "c++", "objc", "objective-c", "objcpp", "objective-c++":
		return true
	default:
		return false
	}
}

// isHeaderFile checks if a file is a header file
func (a *HeaderImplAssociator) isHeaderFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".h" || ext == ".hpp" || ext == ".hh" || ext == ".hxx"
}

// findImplementationFile finds the corresponding implementation file for a header
func (a *HeaderImplAssociator) findImplementationFile(headerPath string, fileMap map[string]schema.File) string {
	dir := filepath.Dir(headerPath)
	baseName := strings.TrimSuffix(filepath.Base(headerPath), filepath.Ext(headerPath))
	
	// Try different implementation file extensions
	extensions := []string{".c", ".cpp", ".cc", ".cxx", ".m", ".mm"}
	for _, ext := range extensions {
		implPath := filepath.Join(dir, baseName+ext)
		if _, exists := fileMap[implPath]; exists {
			return implPath
		}
	}
	
	return ""
}

// matchSymbolsAndCreateEdges matches symbols between header and implementation files
func (a *HeaderImplAssociator) matchSymbolsAndCreateEdges(ctx context.Context, pair HeaderImplPair, files []schema.File) ([]schema.DependencyEdge, error) {
	var edges []schema.DependencyEdge
	
	// Find the header and implementation files
	var headerFile, implFile *schema.File
	for i := range files {
		if files[i].Path == pair.HeaderFile {
			headerFile = &files[i]
		}
		if files[i].Path == pair.ImplFile {
			implFile = &files[i]
		}
	}
	
	if headerFile == nil || implFile == nil {
		return nil, fmt.Errorf("could not find header or implementation file")
	}
	
	// Create file-level implements_header edge using virtual file symbols
	// We create virtual symbols to represent the files themselves, similar to external modules
	headerFileSymbol := a.createVirtualFileSymbol(headerFile)
	implFileSymbol := a.createVirtualFileSymbol(implFile)
	
	// Write virtual symbols to database first
	if err := a.writeVirtualSymbols(ctx, []schema.Symbol{headerFileSymbol, implFileSymbol}); err != nil {
		a.logger.WarnWithFields("failed to write virtual file symbols",
			LogField{Key: "error", Value: err},
		)
		// Non-fatal, continue with symbol-level edges
	} else {
		// Create file-level edge using virtual symbol IDs
		fileEdge := schema.DependencyEdge{
			EdgeID:     generateEdgeID(),
			SourceID:   implFileSymbol.SymbolID,
			TargetID:   headerFileSymbol.SymbolID,
			EdgeType:   schema.EdgeImplementsHeader,
			SourceFile: implFile.Path,
			TargetFile: headerFile.Path,
		}
		edges = append(edges, fileEdge)
	}
	
	// Match symbols between header and implementation
	symbolEdges := a.matchSymbols(headerFile, implFile)
	edges = append(edges, symbolEdges...)
	
	return edges, nil
}

// matchSymbols matches symbols between header and implementation files
func (a *HeaderImplAssociator) matchSymbols(headerFile, implFile *schema.File) []schema.DependencyEdge {
	var edges []schema.DependencyEdge
	
	// Create a map of header symbols for quick lookup
	headerSymbols := make(map[string]schema.Symbol)
	for _, symbol := range headerFile.Symbols {
		// Use normalized signature as key for matching
		key := a.normalizeSymbolKey(symbol)
		headerSymbols[key] = symbol
	}
	
	// Match implementation symbols with header symbols
	for _, implSymbol := range implFile.Symbols {
		key := a.normalizeSymbolKey(implSymbol)
		if headerSymbol, found := headerSymbols[key]; found {
			// Check if signatures match
			if a.signaturesMatch(headerSymbol, implSymbol) {
				// Create implements_declaration edge
				edge := schema.DependencyEdge{
					EdgeID:     generateEdgeID(),
					SourceID:   implSymbol.SymbolID,
					TargetID:   headerSymbol.SymbolID,
					EdgeType:   schema.EdgeImplementsDeclaration,
					SourceFile: implFile.Path,
					TargetFile: headerFile.Path,
				}
				edges = append(edges, edge)
			}
		}
	}
	
	return edges
}

// normalizeSymbolKey creates a normalized key for symbol matching
func (a *HeaderImplAssociator) normalizeSymbolKey(symbol schema.Symbol) string {
	// For functions and methods, use name as the primary key
	// In a more sophisticated implementation, we would include parameter types
	return strings.ToLower(symbol.Name)
}

// signaturesMatch checks if two symbols have matching signatures
func (a *HeaderImplAssociator) signaturesMatch(headerSymbol, implSymbol schema.Symbol) bool {
	// Basic matching: check if names match and kinds are compatible
	if headerSymbol.Name != implSymbol.Name {
		return false
	}
	
	// Check if kinds are compatible (e.g., function_declaration matches function)
	if !a.kindsCompatible(headerSymbol.Kind, implSymbol.Kind) {
		return false
	}
	
	// Normalize and compare signatures
	headerSig := a.normalizeSignature(headerSymbol.Signature)
	implSig := a.normalizeSignature(implSymbol.Signature)
	
	return headerSig == implSig
}

// kindsCompatible checks if two symbol kinds are compatible for matching
func (a *HeaderImplAssociator) kindsCompatible(headerKind, implKind schema.SymbolKind) bool {
	headerKindStr := string(headerKind)
	implKindStr := string(implKind)
	
	// Function declarations match function definitions
	if headerKindStr == "function_declaration" && (implKindStr == "function" || implKindStr == "static_function") {
		return true
	}
	
	// Method declarations match method definitions
	if headerKindStr == "method_declaration" && implKindStr == "method" {
		return true
	}
	
	// Same kinds always match
	if headerKind == implKind {
		return true
	}
	
	return false
}

// normalizeSignature normalizes a function signature for comparison
func (a *HeaderImplAssociator) normalizeSignature(signature string) string {
	// Normalize whitespace: collapse multiple spaces into one
	sig := strings.Join(strings.Fields(signature), " ")
	
	// Trim leading/trailing whitespace
	sig = strings.TrimSpace(sig)
	
	// Remove spaces around parentheses and commas for consistent comparison
	sig = strings.ReplaceAll(sig, " (", "(")
	sig = strings.ReplaceAll(sig, "( ", "(")
	sig = strings.ReplaceAll(sig, " )", ")")
	sig = strings.ReplaceAll(sig, " ,", ",")
	sig = strings.ReplaceAll(sig, ", ", ",")
	
	return sig
}

// createVirtualFileSymbol creates a virtual symbol to represent a file
// This allows us to create file-level edges while satisfying foreign key constraints
func (a *HeaderImplAssociator) createVirtualFileSymbol(file *schema.File) schema.Symbol {
	// Generate deterministic ID for file symbols
	symbolID := generateDeterministicUUID("file:" + file.Path)
	
	return schema.Symbol{
		SymbolID:  symbolID,
		FileID:    file.FileID,
		Name:      fmt.Sprintf("__file__:%s", filepath.Base(file.Path)),
		Kind:      schema.SymbolModule, // Use module kind for file-level symbols
		Signature: fmt.Sprintf("file: %s", file.Path),
		Span: schema.Span{
			StartLine: 1,
			EndLine:   1,
			StartByte: 0,
			EndByte:   0,
		},
		Docstring: fmt.Sprintf("Virtual symbol representing file %s", file.Path),
	}
}

// writeVirtualSymbols writes virtual symbols to the database
func (a *HeaderImplAssociator) writeVirtualSymbols(ctx context.Context, symbols []schema.Symbol) error {
	if len(symbols) == 0 {
		return nil
	}
	
	// Skip if no database connection (for testing)
	if a.db == nil {
		return nil
	}
	
	symbolRepo := models.NewSymbolRepository(a.db)
	
	for _, symbol := range symbols {
		// Check if symbol already exists
		existing, err := symbolRepo.GetByID(ctx, symbol.SymbolID)
		if err == nil && existing != nil {
			// Symbol already exists, skip
			continue
		}
		
		// Create new symbol
		modelSymbol := &models.Symbol{
			SymbolID:  symbol.SymbolID,
			FileID:    symbol.FileID,
			Name:      symbol.Name,
			Kind:      string(symbol.Kind),
			Signature: symbol.Signature,
			StartLine: symbol.Span.StartLine,
			EndLine:   symbol.Span.EndLine,
			StartByte: symbol.Span.StartByte,
			EndByte:   symbol.Span.EndByte,
			Docstring: symbol.Docstring,
		}
		
		if err := symbolRepo.Create(ctx, modelSymbol); err != nil {
			return fmt.Errorf("failed to create virtual symbol %s: %w", symbol.Name, err)
		}
	}
	
	return nil
}

// generateDeterministicUUID generates a deterministic UUID from a string
func generateDeterministicUUID(input string) string {
	// Use the same approach as external modules
	return utils.GenerateDeterministicUUID(input)
}

// convertToModelEdges converts schema edges to model edges
func (a *HeaderImplAssociator) convertToModelEdges(edges []schema.DependencyEdge) []*models.Edge {
	modelEdges := make([]*models.Edge, 0, len(edges))
	
	for _, edge := range edges {
		var targetID *string
		if edge.TargetID != "" {
			targetID = &edge.TargetID
		}
		
		var targetFile *string
		if edge.TargetFile != "" {
			targetFile = &edge.TargetFile
		}
		
		modelEdge := &models.Edge{
			EdgeID:     edge.EdgeID,
			SourceID:   edge.SourceID,
			TargetID:   targetID,
			EdgeType:   string(edge.EdgeType),
			SourceFile: edge.SourceFile,
			TargetFile: targetFile,
		}
		modelEdges = append(modelEdges, modelEdge)
	}
	
	return modelEdges
}

// generateEdgeID generates a unique edge ID
func generateEdgeID() string {
	return uuid.New().String()
}

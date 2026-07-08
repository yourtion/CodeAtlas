package schema

import (
	"fmt"
	"path/filepath"
	"sort"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
	"github.com/yourtionguo/CodeAtlas/internal/utils"
)

// SchemaMapper transforms parsed files into the unified schema format
type SchemaMapper struct {
	// 旧字段（向后兼容 MapToSchema 单文件场景）
	symbolIDMap map[string]string
	// Map to track external symbols (module name -> Symbol)
	externalSymbols map[string]*Symbol

	// 新字段：两遍扫描
	// 候选集：符号名 → 该名字的所有候选（累积，不覆盖）
	symbolCandidates map[string][]symbolCandidate
	// import 关系：fileID → 该文件 import 过的模块/文件路径集合
	fileImports map[string]map[string]bool
	// 累积的待解析依赖（第一遍收集，第二遍解析）
	pendingDeps []pendingDependency
	// 日志函数（消歧告警用），nil 则不记日志
	warnLog func(format string, args ...interface{})
}

type symbolCandidate struct {
	SymbolID string
	FileID   string
	FilePath string
}

type pendingDependency struct {
	Dep            parser.ParsedDependency
	SourceFileID   string
	SourceFilePath string
}

// NewSchemaMapper creates a new schema mapper
func NewSchemaMapper() *SchemaMapper {
	return &SchemaMapper{
		symbolIDMap:      make(map[string]string),
		externalSymbols:  make(map[string]*Symbol),
		symbolCandidates: make(map[string][]symbolCandidate),
		fileImports:      make(map[string]map[string]bool),
	}
}

// SetWarnLog sets a logger function for disambiguation warnings
func (m *SchemaMapper) SetWarnLog(fn func(format string, args ...interface{})) {
	m.warnLog = fn
}

// CollectSymbols is the first pass: collects symbols to candidate set,
// import relations, pending deps (including imports), external symbols,
// and AST nodes — but does NOT resolve edges.
func (m *SchemaMapper) CollectSymbols(parsed *parser.ParsedFile) (*File, error) {
	checksum := utils.SHA256Checksum(parsed.Content)
	fileID := utils.GenerateDeterministicUUID(fmt.Sprintf("file:%s:%s", parsed.Path, checksum))

	file := &File{
		FileID:   fileID,
		Path:     parsed.Path,
		Language: parsed.Language,
		Size:     int64(len(parsed.Content)),
		Checksum: checksum,
		Nodes:    []ASTNode{},
		Symbols:  []Symbol{},
	}

	// 收集符号到候选集（累积，不覆盖）
	for _, parsedSymbol := range parsed.Symbols {
		symbol := m.mapSymbol(parsedSymbol, fileID)
		file.Symbols = append(file.Symbols, symbol)
		m.symbolIDMap[parsedSymbol.Name] = symbol.SymbolID // 向后兼容
		m.symbolCandidates[parsedSymbol.Name] = append(
			m.symbolCandidates[parsedSymbol.Name],
			symbolCandidate{SymbolID: symbol.SymbolID, FileID: fileID, FilePath: parsed.Path},
		)
	}

	// 收集外部模块符号（保持现有逻辑）
	externalModules := m.collectExternalModules(parsed.Dependencies)
	for moduleName := range externalModules {
		externalSymbol := m.createExternalSymbol(moduleName)
		m.externalSymbols[moduleName] = &externalSymbol
		m.symbolIDMap[moduleName] = externalSymbol.SymbolID
	}

	// 收集 import 关系
	m.collectFileImports(parsed.Dependencies, fileID)

	// 收集所有 dep 到 pendingDeps（含 import，第二遍统一解析）
	for _, dep := range parsed.Dependencies {
		m.pendingDeps = append(m.pendingDeps, pendingDependency{
			Dep: dep, SourceFileID: fileID, SourceFilePath: parsed.Path,
		})
	}

	// 映射 AST 节点（保持现有逻辑）
	if parsed.RootNode != nil {
		file.Nodes = m.mapASTNodes(parsed.RootNode, fileID, "", parsed.Content)
	}

	return file, nil
}

// collectFileImports records the import relations for a file
func (m *SchemaMapper) collectFileImports(deps []parser.ParsedDependency, fileID string) {
	for _, dep := range deps {
		if dep.Type == "import" && dep.TargetModule != "" {
			if m.fileImports[fileID] == nil {
				m.fileImports[fileID] = make(map[string]bool)
			}
			m.fileImports[fileID][dep.TargetModule] = true
		}
	}
}

// ResolveEdges is the second pass: resolves all accumulated pending deps
// using the candidate set collected during CollectSymbols.
//
// 边 ID 是确定性的（基于 source_id/edge_type/target_id/source_file 等字段），
// 同一源文件里多次出现同一 (source, type, target) 依赖（如多次调用 console）
// 会生成相同 edge_id。这里按 edge_id 去重，保留首次出现的边——语义上它们是
// 同一条关系，与索引器的 upsert 行为一致，也避免 validator 的 duplicate_id 报错。
func (m *SchemaMapper) ResolveEdges() ([]DependencyEdge, error) {
	var edges []DependencyEdge
	seen := make(map[string]bool, len(m.pendingDeps))
	for _, pd := range m.pendingDeps {
		edge := m.resolveEdge(pd)
		if edge == nil {
			continue
		}
		if seen[edge.EdgeID] {
			continue
		}
		seen[edge.EdgeID] = true
		edges = append(edges, *edge)
	}
	return edges, nil
}

// resolveEdge resolves a single pending dependency to an edge
func (m *SchemaMapper) resolveEdge(pd pendingDependency) *DependencyEdge {
	dep := pd.Dep
	sourceID := m.resolveCandidateID(dep.Source, pd.SourceFileID, pd.SourceFilePath)

	if dep.Type == "import" {
		edgeType := m.mapEdgeType(dep.Type)
		edgeID := utils.GenerateDeterministicUUID(fmt.Sprintf("edge:%s:%s:%s:%s", pd.SourceFilePath, dep.Type, dep.Target, dep.TargetModule))
		targetID := m.resolveImportTarget(dep)
		return &DependencyEdge{
			EdgeID:       edgeID,
			SourceID:     sourceID,
			TargetID:     targetID,
			EdgeType:     edgeType,
			SourceFile:   pd.SourceFilePath,
			TargetModule: dep.TargetModule,
		}
	}

	// 非 import 边：source 必须存在
	if sourceID == "" {
		return nil
	}

	targetID, targetFile := m.resolveCandidateWithFile(dep.Target, pd.SourceFileID, pd.SourceFilePath)
	edgeType := m.mapEdgeType(dep.Type)
	edgeID := utils.GenerateDeterministicUUID(fmt.Sprintf("edge:%s:%s:%s:%s", sourceID, dep.Type, targetID, pd.SourceFilePath))

	return &DependencyEdge{
		EdgeID:     edgeID,
		SourceID:   sourceID,
		TargetID:   targetID,
		EdgeType:   edgeType,
		SourceFile: pd.SourceFilePath,
		TargetFile: targetFile,
	}
}

// resolveCandidateWithFile 解析符号到 (symbolID, filePath)。
// 与 resolveCandidateID 逻辑一致，但额外返回候选的 FilePath（用于填充 TargetFile）。
func (m *SchemaMapper) resolveCandidateWithFile(name, sourceFileID, sourceFilePath string) (string, string) {
	if name == "" {
		return "", ""
	}
	candidates := m.symbolCandidates[name]
	if len(candidates) == 0 {
		return m.symbolIDMap[name], ""
	}
	if len(candidates) == 1 {
		return candidates[0].SymbolID, candidates[0].FilePath
	}
	// 多候选消歧——disambiguate 返回 ID，我们再反查 FilePath
	resolvedID := m.disambiguate(name, candidates, sourceFileID, sourceFilePath)
	for _, c := range candidates {
		if c.SymbolID == resolvedID {
			return resolvedID, c.FilePath
		}
	}
	return resolvedID, ""
}

// resolveCandidateID resolves a symbol name to its ID using the candidate set,
// falling back to symbolIDMap (which contains external symbols).
func (m *SchemaMapper) resolveCandidateID(name, sourceFileID, sourceFilePath string) string {
	if name == "" {
		return ""
	}
	candidates := m.symbolCandidates[name]
	if len(candidates) == 0 {
		return m.symbolIDMap[name] // 降级查旧 map（含外部符号）
	}
	if len(candidates) == 1 {
		return candidates[0].SymbolID
	}
	return m.disambiguate(name, candidates, sourceFileID, sourceFilePath)
}

// disambiguate picks a single candidate when multiple exist:
// 1) same-file candidate, 2) import-path match, 3) first candidate with warning.
func (m *SchemaMapper) disambiguate(name string, candidates []symbolCandidate, sourceFileID, sourceFilePath string) string {
	// 1. 同文件优先
	var sameFile []symbolCandidate
	for _, c := range candidates {
		if c.FilePath == sourceFilePath {
			sameFile = append(sameFile, c)
		}
	}
	if len(sameFile) == 1 {
		return sameFile[0].SymbolID
	}
	if len(sameFile) > 1 {
		sort.Slice(sameFile, func(i, j int) bool { return sameFile[i].SymbolID < sameFile[j].SymbolID })
		m.logDisambiguation(name, sourceFilePath, len(sameFile), "same_file_multiple")
		return sameFile[0].SymbolID
	}

	// 2. import 文件优先（精确匹配文件名，避免 strings.Contains 误匹配）
	imports := m.fileImports[sourceFileID]
	if imports != nil {
		var importMatch []symbolCandidate
		for _, c := range candidates {
			for mod := range imports {
				modBase := filepath.Base(mod)
				if filepath.Base(c.FilePath) == modBase || c.FilePath == mod {
					importMatch = append(importMatch, c)
					break
				}
			}
		}
		if len(importMatch) == 1 {
			return importMatch[0].SymbolID
		}
		if len(importMatch) > 1 {
			sort.Slice(importMatch, func(i, j int) bool { return importMatch[i].SymbolID < importMatch[j].SymbolID })
			m.logDisambiguation(name, sourceFilePath, len(importMatch), "import_multiple")
			return importMatch[0].SymbolID
		}
	}

	// 3. 首个候选 + 日志
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].SymbolID < candidates[j].SymbolID })
	m.logDisambiguation(name, sourceFilePath, len(candidates), "first_candidate")
	return candidates[0].SymbolID
}

// logDisambiguation logs a disambiguation warning if a warnLog is set
func (m *SchemaMapper) logDisambiguation(name, sourceFile string, count int, strategy string) {
	if m.warnLog != nil {
		m.warnLog("符号消歧: %q 在 %s 有 %d 候选，策略=%s", name, sourceFile, count, strategy)
	}
}

// resolveImportTarget resolves an import edge's target to a symbol ID
// by matching the target module against collected candidate file paths.
func (m *SchemaMapper) resolveImportTarget(dep parser.ParsedDependency) string {
	if dep.TargetModule == "" {
		return ""
	}
	target := dep.TargetModule
	// 取 import 路径的最后一段作为文件名（如 "c_library.h" from "include/c_library.h"）
	targetBase := filepath.Base(target)
	for _, candidates := range m.symbolCandidates {
		for _, c := range candidates {
			// 精确匹配文件名，或文件路径精确等于 target
			if c.FilePath == target || filepath.Base(c.FilePath) == targetBase {
				return c.SymbolID
			}
		}
	}
	return ""
}

// MapToSchema transforms a ParsedFile into a schema.File (single-file backward-compat).
// Internally calls CollectSymbols, then uses the legacy mapDependencies (via symbolIDMap)
// to keep existing single-file behavior intact. For cross-file resolution, use the
// two-pass CollectSymbols + ResolveEdges flow instead.
func (m *SchemaMapper) MapToSchema(parsed *parser.ParsedFile) (*File, []DependencyEdge, error) {
	// MapToSchema 是单文件向后兼容入口，保持隔离语义
	// 跨文件场景用 CollectSymbols + ResolveEdges
	m.symbolIDMap = make(map[string]string)
	m.symbolCandidates = make(map[string][]symbolCandidate)
	m.fileImports = make(map[string]map[string]bool)
	m.pendingDeps = nil

	file, err := m.CollectSymbols(parsed)
	if err != nil {
		return nil, nil, err
	}
	// 单文件场景：用旧 mapDependencies（symbolIDMap 已在 CollectSymbols 填充）
	edges := m.mapDependencies(parsed.Dependencies, file.FileID, parsed.Path)
	return file, edges, nil
}

// mapSymbol transforms a ParsedSymbol into a schema.Symbol
func (m *SchemaMapper) mapSymbol(parsed parser.ParsedSymbol, fileID string) Symbol {
	// Generate deterministic UUID based on file_id, name, start_line, and start_byte
	// This ensures the same symbol always gets the same ID across multiple parses
	symbolKey := fmt.Sprintf("%s:%s:%d:%d", fileID, parsed.Name, parsed.Span.StartLine, parsed.Span.StartByte)
	symbolID := utils.GenerateDeterministicUUID(symbolKey)

	// Map symbol kind
	kind := m.mapSymbolKind(parsed.Kind)

	// Map span
	span := Span{
		StartLine: parsed.Span.StartLine,
		EndLine:   parsed.Span.EndLine,
		StartByte: parsed.Span.StartByte,
		EndByte:   parsed.Span.EndByte,
	}

	symbol := Symbol{
		SymbolID:  symbolID,
		FileID:    fileID,
		Name:      parsed.Name,
		Kind:      kind,
		Signature: parsed.Signature,
		Span:      span,
		Docstring: parsed.Docstring,
	}

	return symbol
}

// mapSymbolKind maps parser symbol kinds to schema symbol kinds
func (m *SchemaMapper) mapSymbolKind(kind string) SymbolKind {
	switch kind {
	case "function", "method":
		return SymbolFunction
	case "class", "struct":
		return SymbolClass
	case "interface":
		return SymbolInterface
	case "variable", "field", "type":
		return SymbolVariable
	case "package":
		return SymbolPackage
	case "module":
		return SymbolModule
	default:
		return SymbolVariable
	}
}

// mapASTNodes recursively transforms Tree-sitter nodes into schema.ASTNode
func (m *SchemaMapper) mapASTNodes(node *sitter.Node, fileID string, parentID string, content []byte) []ASTNode {
	if node == nil {
		return nil
	}

	var nodes []ASTNode

	// Create node for current Tree-sitter node
	nodeID := utils.GenerateUUID()

	span := Span{
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		StartByte: int(node.StartByte()),
		EndByte:   int(node.EndByte()),
	}

	// Extract text for small nodes (< 100 bytes)
	text := ""
	nodeSize := int(node.EndByte() - node.StartByte())
	if nodeSize < 100 {
		text = node.Content(content)
	}

	astNode := ASTNode{
		NodeID:     nodeID,
		FileID:     fileID,
		Type:       node.Type(),
		Span:       span,
		ParentID:   parentID,
		Text:       text,
		Attributes: make(map[string]string),
	}

	// Add node type as attribute
	if node.IsNamed() {
		astNode.Attributes["named"] = "true"
	}

	nodes = append(nodes, astNode)

	// Recursively process children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		childNodes := m.mapASTNodes(child, fileID, nodeID, content)
		nodes = append(nodes, childNodes...)
	}

	return nodes
}

// mapDependencies transforms ParsedDependency into schema.DependencyEdge
func (m *SchemaMapper) mapDependencies(dependencies []parser.ParsedDependency, fileID string, filePath string) []DependencyEdge {
	var edges []DependencyEdge

	for _, dep := range dependencies {
		edge := m.mapDependency(dep, fileID, filePath)
		if edge != nil {
			edges = append(edges, *edge)
		}
	}

	return edges
}

// mapDependency transforms a single ParsedDependency into a schema.DependencyEdge
func (m *SchemaMapper) mapDependency(dep parser.ParsedDependency, fileID string, filePath string) *DependencyEdge {
	edgeID := utils.GenerateUUID()

	// Map edge type
	edgeType := m.mapEdgeType(dep.Type)

	// Resolve source and target symbol IDs
	sourceID := m.resolveSymbolID(dep.Source)
	targetID := m.resolveSymbolID(dep.Target)

	// For imports, we may not have a target symbol ID (external module)
	// In that case, we still create the edge with target module information
	if dep.Type == "import" {
		edge := DependencyEdge{
			EdgeID:       edgeID,
			SourceID:     sourceID,
			TargetID:     targetID,
			EdgeType:     edgeType,
			SourceFile:   filePath,
			TargetModule: dep.TargetModule,
		}
		return &edge
	}

	// For other edge types, we need both source and target IDs
	if sourceID == "" || targetID == "" {
		// Cannot create edge without both IDs
		return nil
	}

	edge := DependencyEdge{
		EdgeID:     edgeID,
		SourceID:   sourceID,
		TargetID:   targetID,
		EdgeType:   edgeType,
		SourceFile: filePath,
	}

	return &edge
}

// mapEdgeType maps parser dependency types to schema edge types
func (m *SchemaMapper) mapEdgeType(depType string) EdgeType {
	switch depType {
	case "import":
		return EdgeImport
	case "call":
		return EdgeCall
	case "extends":
		return EdgeExtends
	case "implements":
		return EdgeImplements
	case "implements_declaration":
		return EdgeImplementsDeclaration
	case "calls_declaration":
		return EdgeCallsDeclaration
	default:
		return EdgeReference
	}
}

// resolveSymbolID looks up the symbol ID for a given symbol name
func (m *SchemaMapper) resolveSymbolID(symbolName string) string {
	if symbolName == "" {
		return ""
	}
	return m.symbolIDMap[symbolName]
}

// collectExternalModules collects all external module names from dependencies
func (m *SchemaMapper) collectExternalModules(deps []parser.ParsedDependency) map[string]bool {
	modules := make(map[string]bool)
	for _, dep := range deps {
		if dep.IsExternal && dep.Type == "import" && dep.TargetModule != "" {
			modules[dep.TargetModule] = true
		}
	}
	return modules
}

// createExternalSymbol creates a virtual symbol for an external module
func (m *SchemaMapper) createExternalSymbol(moduleName string) Symbol {
	// Generate deterministic ID for external modules
	symbolID := utils.GenerateDeterministicUUID("external:" + moduleName)

	return Symbol{
		SymbolID:  symbolID,
		FileID:    ExternalFileID,
		Name:      moduleName,
		Kind:      SymbolExternal,
		Signature: fmt.Sprintf("external module: %s", moduleName),
		Span: Span{
			StartLine: 1, // Validators require start_line >= 1
			EndLine:   1,
			StartByte: 0,
			EndByte:   0,
		},
	}
}

// GetExternalSymbols returns all external symbols collected during mapping
func (m *SchemaMapper) GetExternalSymbols() []Symbol {
	symbols := make([]Symbol, 0, len(m.externalSymbols))
	for _, symbol := range m.externalSymbols {
		symbols = append(symbols, *symbol)
	}
	return symbols
}

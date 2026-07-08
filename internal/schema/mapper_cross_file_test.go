package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/parser"
)

// makeParsedFile 构造测试用的 ParsedFile。
func makeParsedFile(path, language string, symbols []parser.ParsedSymbol, deps []parser.ParsedDependency) *parser.ParsedFile {
	return &parser.ParsedFile{
		Path:         path,
		Language:     language,
		Content:      []byte("test content for " + path),
		Symbols:      symbols,
		Dependencies: deps,
	}
}

// spanOf 是构造 ParsedSpan 的便捷函数，避免每个用例重复写字段名。
func spanOf(startLine, startByte int) parser.ParsedSpan {
	return parser.ParsedSpan{
		StartLine: startLine,
		EndLine:   startLine,
		StartByte: startByte,
		EndByte:   startByte + 1,
	}
}

// findEdge 按 EdgeType 在边列表中查找首条匹配的边。
func findEdge(edges []DependencyEdge, et EdgeType) (DependencyEdge, bool) {
	for _, e := range edges {
		if e.EdgeType == et {
			return e, true
		}
	}
	return DependencyEdge{}, false
}

// TestResolveEdges_CrossFileCall 验证文件 A 的 caller 调用文件 B 的 callee 时，
// target_id 正确消解到 B 的符号。
func TestResolveEdges_CrossFileCall(t *testing.T) {
	mapper := NewSchemaMapper()

	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: spanOf(1, 0)},
		},
		[]parser.ParsedDependency{
			{Type: "call", Source: "caller", Target: "callee"},
		},
	)
	fileB := makeParsedFile("b.go", "go",
		[]parser.ParsedSymbol{
			{Name: "callee", Kind: "function", Span: spanOf(1, 0)},
		},
		nil,
	)

	fileAOut, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	fileBOut, err := mapper.CollectSymbols(fileB)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	callEdge, ok := findEdge(edges, EdgeCall)
	require.True(t, ok, "应存在 call 边")
	assert.Equal(t, fileAOut.Symbols[0].SymbolID, callEdge.SourceID, "SourceID 应为 A 的 caller")
	assert.Equal(t, fileBOut.Symbols[0].SymbolID, callEdge.TargetID, "TargetID 应消解到 B 的 callee")
	assert.NotEmpty(t, callEdge.TargetID, "跨文件调用 target_id 不应为空")
}

// TestResolveEdges_SameNameDisambiguation_SameFile 验证同文件同名符号优先于
// 另一文件的同名符号：caller 与 helper 同在 a.go，b.go 也有 helper，
// 消歧应选中 a.go 的 helper。
func TestResolveEdges_SameNameDisambiguation_SameFile(t *testing.T) {
	mapper := NewSchemaMapper()

	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: spanOf(1, 0)},
			{Name: "helper", Kind: "function", Span: spanOf(2, 0)},
		},
		[]parser.ParsedDependency{
			{Type: "call", Source: "caller", Target: "helper"},
		},
	)
	fileB := makeParsedFile("b.go", "go",
		[]parser.ParsedSymbol{
			{Name: "helper", Kind: "function", Span: spanOf(1, 0)},
		},
		nil,
	)

	fileAOut, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	_, err = mapper.CollectSymbols(fileB)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	callEdge, ok := findEdge(edges, EdgeCall)
	require.True(t, ok, "应存在 call 边")

	aHelper := fileAOut.Symbols[1] // a.go 的 helper
	assert.Equal(t, aHelper.SymbolID, callEdge.TargetID, "同文件 helper 应优先")
	assert.NotEqual(t, "", callEdge.TargetID, "target_id 不应为空")
}

// TestResolveEdges_SameNameDisambiguation_ImportFile 验证当源文件无同名符号时，
// import 路径匹配的文件优先：a.go import b.go，b.go 与 c.go 都有 helper，
// 消歧应选中 b.go 的 helper。
func TestResolveEdges_SameNameDisambiguation_ImportFile(t *testing.T) {
	mapper := NewSchemaMapper()

	// a.go 通过 import 依赖 b.go（TargetModule 取 base 后与 b.go 文件名匹配）
	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: spanOf(1, 0)},
		},
		[]parser.ParsedDependency{
			{Type: "import", Source: "caller", Target: "b", TargetModule: "pkg/b.go"},
			{Type: "call", Source: "caller", Target: "helper"},
		},
	)
	fileB := makeParsedFile("b.go", "go",
		[]parser.ParsedSymbol{
			{Name: "helper", Kind: "function", Span: spanOf(1, 0)},
		},
		nil,
	)
	fileC := makeParsedFile("c.go", "go",
		[]parser.ParsedSymbol{
			{Name: "helper", Kind: "function", Span: spanOf(1, 0)},
		},
		nil,
	)

	_, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	fileBOut, err := mapper.CollectSymbols(fileB)
	require.NoError(t, err)
	_, err = mapper.CollectSymbols(fileC)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	callEdge, ok := findEdge(edges, EdgeCall)
	require.True(t, ok, "应存在 call 边")
	assert.Equal(t, fileBOut.Symbols[0].SymbolID, callEdge.TargetID,
		"import 匹配的 b.go helper 应优先于 c.go")
	assert.NotEmpty(t, callEdge.TargetID, "target_id 不应为空")
}

// TestResolveEdges_DanglingEdge 验证调用不存在的符号时保留悬空边：
// target_id 为空但边仍然生成。
func TestResolveEdges_DanglingEdge(t *testing.T) {
	mapper := NewSchemaMapper()

	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: spanOf(1, 0)},
		},
		[]parser.ParsedDependency{
			{Type: "call", Source: "caller", Target: "nonexistent"},
		},
	)

	fileAOut, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	callEdge, ok := findEdge(edges, EdgeCall)
	require.True(t, ok, "悬空 call 边应保留")
	assert.Equal(t, fileAOut.Symbols[0].SymbolID, callEdge.SourceID, "SourceID 应为 caller")
	assert.Empty(t, callEdge.TargetID, "不存在的 target 应为空（悬空边）")
}

// TestResolveEdges_ImportEdge 验证 import 边保留 TargetModule 字段。
func TestResolveEdges_ImportEdge(t *testing.T) {
	mapper := NewSchemaMapper()

	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: spanOf(1, 0)},
		},
		[]parser.ParsedDependency{
			{Type: "import", Source: "caller", Target: "fmt", TargetModule: "fmt", IsExternal: true},
		},
	)

	_, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	importEdge, ok := findEdge(edges, EdgeImport)
	require.True(t, ok, "应存在 import 边")
	assert.Equal(t, EdgeImport, importEdge.EdgeType)
	assert.Equal(t, "fmt", importEdge.TargetModule, "import 边应保留 TargetModule")
}

// TestResolveEdges_MapEdgeType_ImplementsDeclaration 验证 implements_declaration
// 类型被正确映射为 EdgeImplementsDeclaration。
func TestResolveEdges_MapEdgeType_ImplementsDeclaration(t *testing.T) {
	mapper := NewSchemaMapper()

	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "impl", Kind: "function", Span: spanOf(1, 0)},
			{Name: "decl", Kind: "function", Span: spanOf(2, 0)},
		},
		[]parser.ParsedDependency{
			{Type: "implements_declaration", Source: "impl", Target: "decl"},
		},
	)

	fileAOut, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	edge, ok := findEdge(edges, EdgeImplementsDeclaration)
	require.True(t, ok, "应存在 implements_declaration 边")
	assert.Equal(t, EdgeImplementsDeclaration, edge.EdgeType)
	assert.Equal(t, fileAOut.Symbols[0].SymbolID, edge.SourceID, "SourceID 应为 impl")
	assert.Equal(t, fileAOut.Symbols[1].SymbolID, edge.TargetID, "TargetID 应为 decl")
	assert.NotEmpty(t, edge.TargetID, "target_id 不应为空")
}

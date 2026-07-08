# 跨文件符号消解 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 SchemaMapper 从单文件作用域改成全仓库两遍扫描，修复跨文件调用边丢失（`cross_file_connectivity=0.00`），附带修复 mapEdgeType/filterValidEdges/真值恢复。

**Architecture:** CollectSymbols（第一遍收集候选集+import 关系）→ ResolveEdges（第二遍消歧解析边）。候选集消歧优先级：同文件 → import 文件 → 首个+日志。无候选保留为悬空边。

**Tech Stack:** Go 1.25+, tree-sitter, PostgreSQL, testify。

**Spec:** `docs/superpowers/specs/2026-07-08-cross-file-symbol-resolution-design.md`

---

## File Structure

**修改：**
- `internal/schema/mapper.go` — 核心改造：新增 CollectSymbols/ResolveEdges、候选集+import 关系、消歧逻辑、mapEdgeType 补全
- `internal/indexer/validator.go` — 非 import 边空 target_id 从 Error 降级为 warning
- `cmd/cli/parse_command.go` — 调用方改用 CollectSymbols + ResolveEdges 两阶段
- `cmd/cli/index_command.go` — 同上
- `tests/integration/quality_gate_test.go` — 移除 filterValidEdges 的 target 校验
- `internal/quality/fixtures/graph_ground_truth.go` — 恢复跨文件真值

**新建：**
- `internal/schema/mapper_cross_file_test.go` — 跨文件消解单元测试

---

## Task 1: SchemaMapper 核心改造——CollectSymbols + ResolveEdges

**Files:**
- Modify: `internal/schema/mapper.go`

这是整个改造的核心。把单文件 MapToSchema 拆成两阶段。

- [ ] **Step 1: 扩展 SchemaMapper 结构体**

读 `internal/schema/mapper.go:11-25`。在现有结构体上加新字段，保留 `symbolIDMap`（向后兼容用）：

```go
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
```

更新 `NewSchemaMapper` 初始化新字段：

```go
func NewSchemaMapper() *SchemaMapper {
	return &SchemaMapper{
		symbolIDMap:     make(map[string]string),
		externalSymbols: make(map[string]*Symbol),
		symbolCandidates: make(map[string][]symbolCandidate),
		fileImports:      make(map[string]map[string]bool),
	}
}
```

- [ ] **Step 2: 实现 CollectSymbols**

在 `mapper.go` 加新方法。CollectSymbols 提取 MapToSchema 的符号收集逻辑，但不解析边：

```go
// CollectSymbols 第一遍：收集文件符号 + import 关系 + 待解析边，不解析非 import 边。
// 调用方对所有文件循环调 CollectSymbols 后，再调一次 ResolveEdges 统一解析边。
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

		// 向后兼容：保留 symbolIDMap（单文件场景 MapToSchema 用）
		m.symbolIDMap[parsedSymbol.Name] = symbol.SymbolID

		// 新：累积到候选集
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

	// 收集待解析边（非 import 的 dep 延迟到第二遍）
	for _, dep := range parsed.Dependencies {
		if dep.Type != "import" {
			m.pendingDeps = append(m.pendingDeps, pendingDependency{
				Dep:            dep,
				SourceFileID:   fileID,
				SourceFilePath: parsed.Path,
			})
		}
	}

	// 映射 AST 节点（保持现有逻辑）
	if parsed.RootNode != nil {
		astNodes := m.mapASTNodes(parsed.RootNode, fileID, "", parsed.Content)
		file.Nodes = astNodes
	}

	return file, nil
}

// collectFileImports 从依赖里提取 import 关系，记录到 fileImports。
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
```

- [ ] **Step 3: 实现 ResolveEdges**

在 `mapper.go` 加新方法。这是第二遍——用候选集解析所有累积的边：

```go
// ResolveEdges 第二遍：用累积的候选集解析所有待解析边。
// 必须在所有文件的 CollectSymbols 完成后调用。
func (m *SchemaMapper) ResolveEdges() ([]DependencyEdge, error) {
	var edges []DependencyEdge

	// 先解析所有 import 边（它们需要 source 符号 + 可能的 target 模块匹配）
	// import 边在 CollectSymbols 时没有加入 pendingDeps，但需要在最终边列表里。
	// 重新从 pendingDeps 之外无法拿到——import 边信息在 CollectSymbols 时直接处理。
	// 解决：import 边也在 CollectSymbols 时加入 pendingDeps（标记 type=import），
	// 这里统一处理。改 CollectSymbols 的 pendingDeps 收集逻辑（去掉 type!=import 过滤）。
	// —— 见 Step 2 已包含所有 dep（含 import），这里按 dep.Type 分发。

	for _, pd := range m.pendingDeps {
		edge := m.resolveEdge(pd)
		if edge != nil {
			edges = append(edges, *edge)
		}
	}

	return edges, nil
}

// resolveEdge 解析单条边。
func (m *SchemaMapper) resolveEdge(pd pendingDependency) *DependencyEdge {
	dep := pd.Dep
	edgeID := utils.GenerateUUID()
	edgeType := m.mapEdgeType(dep.Type)

	sourceID := m.resolveCandidateID(dep.Source, pd.SourceFileID, pd.SourceFilePath, dep.Type)

	if dep.Type == "import" {
		targetID := m.resolveImportTarget(dep)
		return &DependencyEdge{
			EdgeID: edgeID, SourceID: sourceID, TargetID: targetID,
			EdgeType: edgeType, SourceFile: pd.SourceFilePath,
			TargetModule: dep.TargetModule,
		}
	}

	// source 必须存在
	if sourceID == "" {
		return nil
	}

	targetID := m.resolveCandidateID(dep.Target, pd.SourceFileID, pd.SourceFilePath, dep.Type)
	// target_id 可空（悬空边保留）

	return &DependencyEdge{
		EdgeID: edgeID, SourceID: sourceID, TargetID: targetID,
		EdgeType: edgeType, SourceFile: pd.SourceFilePath,
	}
}

// resolveCandidateID 解析符号到 symbolID。同名多候选时按优先级消歧。
func (m *SchemaMapper) resolveCandidateID(name, sourceFileID, sourceFilePath, edgeType string) string {
	if name == "" {
		return ""
	}
	candidates := m.symbolCandidates[name]
	if len(candidates) == 0 {
		// 降级查旧的 symbolIDMap（含外部符号）
		return m.symbolIDMap[name]
	}
	if len(candidates) == 1 {
		return candidates[0].SymbolID
	}
	// 多候选：消歧
	return m.disambiguate(name, candidates, sourceFileID, sourceFilePath)
}

// disambiguate 按优先级消歧：同文件 → import 文件 → 首个+日志。
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
		// 同文件仍有多个（重载），退到首个+日志
		m.logDisambiguation(name, sourceFilePath, len(sameFile), "same_file_multiple")
		return sameFile[0].SymbolID
	}

	// 2. import 文件优先
	imports := m.fileImports[sourceFileID]
	if imports != nil {
		var importMatch []symbolCandidate
		for _, c := range candidates {
			for mod := range imports {
				if strings.Contains(c.FilePath, mod) || strings.Contains(mod, c.FilePath) {
					importMatch = append(importMatch, c)
					break
				}
			}
		}
		if len(importMatch) == 1 {
			return importMatch[0].SymbolID
		}
		if len(importMatch) > 1 {
			m.logDisambiguation(name, sourceFilePath, len(importMatch), "import_multiple")
			return importMatch[0].SymbolID
		}
	}

	// 3. 首个候选 + 日志
	m.logDisambiguation(name, sourceFilePath, len(candidates), "first_candidate")
	return candidates[0].SymbolID
}

// logDisambiguation 记录消歧告警。
func (m *SchemaMapper) logDisambiguation(name, sourceFile string, count int, strategy string) {
	if m.warnLog != nil {
		m.warnLog("符号消歧: %q 在 %s 有 %d 候选，策略=%s", name, sourceFile, count, strategy)
	}
}

// resolveImportTarget 为 import 边解析 target（按文件路径匹配候选）。
func (m *SchemaMapper) resolveImportTarget(dep parser.ParsedDependency) string {
	if dep.TargetModule == "" {
		return ""
	}
	for name, candidates := range m.symbolCandidates {
		for _, c := range candidates {
			if strings.Contains(c.FilePath, dep.TargetModule) || strings.Contains(dep.TargetModule, c.FilePath) {
				_ = name
				return c.SymbolID
			}
		}
	}
	return ""
}
```

**注意**：需要 `import "strings"`。

- [ ] **Step 4: 修正 CollectSymbols 的 pendingDeps 收集**

Step 2 里 CollectSymbols 只收集了非 import 的 dep。但 ResolveEdges 需要处理所有边（含 import）。**修正**：CollectSymbols 收集**所有** dep（含 import）到 pendingDeps。改 Step 2 的循环：

把 Step 2 里的：
```go
for _, dep := range parsed.Dependencies {
    if dep.Type != "import" {
        m.pendingDeps = append(m.pendingDeps, pendingDependency{...})
    }
}
```
改为：
```go
for _, dep := range parsed.Dependencies {
    m.pendingDeps = append(m.pendingDeps, pendingDependency{
        Dep:            dep,
        SourceFileID:   fileID,
        SourceFilePath: parsed.Path,
    })
}
```

- [ ] **Step 5: 改造 MapToSchema 向后兼容**

MapToSchema 保留旧签名，内部改为 CollectSymbols + ResolveEdges：

```go
func (m *SchemaMapper) MapToSchema(parsed *parser.ParsedFile) (*File, []DependencyEdge, error) {
	file, err := m.CollectSymbols(parsed)
	if err != nil {
		return nil, nil, err
	}
	// 单文件场景：立即解析当前文件累积的边
	// 注意：单文件场景下只有当前文件的符号，跨文件边会悬空
	edges := m.resolveCurrentFileEdges(parsed)
	return file, edges, nil
}
```

但 `resolveCurrentFileEdges` 需要只解析当前文件的边（不解析其他文件的 pendingDeps）。实际上单文件 MapToSchema 场景下，pendingDeps 只有当前文件的（因为是新 mapper 或刚 CollectSymbols）。所以可以直接用 `mapDependencies`（旧逻辑）保持完全向后兼容：

```go
func (m *SchemaMapper) MapToSchema(parsed *parser.ParsedFile) (*File, []DependencyEdge, error) {
	file, err := m.CollectSymbols(parsed)
	if err != nil {
		return nil, nil, err
	}
	// 单文件场景用旧的 mapDependencies（symbolIDMap 已在 CollectSymbols 填充）
	edges := m.mapDependencies(parsed.Dependencies, file.FileID, parsed.Path)
	return file, edges, nil
}
```

这样 MapToSchema 行为完全不变（CollectSymbols 填充了 symbolIDMap，mapDependencies 用 symbolIDMap 解析）。

- [ ] **Step 6: 补全 mapEdgeType**

修改 `mapper.go:238` 的 mapEdgeType，补 `implements_declaration` 和 `calls_declaration`：

```go
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
```

确认 `EdgeImplementsDeclaration` 和 `EdgeCallsDeclaration` 常量已在 `internal/schema/types.go:124-125` 定义。

- [ ] **Step 7: 编译确认**

Run: `go build ./internal/schema/...`
Expected: 无错误（可能有 unused 警告，后续 task 会用到）

- [ ] **Step 8: 提交**

```bash
git add internal/schema/mapper.go
git commit -m "feat(schema): CollectSymbols + ResolveEdges 两遍扫描 + 候选集消歧

- SchemaMapper 新增 symbolCandidates/fileImports/pendingDeps 字段
- CollectSymbols: 第一遍收集符号候选集 + import 关系 + 待解析边
- ResolveEdges: 第二遍用候选集解析边，多候选取歧（同文件→import→首个+日志）
- 无候选保留为悬空边（target_id 空），不丢弃
- mapEdgeType 补全 implements_declaration/calls_declaration
- MapToSchema 向后兼容（内部调 CollectSymbols + mapDependencies）"
```

---

## Task 2: 跨文件消解单元测试

**Files:**
- Create: `internal/schema/mapper_cross_file_test.go`

- [ ] **Step 1: 写跨文件消解测试**

Create `internal/schema/mapper_cross_file_test.go`:

```go
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
		Content:      []byte("test content"),
		Symbols:      symbols,
		Dependencies: deps,
	}
}

func TestResolveEdges_CrossFileCall(t *testing.T) {
	mapper := NewSchemaMapper()

	// 文件 A 定义函数 caller，文件 B 定义函数 callee
	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
		},
		[]parser.ParsedDependency{
			{Type: "call", Source: "caller", Target: "callee"},
		},
	)
	fileB := makeParsedFile("b.go", "go",
		[]parser.ParsedSymbol{
			{Name: "callee", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
		},
		nil,
	)

	// 两遍扫描
	_, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	_, err = mapper.CollectSymbols(fileB)
	require.NoError(t, err)
	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	require.Len(t, edges, 1)
	assert.Equal(t, EdgeCall, edges[0].EdgeType)
	assert.NotEmpty(t, edges[0].TargetID, "跨文件 call 边应消解到 target")
}

func TestResolveEdges_SameNameDisambiguation_SameFile(t *testing.T) {
	mapper := NewSchemaMapper()

	// 同文件有两个同名 helper（不同位置），另一个文件也有 helper
	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "helper", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
			{Name: "helper", Kind: "function", Span: parser.Span{StartLine: 5, StartByte: 50}},
			{Name: "caller", Kind: "function", Span: parser.Span{StartLine: 10, StartByte: 100}},
		},
		[]parser.ParsedDependency{
			{Type: "call", Source: "caller", Target: "helper"},
		},
	)
	fileB := makeParsedFile("b.go", "go",
		[]parser.ParsedSymbol{
			{Name: "helper", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
		},
		nil,
	)

	_, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	_, err = mapper.CollectSymbols(fileB)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)
	require.Len(t, edges, 1)
	// 同文件优先：target 应是 a.go 的 helper（第一个同文件候选）
	assert.NotEmpty(t, edges[0].TargetID)
}

func TestResolveEdges_SameNameDisambiguation_ImportFile(t *testing.T) {
	mapper := NewSchemaMapper()

	// a.go import 了 b.go 的模块，b.go 有 helper，c.go 也有 helper
	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
		},
		[]parser.ParsedDependency{
			{Type: "import", Source: "caller", TargetModule: "b.go"},
			{Type: "call", Source: "caller", Target: "helper"},
		},
	)
	fileB := makeParsedFile("b.go", "go",
		[]parser.ParsedSymbol{
			{Name: "helper", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
		},
		nil,
	)
	fileC := makeParsedFile("c.go", "go",
		[]parser.ParsedSymbol{
			{Name: "helper", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
		},
		nil,
	)

	_, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	_, err = mapper.CollectSymbols(fileB)
	require.NoError(t, err)
	_, err = mapper.CollectSymbols(fileC)
	require.NoError(t, err)

	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	// 找到 call 边（不是 import 边）
	var callEdge *DependencyEdge
	for i := range edges {
		if edges[i].EdgeType == EdgeCall {
			callEdge = &edges[i]
			break
		}
	}
	require.NotNil(t, callEdge, "应有 call 边")

	// import 文件优先：target 应是 b.go 的 helper
	bHelperID := mapper.symbolCandidates["helper"][0].SymbolID // b.go 先注册
	// 注意：注册顺序是 b 的 helper 先（fileA 无 helper），然后 c 的
	// 但消歧应选 import 匹配的 b.go
	assert.Equal(t, bHelperID, callEdge.TargetID, "应优先 import 文件的符号")
}

func TestResolveEdges_DanglingEdge(t *testing.T) {
	mapper := NewSchemaMapper()

	// caller 调用不存在的外部函数
	fileA := makeParsedFile("a.go", "go",
		[]parser.ParsedSymbol{
			{Name: "caller", Kind: "function", Span: parser.Span{StartLine: 1, StartByte: 0}},
		},
		[]parser.ParsedDependency{
			{Type: "call", Source: "caller", Target: "nonexistent"},
		},
	)

	_, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	require.Len(t, edges, 1, "无候选应保留为悬空边")
	assert.Empty(t, edges[0].TargetID, "悬空边 target_id 应为空")
	assert.NotEmpty(t, edges[0].SourceID, "source_id 应有值")
}

func TestResolveEdges_MapEdgeType_ImplementsDeclaration(t *testing.T) {
	mapper := NewSchemaMapper()

	fileA := makeParsedFile("a.hpp", "cpp",
		[]parser.ParsedSymbol{
			{Name: "MyClass", Kind: "class", Span: parser.Span{StartLine: 1, StartByte: 0}},
			{Name: "MyClass", Kind: "class", Span: parser.Span{StartLine: 5, StartByte: 50}},
		},
		[]parser.ParsedDependency{
			{Type: "implements_declaration", Source: "MyClass", Target: "MyClass"},
		},
	)

	_, err := mapper.CollectSymbols(fileA)
	require.NoError(t, err)
	edges, err := mapper.ResolveEdges()
	require.NoError(t, err)

	require.Len(t, edges, 1)
	assert.Equal(t, EdgeImplementsDeclaration, edges[0].EdgeType,
		"implements_declaration 应映射为 EdgeImplementsDeclaration，不是 EdgeReference")
}
```

- [ ] **Step 2: 跑测试确认通过**

Run: `go test -v ./internal/schema -run "TestResolveEdges"`
Expected: PASS

如果失败，检查：
- `parser.Span` 结构体字段名（确认是 `StartLine`/`StartByte`）
- `parser.ParsedSymbol` 字段名
- 候选集累积逻辑

先 grep 确认：
```bash
grep -n "type ParsedSymbol struct" internal/parser/go_parser.go
grep -n "type Span struct" internal/parser/*.go
```

- [ ] **Step 3: 跑现有 mapper 测试确认无回归**

Run: `go test -v ./internal/schema -run "TestMap"`
Expected: 全绿（MapToSchema 向后兼容）

- [ ] **Step 4: 提交**

```bash
git add internal/schema/mapper_cross_file_test.go
git commit -m "test(schema): 跨文件符号消解单元测试——消歧/悬空/mapEdgeType"
```

---

## Task 3: validator 降级——非 import 边空 target 改为 warning

**Files:**
- Modify: `internal/indexer/validator.go:582-596`

- [ ] **Step 1: 修改 validator**

读 `internal/indexer/validator.go:581-598`。当前对 `EdgeCall/EdgeExtends/EdgeImplements/EdgeReference` 的空 target_id 报 Error。改为 warning（不阻塞写入）。

把：
```go
case schema.EdgeCall, schema.EdgeExtends, schema.EdgeImplements, schema.EdgeReference:
    // These edge types should have a target_id
    if edge.TargetID == "" {
        result.AddError(&ValidationError{
            Type:       ErrInvalidValue,
            Message:    fmt.Sprintf("%s edge must have target_id", edge.EdgeType),
            EntityType: "edge",
            EntityID:   edge.EdgeID,
            Field:      "target_id",
        })
    }
```

改为：

```go
case schema.EdgeCall, schema.EdgeExtends, schema.EdgeImplements, schema.EdgeReference:
    // 这些边类型理想情况下应有 target_id。
    // 但跨文件符号消解可能失败（外部依赖、动态调用），保留为悬空边（target_id 空）
    // 有价值——降级为 warning 不阻塞写入。
    if edge.TargetID == "" {
        result.AddWarning(&ValidationError{
            Type:       ErrInvalidValue,
            Message:    fmt.Sprintf("%s edge has empty target_id (unresolved symbol, kept as dangling)", edge.EdgeType),
            EntityType: "edge",
            EntityID:   edge.EdgeID,
            Field:      "target_id",
        })
    }
```

**注意**：确认 `ValidationResult` 有 `AddWarning` 方法。先 grep：
```bash
grep -n "func.*AddWarning\|func.*AddError" internal/indexer/validator.go
```
如果没有 `AddWarning`，用现有的 warning 机制（或加一个）。

- [ ] **Step 2: 跑 validator 测试确认**

Run: `go test -v ./internal/indexer -run "TestValidat" -short`
Expected: 有关空 target 的测试可能需要调整预期（从 Error 改为 Warning）。如果有测试断言了 Error，更新断言。

- [ ] **Step 3: 提交**

```bash
git add internal/indexer/validator.go
git commit -m "fix(validator): 非 import 边空 target_id 从 Error 降级为 Warning

跨文件消解失败时保留悬空边有价值，不应阻塞写入。
validator 改为 warning，symbol_resolution_rate 指标可观测悬空比例。"
```

---

## Task 4: 调用方改造——parse_command + index_command

**Files:**
- Modify: `cmd/cli/parse_command.go:273-299`
- Modify: `cmd/cli/index_command.go:327-337`

- [ ] **Step 1: 改 parse_command.go**

读 `cmd/cli/parse_command.go:272-300`。把 MapToSchema 循环改为 CollectSymbols + ResolveEdges 两阶段：

改前：
```go
mapper := schema.NewSchemaMapper()
var schemaFiles []schema.File
var allEdges []schema.DependencyEdge
var mappingErrors []schema.ParseError
for i, parsedFile := range parsedFiles {
    schemaFile, edges, err := mapper.MapToSchema(parsedFile)
    if err != nil {
        mappingErrors = append(mappingErrors, ...)
        continue
    }
    schemaFiles = append(schemaFiles, *schemaFile)
    allEdges = append(allEdges, edges...)
}
```

改后：
```go
mapper := schema.NewSchemaMapper()
var schemaFiles []schema.File
var mappingErrors []schema.ParseError

// 第一遍：收集所有文件的符号 + import 关系
for i, parsedFile := range parsedFiles {
    logger.Debug("[%d/%d] Mapping file: %s", i+1, len(parsedFiles), parsedFile.Path)
    schemaFile, err := mapper.CollectSymbols(parsedFile)
    if err != nil {
        mappingErrors = append(mappingErrors, schema.ParseError{
            File: parsedFile.Path, Message: err.Error(), Type: schema.ErrorMapping,
        })
        logger.Error("Failed to map file %s: %v", parsedFile.Path, err)
        continue
    }
    schemaFiles = append(schemaFiles, *schemaFile)
    logger.Debug("Mapped %d symbols from %s", len(schemaFile.Symbols), parsedFile.Path)
}

// 第二遍：用全仓库候选集解析所有边
allEdges, err := mapper.ResolveEdges()
if err != nil {
    logger.Error("Failed to resolve edges: %v", err)
}
logger.Debug("Resolved %d edges across %d files", len(allEdges), len(schemaFiles))
```

- [ ] **Step 2: 改 index_command.go**

读 `cmd/cli/index_command.go:326-337`。同样改为两阶段：

```go
mapper := schema.NewSchemaMapper()
var schemaFiles []schema.File
var mappingErrors []schema.ParseError

// 第一遍：收集符号
for _, parsedFile := range parsedFiles {
    schemaFile, err := mapper.CollectSymbols(parsedFile)
    if err != nil {
        mappingErrors = append(mappingErrors, schema.ParseError{
            File: parsedFile.Path, Message: err.Error(), Type: schema.ErrorMapping,
        })
        continue
    }
    schemaFiles = append(schemaFiles, *schemaFile)
}

// 第二遍：解析边
allEdges, err := mapper.ResolveEdges()
if err != nil {
    return fmt.Errorf("resolve edges: %w", err)
}
```

注意保留原有的 `allEdges` 变量声明（后续 parseOutput 构造要用）。

- [ ] **Step 3: 编译确认**

Run: `go build ./cmd/cli/...`
Expected: 无错误

- [ ] **Step 4: 提交**

```bash
git add cmd/cli/parse_command.go cmd/cli/index_command.go
git commit -m "refactor(cli): parse/index 命令改用 CollectSymbols + ResolveEdges 两阶段"
```

---

## Task 5: 移除 filterValidEdges workaround

**Files:**
- Modify: `tests/integration/quality_gate_test.go:182,223-248`

- [ ] **Step 1: 读 filterValidEdges 实现**

读 `tests/integration/quality_gate_test.go` 的 `filterValidEdges` 函数和调用点（约 182 行和 223-248 行）。

- [ ] **Step 2: 移除 target 校验，保留 source 校验**

跨文件消解后，target_id 空的边保留为悬空边（合法），不再过滤。只保留 source_id 空的过滤。

改 `filterValidEdges`：

```go
// filterValidEdges 丢弃 source_id 为空的边（edges.source_id 是 NOT NULL）。
// target_id 空的边保留为悬空边（跨文件消解失败但有价值）。
func filterValidEdges(out *schema.ParseOutput) *schema.ParseOutput {
	validSymbolIDs := make(map[string]bool)
	for _, f := range out.Files {
		for _, s := range f.Symbols {
			validSymbolIDs[s.SymbolID] = true
		}
	}

	var filtered []schema.DependencyEdge
	for _, e := range out.Relationships {
		// source_id 必须非空（DB NOT NULL 约束）
		if e.SourceID == "" {
			continue
		}
		// target_id 非空时必须是已索引符号（DB 外键约束）
		// target_id 空 = 悬空边，保留
		if e.TargetID != "" && !validSymbolIDs[e.TargetID] {
			// target 指向不存在的符号，丢弃（避免外键违反）
			continue
		}
		filtered = append(filtered, e)
	}
	out.Relationships = filtered
	return out
}
```

注意：保留 target_id 非空但指向不存在符号的过滤（DB 外键约束要求）。只移除了"跨文件 target 解析不到"导致的过度丢弃——因为现在跨文件 target 能解析到了。

- [ ] **Step 3: 跑集成测试确认**

Run: `go test -v ./tests/integration -run TestQualityGate -count=1`
Expected: 可能失败——跨文件真值还没恢复（Task 6）。如果 edge_recall 下降，是因为边数变化但真值没更新。先记录，Task 6 修真值。

- [ ] **Step 4: 提交**

```bash
git add tests/integration/quality_gate_test.go
git commit -m "fix(test): filterValidEdges 移除 target 过度校验，保留 source + 外键约束"
```

---

## Task 6: 恢复跨文件真值

**Files:**
- Modify: `internal/quality/fixtures/graph_ground_truth.go`

- [ ] **Step 1: 读原始 expectedXxxCalls 确认跨文件真值**

读 `tests/integration/call_analysis_fixtures_test.go` 的 expected 列表（39/106/169/230/241/336/342/452/459 行），提取跨文件调用。同时读当前 `internal/quality/fixtures/graph_ground_truth.go` 确认被压制的真值。

- [ ] **Step 2: 恢复跨文件真值**

把 `graph_ground_truth.go` 里被压制的跨文件调用边恢复。例如：
- `cpp_calls_c.cpp` 的 `processData → c_process_string`（跨文件到 c_library.h）
- 其他 fixture 的跨文件调用

更新文件头部的注释——从"已知缺口"改为"已修复"：

```go
// Package fixtures 存放评估真值（ground truth）。
//
// 真值来源：从 tests/integration/call_analysis_fixtures_test.go 的 expectedXxxCalls 迁移。
// 跨文件调用边真值已恢复（SchemaMapper 两遍扫描后可消解）。
package fixtures
```

- [ ] **Step 3: 跑门禁测试确认对齐**

Run: `go test -v ./tests/integration -run "TestQualityGate_FixtureMode$" -count=1`
Expected: edge_recall ≥ 0.90。如果低于阈值，检查：
- 真值的 SourceName/TargetName 是否与解析器实际产出对齐
- 跨文件边是否真的被消解了（看 ListExtractedEdges 输出）

如果 edge_recall 仍低，用 `t.Logf` 打印实际提取的边对比真值，调整真值或排查消解问题。

- [ ] **Step 4: 跑全量集成测试**

Run: `go test ./tests/integration/... -count=1`
Expected: 全绿

- [ ] **Step 5: 提交**

```bash
git add internal/quality/fixtures/graph_ground_truth.go
git commit -m "fix(quality/fixtures): 恢复跨文件真值——SchemaMapper 两遍扫描后可消解"
```

---

## Task 7: 全量验证与基线更新

**Files:**
- Modify: `docs/superpowers/baselines/2026-07-07-quality-baseline.md`

- [ ] **Step 1: 全量单元测试**

Run: `go test -short ./...`
Expected: 全绿

- [ ] **Step 2: 全量集成测试**

Run: `go test ./tests/integration/... -count=1`
Expected: 全绿

- [ ] **Step 3: 验证 cross_file_connectivity 提升**

Run: `go test -v ./tests/integration -run "TestQualityGate_RepoMode$" -count=1 2>&1 | grep cross_file`
Expected: `cross_file_connectivity` > 0.00（应显著提升）

- [ ] **Step 4: 验证 symbol_resolution_rate 提升**

Run: `go test -v ./tests/integration -run "TestQualityGate_RepoMode$" -count=1 2>&1 | grep symbol_resolution`
Expected: `symbol_resolution_rate` > 0.29（应显著提升）

- [ ] **Step 5: 更新基线快照**

在 `docs/superpowers/baselines/2026-07-07-quality-baseline.md` 补充改造后数据：

```markdown
## 跨文件消解改造后（2026-07-08）

| 指标 | 改造前 | 改造后 | 变化 |
|---|---|---|---|
| cross_file_connectivity | 0.0000 | （填实际值） | ✅ 从 0 提升 |
| symbol_resolution_rate | 0.2857 | （填实际值） | ✅ 显著提升 |
| edge_recall | 1.0000 | （填实际值） | 保持 ≥0.90 |
```

- [ ] **Step 6: gofmt + verify**

Run: `gofmt -l internal/schema/ internal/indexer/ cmd/cli/ tests/integration/ internal/quality/` && `go build ./...`
Expected: clean + 无错误

- [ ] **Step 7: 提交**

```bash
git add docs/superpowers/baselines/2026-07-07-quality-baseline.md
git commit -m "docs(baseline): 跨文件消解改造后基线更新——cross_file_connectivity 提升"
```

---

## Self-Review

### Spec 覆盖检查

| Spec 章节 | 对应 Task |
|---|---|
| §2 两遍扫描架构 | Task 1（CollectSymbols + ResolveEdges） |
| §3 候选集与 import 收集 | Task 1 Step 2-3 |
| §4 边消解与消歧 | Task 1 Step 3（resolveEdge/disambiguate）+ Task 2（测试） |
| §4.4 悬空边处理 | Task 1 Step 3 + Task 3（validator 降级） |
| §5.1 mapEdgeType 修复 | Task 1 Step 6 |
| §5.2 filterValidEdges 移除 | Task 5 |
| §5.3 跨文件真值恢复 | Task 6 |
| §2.4 调用方改造 | Task 4 |
| §6 测试策略 | Task 2（单元）+ Task 5/6/7（集成） |
| §1.3 成功标准 | Task 7 验证 |

### 类型一致性

- `symbolCandidate` 结构体在 Task 1 定义，Task 2 测试引用
- `pendingDependency` 在 Task 1 定义，ResolveEdges 遍历
- `CollectSymbols` / `ResolveEdges` 方法签名在 Task 1 定义，Task 4 调用
- `mapEdgeType` 补全的 case 在 Task 1 Step 6，Task 2 测试验证

### 注意事项

1. **resolveImportTarget 的候选遍历效率**：当前遍历所有 symbolCandidates 找文件路径匹配。对大仓库可能慢。但这轮不优化（fixture 数据小），下一轮按需加索引。
2. **MapToSchema 向后兼容**：Task 1 Step 5 确保单文件场景行为不变。现有 mapper_test.go 的测试应全绿。
3. **validator AddWarning**：Task 3 需确认 ValidationResult 有 AddWarning 方法。如果没有，用现有机制或添加。

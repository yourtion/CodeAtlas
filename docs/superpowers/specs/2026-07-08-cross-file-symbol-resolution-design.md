# 跨文件符号消解设计（精准依赖图）

- **日期**: 2026-07-08
- **状态**: 已批准（设计阶段），待写实现计划
- **范围**: SchemaMapper 两遍扫描 + 候选集消歧，修复跨文件调用边丢失问题，附带修复 mapEdgeType/filterValidEdges/真值恢复
- **不做**: 签名消歧、edges 多对多、动态调用消解、跨语言符号绑定

---

## 1. 背景与目标

### 1.1 问题来源

质量评估系统（PR#3）跑出基线数据，明确暴露了 SchemaMapper 的跨文件解析缺失：

| 指标 | 基线值 | 健康基线 |
|---|---|---|
| `cross_file_connectivity` | **0.0000** | > 0.20 |
| `symbol_resolution_rate` | **0.2857** | > 0.70 |

根因：`internal/schema/mapper.go:45` 每次 `MapToSchema` 调用清空 `symbolIDMap`，导致跨文件边的 target 查不到，被 `mapDependency:220-224` 的 `if targetID == "" { return nil }` 丢弃。

### 1.2 核心改造

把 SchemaMapper 从"单文件作用域"改成"全仓库两遍扫描"：
- **第一遍 CollectSymbols**：遍历所有文件，累积符号候选集 + import 关系图 + 待解析边
- **第二遍 ResolveEdges**：遍历累积的边，用候选集 + 消歧策略解析 target

### 1.3 成功标准

1. `cross_file_connectivity` 从 0.00 提升到 > 0.20
2. `symbol_resolution_rate` 从 0.29 提升到 > 0.50（call 类 > 0.70）
3. 跨文件真值恢复后，`edge_recall` 仍 ≥ 0.90
4. 现有测试无回归（`make test` + `make test-integration` 全绿）
5. 消歧日志可观测（同名歧义场景有 WARN 日志）
6. `mapEdgeType` 补全 `implements_declaration` → `EdgeImplementsDeclaration`

---

## 2. 架构与两遍扫描流程

```
调用方（parse_command / index_command / 集成测试）
    │
    │  第一遍：循环每个文件
    ▼
CollectSymbols(parsedFile) → (*File, error)
    - 生成 fileID、symbolID（确定性，与现有逻辑一致）
    - 把 (符号名 → []candidate) 累积到 symbolCandidates map（不覆盖）
    - 收集 import 关系（fileImports[fileID][targetModule] = true）
    - 收集待解析边（pendingDeps）
    - 不解析 call/extends/implements 等边
    │
    │  所有文件收集完后
    ▼
ResolveEdges() → ([]DependencyEdge, error)
    - 遍历 pendingDeps
    - 对每条边 target 查候选集 → 唯一消解 / 消歧 / 悬空保留
    - 返回所有边
```

### 2.1 SchemaMapper 新增字段

```go
type SchemaMapper struct {
    // 候选集：符号名 → 该名字的所有候选（累积，不覆盖）
    symbolCandidates map[string][]symbolCandidate
    // import 关系：fileID → 该文件 import 过的模块/文件路径集合
    fileImports map[string]map[string]bool
    // 累积的待解析依赖（第一遍收集，第二遍解析）
    pendingDeps []pendingDependency
    // 外部符号（保持不变）
    externalSymbols map[string]*Symbol
    // 日志函数（消歧告警用）
    warnLog func(format string, args ...interface{})
}

type symbolCandidate struct {
    SymbolID string  // 确定性 UUID（fileID:name:startLine:startByte）
    FileID   string
    FilePath string  // 用于 import 消歧时路径匹配
}

type pendingDependency struct {
    Dep            parser.ParsedDependency
    SourceFileID   string
    SourceFilePath string
}
```

### 2.2 新增方法签名

```go
// CollectSymbols 第一遍：收集文件符号 + import 关系 + 待解析边，不解析边
func (m *SchemaMapper) CollectSymbols(parsed *parser.ParsedFile) (*File, error)

// ResolveEdges 第二遍：用累积的候选集解析所有边
func (m *SchemaMapper) ResolveEdges() ([]DependencyEdge, error)
```

### 2.3 MapToSchema 向后兼容

`MapToSchema` 保留（现有签名 `(File, edges, error)` 不变），内部改为调 `CollectSymbols` + 立即 `ResolveEdges`。单文件场景行为不变（同文件边照常解析；跨文件边因没有其他文件符号而保留悬空）。这样不破坏现有 `MapToSchema(singleFile)` 的测试用法。

### 2.4 调用方改造

两处生产调用点（`parse_command.go:273` / `index_command.go:327`）：

```go
// 改前
mapper := schema.NewSchemaMapper()
for _, parsedFile := range parsedFiles {
    file, edges, err := mapper.MapToSchema(parsedFile)
    // edges 合并
}

// 改后
mapper := schema.NewSchemaMapper()
var files []schema.File
for _, parsedFile := range parsedFiles {
    file, err := mapper.CollectSymbols(parsedFile)
    files = append(files, *file)
}
edges, err := mapper.ResolveEdges()
```

---

## 3. 第一遍：符号候选集与 import 关系收集

### 3.1 CollectSymbols 流程

```
CollectSymbols(parsedFile):
  1. 生成 fileID（确定性，基于 path + checksum）——与现有逻辑一致
  2. 创建 File 对象
  3. 遍历 parsedFile.Symbols:
     - mapSymbol 生成 symbolID（确定性）——与现有逻辑一致
     - symbolCandidates[name] = append(..., candidate)  // 累积，不覆盖
     - file.Symbols = append(...)
  4. 收集 import 关系:
     - 遍历 parsedFile.Dependencies 里 Type=="import" 的
     - fileImports[fileID][targetModule] = true
  5. 收集待解析边:
     - 遍历 parsedFile.Dependencies 里 Type!="import" 的
     - pendingDeps = append(pendingDeps, {dep, fileID, filePath})
  6. 收集外部模块符号（保持现有逻辑，externalSymbols 累积）
  7. 映射 AST 节点（保持现有逻辑）
  8. 返回 *File（不含边）
```

### 3.2 import 关系图的语义

`fileImports` 记录"文件 A import 了什么"。消歧时用来判断"边的 source 文件是否 import 过 target 符号所在文件"。

**import target 的形式**（因语言而异）：
- C/C++：`#include "c_library.h"` → targetModule = `"c_library.h"`
- Go：`import "fmt"` → targetModule = `"fmt"`
- JS/TS：`import {x} from "./utils"` → targetModule = `"./utils"`
- Java：`import com.example.Utils` → targetModule = `"com.example.Utils"`

**匹配规则**：消歧时，候选的 `FilePath` 是否与 import 的 targetModule 有后缀/路径匹配关系。如 import `"c_library.h"`，候选文件路径含 `c_library.h` 则视为 import 过。匹配不需要 100% 精确——它是消歧的**优先级提示**，不是硬约束。

---

## 4. 第二遍：边消解与候选集消歧

### 4.1 ResolveEdges 流程

```
ResolveEdges():
  for each pendingDep in pendingDeps:
    sourceID = resolveSymbolID(pendingDep.dep.Source, pendingDep.SourceFileID)
    targetCandidates = symbolCandidates[pendingDep.dep.Target]

    if pendingDep.dep.Type == "import":
      // import 边特殊处理（保持现有逻辑）
      edge.TargetModule = dep.TargetModule
      edge.TargetID = resolveImportTarget(targetCandidates, dep, sourceFileID)

    else if len(targetCandidates) == 0:
      // 无候选：保留为悬空边（target_id 空）

    else if len(targetCandidates) == 1:
      // 唯一候选：直接消解
      edge.TargetID = targetCandidates[0].SymbolID

    else:
      // 多候选：按优先级消歧
      edge.TargetID = disambiguate(targetCandidates, pendingDep)

    if sourceID == "":
      continue  // source 必须存在，否则丢弃
    edges = append(edges, edge)
```

### 4.2 消歧优先级（disambiguate）

按以下顺序尝试，命中即返回：

```
disambiguate(candidates, dep):
  1. 同文件优先：candidates 里 FilePath == dep.SourceFilePath 的
     - 若唯一 → 返回它
     - 若多个（同文件重载）→ 退到步骤 3

  2. import 文件优先：candidates 里 FilePath 在 source 文件的 import 集合里的
     - fileImports[sourceFileID] 里是否有匹配 candidate.FilePath 的
     - 若唯一 → 返回它
     - 若多个 → 退到步骤 3

  3. 仍歧义：选第一个候选（注册顺序），记 WARN 日志
     - 日志内容：符号名、source 文件、边类型、候选数
```

**为什么不用签名消歧**：`ParsedDependency.Target` 只有裸名（如 `"Add"`），没有签名信息。消歧时拿不到 target 的签名做匹配。签名消歧需要解析器产出更丰富的 Target 字段，超出这轮范围。

### 4.3 import 边的 target 消解

import 边的 target 是模块名（如 `"c_library.h"`），不是符号名。`symbolCandidates` 按符号名索引，查不到模块名。处理方式：

```
resolveImportTarget(candidates, dep, sourceFileID):
  for each candidate in candidates:
    if candidate.FilePath 含 dep.TargetModule:
      return candidate.SymbolID
  return ""  // 外部依赖，保留悬空
```

### 4.4 悬空边处理

- `edges.target_id` 可空（migration `20260101000001_init.sql:129`）
- 无候选的非 import 边保留为悬空边（target_id 空），不丢弃
- source_id 空的边丢弃（source 必须存在）

### 4.5 validator 调整

`validator.go:581-598` 对非 import 边的空 target 当前报 error。改为：
- 非 import 边 target_id 空：降级为 **warning**（不阻塞写入）
- 原因：跨文件消解失败但边有保留价值

### 4.6 与现有行为的对比

| 场景 | 现有 mapDependency | 新 ResolveEdges |
|---|---|---|
| 同文件、唯一 target | ✅ 消解 | ✅ 消解（同文件优先命中） |
| 跨文件、唯一 target | ❌ 丢弃（map 重置） | ✅ 消解 |
| 跨文件、多候选 | ❌ 丢弃 | ✅ 按优先级消歧 |
| 无候选（外部依赖） | ❌ 非 import 丢弃 | ✅ 保留为悬空边 |
| import 边 | ✅ 保留（TargetModule） | ✅ 保留（不变） |

---

## 5. 附带修复

### 5.1 修复 mapEdgeType（`mapper.go:238`）

现有 `mapEdgeType` 缺 `implements_declaration` 分支，走 default 变成 `EdgeReference`。补全：

```go
case "implements_declaration":
    return EdgeImplementsDeclaration
case "calls_declaration":        // 顺带补全（C/C++ 解析器产出）
    return EdgeCallsDeclaration
```

### 5.2 移除 filterValidEdges（`quality_gate_test.go:182,223`）

跨文件消解后，原先被丢弃的边现在能消解。`filterValidEdges` 简化：
- `source_id 必须非空`：**保留**（source 是必须的）
- `target_id 非空时必须指向已索引符号`：**移除**（消解不到的保留为悬空）

### 5.3 恢复跨文件真值（`graph_ground_truth.go`）

前面因跨文件边被丢弃，真值被压缩为只有同文件边。现在恢复跨文件真值——从 `tests/integration/call_analysis_fixtures_test.go` 的原始 `expectedXxxCalls` 重新提取跨文件调用。真值文件头部的"已知缺口"注释更新为"已修复"。

---

## 6. 测试策略

| 层 | 测试文件 | 类型 | 覆盖 |
|---|---|---|---|
| schema | `mapper_cross_file_test.go`（新建） | 单元 | 跨文件 call 边消解、同名消歧（同文件/import/首个）、悬空边保留 |
| schema | `mapper_test.go`（扩展） | 单元 | mapEdgeType 补全、CollectSymbols+ResolveEdges 两阶段、MapToSchema 向后兼容 |
| 集成 | `quality_gate_test.go`（改） | 集成 | 移除 filterValidEdges、恢复跨文件真值、门禁重新对齐 |
| 集成 | 全量回归 | 集成 | `make test-integration` 确保现有边数/符号数测试不误判 |

**关键测试用例**（`mapper_cross_file_test.go`）：
- `TestResolveEdges_CrossFileCall`：文件 A 调用文件 B 的函数 → 消解成功
- `TestResolveEdges_SameNameDisambiguation_SameFile`：同文件同名 → 同文件优先
- `TestResolveEdges_SameNameDisambiguation_ImportFile`：跨文件同名 → import 优先
- `TestResolveEdges_SameNameDisambiguate_FirstCandidate`：无 import 的多候选 → 首个+不报错
- `TestResolveEdges_DanglingEdge`：target 无候选 → 保留悬空边
- `TestResolveEdges_ImportEdge`：import 边保持现有行为

---

## 7. 不做的事（YAGNI）

- ❌ 签名消歧（`ParsedDependency.Target` 无签名信息，需解析器层改造）
- ❌ edges 表多对多候选（保留为悬空边已够，不需中间表）
- ❌ 动态/反射调用的符号消解（运行时信息静态分析拿不到）
- ❌ 跨语言符号消解（如 Kotlin→Java 的符号绑定，由解析器层互操作检测处理）
- ❌ SchemaMapper 并发安全（当前单线程顺序调用，无需加锁）

---

## 8. 风险与缓解

| 风险 | 缓解 |
|---|---|
| 消歧策略误连（同名符号连错） | 首个候选+WARN 日志；评估报告 neighbor_hit_rate 异常下降可发现 |
| 现有测试因边数变化而断言失败 | 现有测试断言"至少有边"或"边类型正确"，非精确边数；精确断言按实际调整 |
| MapToSchema 向后兼容的语义变化 | 单文件场景行为不变（同文件边照常解析） |
| validator 放宽后悬空边大量入库 | symbol_resolution_rate 指标可观测；悬空边不占额外存储 |

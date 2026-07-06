# 解析器开发指南

本指南详细说明如何为 CodeAtlas 添加新的编程语言解析器。

## 目录

- [概述](#概述)
- [解析器架构](#解析器架构)
- [解析器接口规范](#解析器接口规范)
- [Tree-sitter 集成](#tree-sitter-集成)
- [添加新语言解析器](#添加新语言解析器)
- [跨语言调用检测](#跨语言调用检测)
- [Parser Pool 并发策略](#parser-pool-并发策略)
- [文件扫描和过滤规则](#文件扫描和过滤规则)
- [测试指南](#测试指南)
- [最佳实践](#最佳实践)

## 概述

CodeAtlas 使用 **Tree-sitter** 作为核心解析引擎,为 9 种编程语言提供精确的语法解析能力。解析器负责:

1. **提取符号** - 函数、类、方法、变量等代码元素
2. **分析依赖** - import、继承、调用关系
3. **构建 AST** - 生成完整的语法树
4. **跨语言检测** - 识别跨语言调用关系

### 支持的语言

| 语言 | 解析器 | 文件扩展名 | 跨语言支持 |
|------|--------|------------|------------|
| Go | `go_parser.go` | `.go` | - |
| JavaScript/TypeScript | `js_parser.go` | `.js`, `.ts`, `.jsx`, `.tsx` | TS → JS |
| Python | `python_parser.go` | `.py` | - |
| Kotlin | `kotlin_parser.go` | `.kt`, `.kts` | Kotlin → Java |
| Java | `java_parser.go` | `.java` | - |
| Swift | `swift_parser.go` | `.swift` | Swift → Objective-C |
| Objective-C | `objc_parser.go` | `.m`, `.h` | - |
| C | `c_parser.go` | `.c`, `.h` (C) | - |
| C++ | `cpp_parser.go` | `.cpp`, `.cc`, `.cxx`, `.hpp`, `.hh`, `.hxx`, `.h` (C++) | C++ → C |

## 解析器架构

### 核心组件

```
┌─────────────────────┐
│   FileScanner       │  扫描文件系统,识别语言
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│  ParserPool         │  并发解析管理 (工作池模式)
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ TreeSitterParser    │  Tree-sitter 封装,统一接口
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ LanguageParser      │  语言特定解析器 (Go, Kotlin, Swift...)
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│   ParsedFile        │  解析结果 (符号 + 依赖)
└─────────────────────┘
```

### 数据流

```
ScannedFile → LanguageParser → ParsedFile
     │                │                │
     │                │                ├── Symbols (代码符号)
     │                │                └── Dependencies (依赖关系)
     │                │
     │                └── Tree-sitter Query (查询 AST)
     │
     └── 文件路径 + 语言 + 内容
```

## 解析器接口规范

所有语言解析器必须实现以下接口:

```go
// LanguageParser 定义语言解析器的通用接口
type LanguageParser interface {
    // Parse 解析文件并提取符号和依赖关系
    Parse(file ScannedFile) (*ParsedFile, error)

    // Language 返回解析器支持的语言名称
    Language() string
}
```

### 核心数据结构

#### ScannedFile

```go
type ScannedFile struct {
    Path     string // 相对路径
    AbsPath  string // 绝对路径
    Language string // 检测到的语言
    Size     int64  // 文件大小
}
```

#### ParsedFile

```go
type ParsedFile struct {
    Path         string          // 文件路径
    Language     string          // 语言标识 (go, kotlin, swift...)
    Content      []byte          // 原始文件内容
    Checksum     string          // 文件校验和 (用于增量索引)
    RootNode     *sitter.Node    // Tree-sitter 根节点
    Symbols      []ParsedSymbol  // 提取的符号
    Dependencies []ParsedDependency // 依赖关系
}
```

#### ParsedSymbol

```go
type ParsedSymbol struct {
    Name      string          // 符号名称
    Kind      string          // 符号类型 (function, class, method...)
    Signature string          // 完整签名
    Span      ParsedSpan      // 位置信息
    Docstring string          // 文档字符串
    Node      *sitter.Node    // AST 节点引用
    Children  []ParsedSymbol  // 子符号 (类成员、接口方法等)
}

type ParsedSpan struct {
    StartLine int // 起始行号 (1-based)
    EndLine   int // 结束行号 (1-based)
    StartByte int // 起始字节偏移
    EndByte   int // 结束字节偏移
}
```

#### ParsedDependency

```go
type ParsedDependency struct {
    Type         string // 关系类型: import, call, extends, implements
    Source       string // 源符号名称
    Target       string // 目标符号/模块名称
    TargetModule string // 目标模块路径 (用于 import)
    IsExternal   bool   // 是否为外部依赖 (第三方库)
}
```

## Tree-sitter 集成

### Tree-sitter 优势

- **增量解析** - 只重新解析变更的代码段
- **错误恢复** - 即使语法错误也能继续解析
- **统一接口** - 所有语言使用相同的查询语言
- **高性能** - C 语言实现,解析速度快

### Tree-sitter 初始化

Tree-sitter 封装在 `TreeSitterParser` 中:

```go
// internal/parser/treesitter.go

type TreeSitterParser struct {
    // 为每种语言维护独立的 parser 和 language 实例
    goParser     *sitter.Parser
    goLang       *sitter.Language
    kotlinParser *sitter.Parser
    kotlinLang   *sitter.Language
    // ... 其他语言
}

func NewTreeSitterParser() (*TreeSitterParser, error)
func (p *TreeSitterParser) Parse(content []byte, language string) (*sitter.Node, error)
func (p *TreeSitterParser) Query(node *sitter.Node, queryString string, language string) ([]*sitter.QueryMatch, error)
```

### Tree-sitter 查询语言

使用类似 CSS 选择器的语法查询 AST:

```scheme
; 查询函数声明
(function_declaration
  name: (identifier) @func.name) @func.def

; 查询类声明
(class_declaration
  name: (type_identifier) @class.name) @class.def

; 查询导入语句
(import_declaration
  (identifier) @import.path)
```

### 查询示例

```go
// 执行查询
query := `(function_declaration name: (identifier) @func.name) @func.def`

matches, err := p.tsParser.Query(rootNode, query, "kotlin")
if err != nil {
    return err
}

// 处理匹配结果
for _, match := range matches {
    for _, capture := range match.Captures {
        if capture.Index == 0 { // func.name
            funcName := capture.Node.Content(content)
            // 处理函数名...
        }
    }
}
```

## 添加新语言解析器

### 步骤 1: 准备工作

#### 1.1 确认 Tree-sitter 语法库

首先检查是否有官方或社区维护的 Tree-sitter 语法库:

```bash
# 搜索可用的语法库
# https://tree-sitter.github.io/tree-sitter#available-parsers
```

常用语法库:

- **官方语法**: [tree-sitter/tree-sitter](https://github.com/tree-sitter/tree-sitter)
- **Go 绑定**: [smacker/go-tree-sitter](https://github.com/smacker/go-tree-sitter)
- **社区语法**: [tree-sitter-grammars](https://github.com/tree-sitter-grammars)

#### 1.2 添加依赖

在 `go.mod` 中添加语言绑定:

```bash
# 示例: 添加 Rust 支持
go get github.com/smacker/go-tree-sitter/rust
```

### 步骤 2: 注册语言

在 `internal/parser/treesitter.go` 中注册新语言:

```go
import (
    // ... 其他导入
    "github.com/smacker/go-tree-sitter/rust"
)

type TreeSitterParser struct {
    // ... 现有字段
    rustParser *sitter.Parser
    rustLang   *sitter.Language
}

func NewTreeSitterParser() (*TreeSitterParser, error) {
    tsp := &TreeSitterParser{}

    // 初始化 Rust parser
    tsp.rustLang = rust.GetLanguage()
    tsp.rustParser = sitter.NewParser()
    tsp.rustParser.SetLanguage(tsp.rustLang)

    return tsp, nil
}

func (p *TreeSitterParser) Parse(content []byte, language string) (*sitter.Node, error) {
    // ... 现有代码
    switch language {
    // ... 现有 case
    case "rust", "rs":
        parser = p.rustParser
    default:
        return nil, fmt.Errorf("unsupported language: %s", language)
    }
    // ...
}

func (p *TreeSitterParser) Query(/* ... */) {
    // ... 添加 rust case
}
```

### 步骤 3: 添加文件扩展名映射

在 `internal/parser/scanner.go` 的 `determineLanguage()` 函数中添加:

```go
func determineLanguage(path string) string {
    ext := strings.ToLower(filepath.Ext(path))

    languageMap := map[string]string{
        // ... 现有映射
        ".rs": "Rust",
    }

    if lang, exists := languageMap[ext]; exists {
        return lang
    }

    return "Unknown"
}
```

### 步骤 4: 创建解析器实现

创建 `internal/parser/rust_parser.go`:

```go
package parser

import (
    "fmt"
    "strings"

    sitter "github.com/smacker/go-tree-sitter"
)

// RustParser parses Rust source code using Tree-sitter
type RustParser struct {
    tsParser *TreeSitterParser
}

// NewRustParser creates a new Rust parser
func NewRustParser(tsParser *TreeSitterParser) *RustParser {
    return &RustParser{
        tsParser: tsParser,
    }
}

// Parse parses a Rust file and extracts symbols and dependencies
func (p *RustParser) Parse(file ScannedFile) (*ParsedFile, error) {
    // 1. 读取文件内容
    content, err := readFileContent(file.AbsPath)
    if err != nil {
        return nil, &DetailedParseError{
            File:    file.Path,
            Message: fmt.Sprintf("failed to read file: %v", err),
            Type:    "filesystem",
        }
    }

    // 2. 使用 Tree-sitter 解析
    rootNode, parseErr := p.tsParser.Parse(content, "rust")

    parsedFile := &ParsedFile{
        Path:     file.Path,
        Language: "rust",
        Content:  content,
        RootNode: rootNode,
    }

    if rootNode == nil {
        return parsedFile, &DetailedParseError{
            File:    file.Path,
            Message: fmt.Sprintf("failed to parse Rust file: %v", parseErr),
            Type:    "parse",
        }
    }

    // 3. 提取符号 (按优先级顺序)
    if err := p.extractCrates(rootNode, parsedFile, content); err != nil {
        // 非致命错误,继续
    }

    if err := p.extractModules(rootNode, parsedFile, content); err != nil {
        // 非致命错误,继续
    }

    if err := p.extractFunctions(rootNode, parsedFile, content); err != nil {
        // 非致命错误,继续
    }

    if err := p.extractStructs(rootNode, parsedFile, content); err != nil {
        // 非致命错误,继续
    }

    if err := p.extractImpls(rootNode, parsedFile, content); err != nil {
        // 非致命错误,继续
    }

    if err := p.extractTraits(rootNode, parsedFile, content); err != nil {
        // 非致命错误,继续
    }

    if err := p.extractImports(rootNode, parsedFile, content); err != nil {
        // 非致命错误,继续
    }

    // 4. 返回结果 (即使有部分错误)
    if parseErr != nil {
        return parsedFile, &DetailedParseError{
            File:    file.Path,
            Message: fmt.Sprintf("syntax error in Rust file: %v", parseErr),
            Type:    "parse",
        }
    }

    return parsedFile, nil
}

// 提取函数声明示例
func (p *RustParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
    query := `(function_item
        name: (identifier) @func.name) @func.def`

    matches, err := p.tsParser.Query(rootNode, query, "rust")
    if err != nil {
        return err
    }

    for _, match := range matches {
        var funcNode *sitter.Node
        var funcName string

        for _, capture := range match.Captures {
            if capture.Index == 0 { // func.name
                funcName = capture.Node.Content(content)
            } else if capture.Index == 1 { // func.def
                funcNode = capture.Node
            }
        }

        if funcNode != nil && funcName != "" {
            signature := p.extractSignature(funcNode, content)
            docstring := p.extractDocComment(funcNode, content)

            symbol := ParsedSymbol{
                Name:      funcName,
                Kind:      "function",
                Signature: signature,
                Span:      nodeToSpan(funcNode),
                Docstring: docstring,
                Node:      funcNode,
            }

            parsedFile.Symbols = append(parsedFile.Symbols, symbol)
        }
    }

    return nil
}

// 辅助方法
func (p *RustParser) extractSignature(node *sitter.Node, content []byte) string {
    // 提取函数签名
    nodeText := node.Content(content)
    lines := strings.Split(nodeText, "\n")

    // 只取第一行或到 { 为止
    for _, line := range lines {
        if strings.Contains(line, "{") {
            if idx := strings.Index(line, "{"); idx != -1 {
                return strings.TrimSpace(line[:idx])
            }
        }
        if strings.TrimSpace(line) != "" {
            return strings.TrimSpace(line)
        }
    }

    return strings.TrimSpace(nodeText)
}

func (p *RustParser) extractDocComment(node *sitter.Node, content []byte) string {
    // 提取 Rust 文档注释 (/// 或 /**/)
    // ... 实现细节
    return ""
}
```

### 步骤 5: 集成到 Parser Pool

在 `internal/parser/parser_pool.go` 的 `worker()` 方法中添加:

```go
func (p *ParserPool) worker(id int, jobs <-chan ParseJob, results chan<- ParseResult, wg *sync.WaitGroup) {
    // ... 初始化

    // 初始化 Rust parser
    rustParser := NewRustParser(workerTSParser)

    for job := range jobs {
        file := job.File

        var parsedFile *ParsedFile
        var parseErr error

        switch file.Language {
        // ... 现有 case
        case "Rust":
            parsedFile, parseErr = rustParser.Parse(file)
        default:
            parseErr = fmt.Errorf("unsupported language: %s", file.Language)
        }

        // ... 发送结果
    }
}
```

### 步骤 6: 编写测试

创建 `internal/parser/rust_parser_test.go`:

```go
package parser

import (
    "testing"
)

func TestRustParser_BasicExtraction(t *testing.T) {
    tests := []struct {
        name     string
        source   string
        wantFunc int // 期望提取的函数数量
        wantStruct int // 期望提取的结构体数量
    }{
        {
            name: "simple function",
            source: `
fn greet(name: &str) -> String {
    format!("Hello, {}!", name)
}
`,
            wantFunc: 1,
        },
        {
            name: "struct with impl",
            source: `
struct User {
    name: String,
    age: u32,
}

impl User {
    fn new(name: String, age: u32) -> Self {
        User { name, age }
    }
}
`,
            wantFunc: 1,
            wantStruct: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tsParser, err := NewTreeSitterParser()
            if err != nil {
                t.Fatalf("NewTreeSitterParser() error = %v", err)
            }

            parser := NewRustParser(tsParser)

            file := ScannedFile{
                Path:    "test.rs",
                AbsPath: "/tmp/test.rs",
                Language: "Rust",
            }

            parsedFile, err := parser.Parse(file)
            if err != nil {
                t.Fatalf("Parse() error = %v", err)
            }

            // 验证函数数量
            funcCount := 0
            for _, symbol := range parsedFile.Symbols {
                if symbol.Kind == "function" {
                    funcCount++
                }
            }
            if funcCount != tt.wantFunc {
                t.Errorf("Parse() extracted %d functions, want %d", funcCount, tt.wantFunc)
            }
        })
    }
}
```

### 步骤 7: 运行测试

```bash
# 运行单个测试文件
go test -v ./internal/parser -run TestRustParser

# 运行所有解析器测试
go test -v ./internal/parser

# 运行集成测试
go test -v ./tests/integration
```

## 跨语言调用检测

### 概述

跨语言调用检测允许 CodeAtlas 识别不同语言之间的互操作关系。这对于现代混合技术栈项目非常重要。

### 支持的跨语言关系

| 语言对 | 检测机制 | 检测率 |
|--------|----------|--------|
| Kotlin → Java | JVM 字节码互操作 | 100% |
| Swift → Objective-C | Objective-C 运行时桥接 | 100% |
| C++ → C | `extern "C"` 块 | 100% |
| TypeScript → JavaScript | ES 模块导入 | 62.5% |

### 实现跨语言检测

#### Kotlin → Java 示例

Kotlin 可以无缝调用 Java 代码。在解析器中需要检测 Java 类和方法调用:

```go
// internal/parser/kotlin_parser.go

func (p *KotlinParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
    // 1. 检测 Java 类型引用
    javaTypeQuery := `(user_type
        (type_identifier) @java.type)`

    matches, _ := p.tsParser.Query(rootNode, javaTypeQuery, "kotlin")

    for _, match := range matches {
        for _, capture := range match.Captures {
            typeName := capture.Node.Content(content)

            // 检查是否为 Java 类型 (通常首字母大写)
            if p.isJavaType(typeName) {
                caller := p.findContainingFunction(capture.Node, parsedFile)
                if caller != "" {
                    dependency := ParsedDependency{
                        Type:         "call",
                        Source:       caller,
                        Target:       typeName,
                        TargetModule: "java", // 标记为跨语言调用
                        IsExternal:   true,
                    }
                    parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
                }
            }
        }
    }

    return nil
}

func (p *KotlinParser) isJavaType(typeName string) bool {
    // 简单启发式: 检查常见 Java 包前缀
    javaPackages := []string{
        "java.", "javax.", "android.", "androidx.",
        "org.", "com.", "io.", "net.",
    }

    for _, prefix := range javaPackages {
        if strings.HasPrefix(typeName, prefix) {
            return true
        }
    }

    return false
}
```

#### Swift → Objective-C 示例

Swift 可以调用 Objective-C 代码。在解析器中检测 Objective-C 类和方法调用:

```go
// internal/parser/swift_parser.go

func (p *SwiftParser) extractCallRelationships(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
    // 1. 检测 Objective-C 类引用
    objcClassQuery := `(user_type
        (type_identifier) @objc.class)`

    matches, _ := p.tsParser.Query(rootNode, objcClassQuery, "swift")

    for _, match := range matches {
        for _, capture := range match.Captures {
            className := capture.Node.Content(content)

            // 检查是否为 Objective-C 类型 (通常以 NS/UI/CG 等前缀开头)
            if p.isObjCClass(className) {
                caller := p.findContainingFunction(capture.Node, parsedFile)
                if caller != "" {
                    dependency := ParsedDependency{
                        Type:         "call",
                        Source:       caller,
                        Target:       className,
                        TargetModule: "objc", // 标记为跨语言调用
                        IsExternal:   true,
                    }
                    parsedFile.Dependencies = append(parsedFile.Dependencies, dependency)
                }
            }
        }
    }

    return nil
}

func (p *SwiftParser) isObjCClass(className string) bool {
    // 检测常见 Objective-C 框架前缀
    objcPrefixes := []string{
        "NS", "UI", "CG", "CF", "CA", "CI", "CL",
        "MK", "AV", "WK", "SC", "GK", "SK",
    }

    for _, prefix := range objcPrefixes {
        if strings.HasPrefix(className, prefix) {
            return true
        }
    }

    return false
}
```

### 验证跨语言检测

编写测试验证跨语言调用检测:

```go
// internal/parser/kotlin_cross_language_test.go

func TestKotlinParser_JavaInterop(t *testing.T) {
    source := `
package com.example

import java.util.ArrayList
import java.io.File

fun processFile() {
    val list = ArrayList<String>()
    val file = File("/tmp/test.txt")
    file.readText()
}
`

    tsParser, _ := NewTreeSitterParser()
    parser := NewKotlinParser(tsParser)

    file := ScannedFile{
        Path:    "Test.kt",
        AbsPath: "/tmp/Test.kt",
        Language: "Kotlin",
    }

    parsedFile, _ := parser.Parse(file)

    // 验证检测到 Java 类型调用
    hasJavaCall := false
    for _, dep := range parsedFile.Dependencies {
        if dep.TargetModule == "java" {
            hasJavaCall = true
            break
        }
    }

    if !hasJavaCall {
        t.Error("Expected to detect Java interop calls")
    }
}
```

## Parser Pool 并发策略

### 工作池模式

Parser Pool 使用工作池模式实现并发解析:

```
┌──────────────┐     ┌──────────────┐
│   Worker 1   │     │   Worker 2   │
│  (独立实例)   │     │  (独立实例)   │
└──────┬───────┘     └──────┬───────┘
       │                     │
       ▼                     ▼
┌─────────────────────────────────┐
│       ParserPool                │
│  (任务分发 + 结果收集)            │
└─────────────────────────────────┘
```

### 关键特性

1. **每个 Worker 独立 Parser 实例**
   - Tree-sitter Parser **不是线程安全的**
   - 每个 worker 必须有自己的 parser 实例

2. **自适应 Worker 数量**

```go
func OptimalWorkerCount(fileCount int) int {
    cpus := runtime.NumCPU()

    // 小文件数量: 减少并发
    if fileCount < 10 {
        return min(2, cpus)
    }

    // 中等文件数量: 使用一半 CPU
    if fileCount < 50 {
        return min(cpus/2, cpus)
    }

    // 大量文件: 使用所有 CPU (最多 16)
    return min(cpus, 16)
}
```

3. **进度跟踪**

```go
type ProgressLogger interface {
    LogProgress(current, total int, file string)
    LogError(file string, err error)
}
```

### 使用示例

```go
// 创建 parser pool
tsParser, _ := NewTreeSitterParser()
pool := NewParserPool(4, tsParser) // 4 个 worker

// 设置进度日志
pool.SetVerbose(true)
pool.SetProgressLogger(&DefaultProgressLogger{})

// 并发解析
parsedFiles, errors := pool.Process(scannedFiles)

// 处理结果
for _, file := range parsedFiles {
    fmt.Printf("Parsed %s: %d symbols\n", file.Path, len(file.Symbols))
}
```

## 文件扫描和过滤规则

### FileScanner 功能

FileScanner 负责扫描文件系统并识别代码文件:

```go
type FileScanner struct {
    rootPath  string
    filter    *IgnoreFilter
    maxSize   int64    // 最大文件大小 (默认 1MB)
    languages []string // 语言过滤器 (空 = 所有语言)
}
```

### 过滤规则

#### 1. 忽略规则 (IgnoreFilter)

支持 `.gitignore` 风格的忽略规则:

```go
filter := NewIgnoreFilter()
filter.AddPatterns([]string{
    "node_modules/",
    "vendor/",
    "*.min.js",
    "*.pb.go", // protobuf 生成文件
})

scanner := NewFileScanner("/path/to/repo", filter)
```

#### 2. 文件大小限制

```go
scanner.SetMaxSize(1024 * 1024) // 1MB
```

#### 3. 二进制文件检测

自动跳过二进制文件:

```go
func isBinaryFile(path string) bool {
    binaryExtensions := map[string]bool{
        ".exe": true, ".dll": true, ".so": true,
        ".jpg": true, ".png": true, ".pdf": true,
        // ...
    }

    ext := strings.ToLower(filepath.Ext(path))
    return binaryExtensions[ext]
}
```

#### 4. 语言过滤

```go
// 只扫描 Go 和 Kotlin 文件
scanner.SetLanguageFilter([]string{"Go", "Kotlin"})
```

### 语言检测

#### 扩展名映射

```go
func determineLanguage(path string) string {
    ext := strings.ToLower(filepath.Ext(path))

    languageMap := map[string]string{
        ".go":  "Go",
        ".kt":  "Kotlin",
        ".kts": "Kotlin",
        ".swift": "Swift",
        ".java": "Java",
        // ...
    }

    if lang, exists := languageMap[ext]; exists {
        return lang
    }

    return "Unknown"
}
```

#### 特殊处理: .h 文件

`.h` 文件可能是 C、C++ 或 Objective-C。需要内容分析:

```go
func detectHeaderLanguage(path string) string {
    content, _ := readFileHeader(path, 4096)

    // 1. 检查 Objective-C 指示符
    if containsObjCIndicators(content) {
        return "Objective-C"
    }

    // 2. 检查 C++ 指示符
    if containsCppIndicators(content) {
        return "C++"
    }

    // 3. 默认为 C
    return "C"
}
```

## 测试指南

### 测试层级

```
tests/
├── unit/                    # 单元测试 (无外部依赖)
│   └── parser/
│       └── rust_parser_test.go
├── integration/             # 集成测试 (需要数据库)
│   └── cross_language_test.go
└── fixtures/                # 测试代码样例
    ├── rust/
    │   └── simple.rs
    └── kotlin-java/
        ├── Main.kt
        └── Helper.java
```

### 单元测试模式

使用表驱动测试:

```go
func TestRustParser_FunctionExtraction(t *testing.T) {
    tests := []struct {
        name       string
        source     string
        wantCount  int
        wantNames  []string
        wantError  bool
    }{
        {
            name: "single function",
            source: `fn hello() {}`,
            wantCount: 1,
            wantNames: []string{"hello"},
        },
        {
            name: "multiple functions",
            source: `
fn foo() {}
fn bar() {}
fn baz() {}
`,
            wantCount: 3,
            wantNames: []string{"foo", "bar", "baz"},
        },
        {
            name: "function with parameters",
            source: `fn add(x: i32, y: i32) -> i32 { x + y }`,
            wantCount: 1,
            wantNames: []string{"add"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 测试逻辑...
        })
    }
}
```

### 集成测试

验证完整索引流程:

```go
func TestCrossLanguage_KotlinJava(t *testing.T) {
    // 1. 准备测试文件
    repoPath := setupTestRepo(t, "kotlin-java")
    defer cleanupTestRepo(t, repoPath)

    // 2. 扫描文件
    scanner := NewFileScanner(repoPath, nil)
    files, _ := scanner.Scan()

    // 3. 解析文件
    tsParser, _ := NewTreeSitterParser()
    pool := NewParserPool(2, tsParser)
    parsedFiles, _ := pool.Process(files)

    // 4. 验证跨语言调用检测
    hasCrossLangCall := false
    for _, file := range parsedFiles {
        for _, dep := range file.Dependencies {
            if dep.TargetModule == "java" {
                hasCrossLangCall = true
            }
        }
    }

    if !hasCrossLangCall {
        t.Error("Expected to detect Kotlin-Java interop")
    }
}
```

### 性能测试

```go
func BenchmarkRustParser(b *testing.B) {
    tsParser, _ := NewTreeSitterParser()
    parser := NewRustParser(tsParser)

    source := generateLargeRustFile(1000) // 1000 个函数

    file := ScannedFile{
        Path:    "bench.rs",
        AbsPath: "/tmp/bench.rs",
        Language: "Rust",
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = parser.Parse(file)
    }
}
```

## 最佳实践

### 1. 错误处理

```go
// ✅ 好的做法: 非致命错误继续处理
if err := p.extractFunctions(rootNode, parsedFile, content); err != nil {
    // 记录错误但继续
}

// ❌ 不好的做法: 遇到错误立即返回
if err := p.extractFunctions(rootNode, parsedFile, content); err != nil {
    return err // 会丢失其他符号
}
```

### 2. 部分结果

即使解析失败,也返回部分结果:

```go
parsedFile := &ParsedFile{/* ... */}

// 尝试解析,失败时仍有部分结果
if parseErr != nil {
    return parsedFile, &DetailedParseError{
        File:    file.Path,
        Message: fmt.Sprintf("syntax error: %v", parseErr),
        Type:    "parse",
    }
}
```

### 3. 性能优化

```go
// ✅ 使用 Query 而非遍历
query := `(function_declaration name: (identifier) @func.name)`
matches, _ := p.tsParser.Query(rootNode, query, "rust")

// ❌ 避免手动遍历 AST
for i := 0; i < int(rootNode.ChildCount()); i++ {
    // 慢且容易出错
}
```

### 4. 内存管理

```go
// 清理查询对象
query, err := sitter.NewQuery([]byte(queryString), lang)
if err != nil {
    return nil, err
}
defer query.Close() // 重要!

cursor := sitter.NewQueryCursor()
defer cursor.Close() // 重要!
```

### 5. 测试覆盖

- **单元测试**: 覆盖所有 extract* 方法
- **集成测试**: 验证跨语言调用检测
- **基准测试**: 确保性能可接受
- **真实代码测试**: 使用开源项目测试

### 6. 文档注释

```go
// extractFunctions extracts top-level and public function declarations
// It captures function signatures including parameters and return types
//
// Query pattern:
//   (function_item
//     name: (identifier) @func.name) @func.def
//
// Returns:
//   - error if query execution fails (non-fatal)
func (p *RustParser) extractFunctions(rootNode *sitter.Node, parsedFile *ParsedFile, content []byte) error {
    // ...
}
```

## 常见问题

### Q: Tree-sitter 解析失败怎么办?

A: Tree-sitter 支持错误恢复,即使有语法错误也会返回部分 AST:

```go
rootNode, parseErr := p.tsParser.Parse(content, "rust")

// rootNode 可能为非 nil,即使有 parseErr
if rootNode != nil {
    // 继续提取符号,返回部分结果
}
```

### Q: 如何处理语言的歧义语法?

A: 使用上下文信息:

```go
// 示例: Rust 的 `foo()` 可能是宏调用或函数调用
// 检查父节点类型
if parent.Type() == "macro_invocation" {
    // 这是宏调用
} else if parent.Type() == "call_expression" {
    // 这是函数调用
}
```

### Q: 如何优化解析速度?

A: 几个建议:

1. **使用 Query 而非遍历**
2. **限制查询范围** - 只查询必要的节点
3. **批量处理** - Parser Pool 已经优化
4. **缓存结果** - 使用 checksum 避免重复解析

### Q: 如何支持新的文件扩展名?

A: 在 `scanner.go` 的 `determineLanguage()` 中添加:

```go
languageMap := map[string]string{
    // ... 现有映射
    ".rs": "Rust",
}
```

## 参考资源

### Tree-sitter 文档

- [Tree-sitter 官方文档](https://tree-sitter.github.io/tree-sitter/)
- [查询语言语法](https://tree-sitter.github.io/tree-sitter/using-parsers#query-syntax)
- [Go 绑定文档](https://github.com/smacker/go-tree-sitter)

### 项目参考

- [Go 解析器](/internal/parser/go_parser.go) - 最完整的参考实现
- [Kotlin 解析器](/internal/parser/kotlin_parser.go) - 跨语言检测示例
- [Swift 解析器](/internal/parser/swift_parser.go) - 复杂语言特性处理

### 测试资源

- [解析器测试](/internal/parser/*_test.go)
- [跨语言测试](/internal/parser/*_cross_language_test.go)
- [集成测试](/tests/integration/call_analysis_test.go)

## 下一步

1. **选择目标语言** - 确定要添加的语言
2. **查找语法库** - 确认 Tree-sitter 语法库可用性
3. **实现基础解析** - 按照步骤 4-6 实现解析器
4. **编写测试** - 确保测试覆盖率 >= 90%
5. **验证跨语言调用** - 如果适用,实现跨语言检测
6. **性能优化** - 运行基准测试,优化性能

## 贡献指南

提交解析器实现时,请确保:

- [ ] 通过所有单元测试 (`make test`)
- [ ] 通过集成测试 (`make test-integration`)
- [ ] 测试覆盖率 >= 90% (`make test-coverage`)
- [ ] 添加测试用例到 `tests/fixtures/`
- [ ] 更新本文档,添加语言特定说明
- [ ] 运行完整验证 (`make verify`)

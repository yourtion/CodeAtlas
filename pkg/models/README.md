# 数据模型层文档

本文档介绍 CodeAtlas 数据模型层的设计和实现。

## 目录

- [概述](#概述)
- [数据库模型](#数据库模型)
- [仓库模式](#仓库模式)
- [事务管理](#事务管理)
- [数据库连接池](#数据库连接池)
- [pgvector 集成](#pgvector-集成)
- [数据迁移](#数据迁移)
- [性能优化](#性能优化)

---

## 概述

数据模型层（`pkg/models/`）提供数据库访问抽象，封装所有数据库操作。

### 技术栈

- **数据库**: PostgreSQL 15+
- **向量扩展**: pgvector
- **驱动**: pgx (PostgreSQL 驱动)

### 设计原则

1. **单一职责** - 每个仓库只负责一种实体
2. **依赖注入** - 通过构造函数注入依赖
3. **错误处理** - 明确的错误类型和错误传播
4. **事务安全** - 支持事务操作
5. **测试友好** - 易于单元测试和集成测试

---

## 数据库模型

### ER 图

```
┌─────────────────┐
│  repositories   │
├─────────────────┤
│ id (PK)         │───┐
│ name            │   │
│ url             │   │
│ branch          │   │
│ commit_hash     │   │
│ created_at      │   │
│ updated_at      │   │
└─────────────────┘   │
                      │
                      │     ┌─────────────────┐
                      │     │     files       │
                      │     ├─────────────────┤
                      └────││ id (PK)         │
                           │ repo_id (FK)     │
                           │ path             │
                           │ language         │
                           │ checksum         │
                           │ content          │
                           └────────┬─────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
            ┌───────▼──────┐ ┌─────▼─────┐ ┌─────▼─────┐
            │   symbols    │ │ ast_nodes │ │  edges    │
            ├──────────────┤ ├───────────┤ ├───────────┤
            │ id (PK)      │ │ id (PK)   │ │ id (PK)   │
            │ file_id (FK) │ │ symbol_id ││ source_id │
            │ name         │ │ node_data ││ target_id │
            │ kind         │ │           │ │ edge_type │
            │ signature    │ └───────────┘ └───────────┘
            │ docstring    │
            └──────────────┘
                    │
            ┌───────▼──────┐
            │   vectors    │
            ├──────────────┤
            │ symbol_id(PK)│
            │ embedding    │
            │ dimensions   │
            └──────────────┘
```

### Repository 模型

```go
// repository.go
type Repository struct {
    ID          string    `db:"id"`
    Name        string    `db:"name"`
    URL         string    `db:"url"`
    Branch      string    `db:"branch"`
    CommitHash  string    `db:"commit_hash"`
    CreatedAt   time.Time `db:"created_at"`
    UpdatedAt   time.Time `db:"updated_at"`
}
```

**字段说明**：
- `id` - UUID 主键
- `name` - 仓库名称
- `url` - Git 仓库 URL
- `branch` - 分支名称（默认 main）
- `commit_hash` - 最后一次索引的 commit hash
- `created_at` - 创建时间
- `updated_at` - 更新时间

### File 模型

```go
// file.go
type File struct {
    ID        string    `db:"id"`
    RepoID    string    `db:"repo_id"`
    Path      string    `db:"path"`
    Language  string    `db:"language"`
    Checksum  string    `db:"checksum"`
    Content   []byte    `db:"content"`
    Size      int64     `db:"size"`
    CreatedAt time.Time `db:"created_at"`
    UpdatedAt time.Time `db:"updated_at"`
}
```

**字段说明**：
- `id` - UUID 主键
- `repo_id` - 外键关联到 repositories
- `path` - 相对路径
- `language` - 编程语言
- `checksum` - SHA256 校验和（用于增量索引）
- `content` - 文件内容
- `size` - 文件大小（字节）
- `created_at` - 创建时间
- `updated_at` - 更新时间

### Symbol 模型

```go
// symbol.go
type Symbol struct {
    ID         string    `db:"id"`
    FileID     string    `db:"file_id"`
    Name       string    `db:"name"`
    Kind       string    `db:"kind"`
    Signature  string    `db:"signature"`
    Docstring  string    `db:"docstring"`
    StartLine  int       `db:"start_line"`
    EndLine    int       `db:"end_line"`
    StartByte  int       `db:"start_byte"`
    EndByte    int       `db:"end_byte"`
    CreatedAt  time.Time `db:"created_at"`
    UpdatedAt  time.Time `db:"updated_at"`
}
```

**字段说明**：
- `id` - UUID 主键
- `file_id` - 外键关联到 files
- `name` - 符号名称
- `kind` - 符号类型（function, class, method, variable 等）
- `signature` - 函数签名或类型签名
- `docstring` - 文档注释
- `start_line` - 起始行号（1-based）
- `end_line` - 结束行号
- `start_byte` - 起始字节位置
- `end_byte` - 结束字节位置

### ASTNode 模型

```go
// ast_node.go
type ASTNode struct {
    ID         string    `db:"id"`
    SymbolID   string    `db:"symbol_id"`
    NodeType   string    `db:"node_type"`
    NodeData   string    `db:"node_data"`
    ParentID   *string   `db:"parent_id"`
    CreatedAt  time.Time `db:"created_at"`
}
```

**字段说明**：
- `id` - UUID 主键
- `symbol_id` - 关联的符号 ID
- `node_type` - 节点类型（对应 Tree-sitter 节点类型）
- `node_data` - 节点数据（JSON 格式）
- `parent_id` - 父节点 ID（可选）

### Edge 模型

```go
// edge.go
type Edge struct {
    ID           string    `db:"id"`
    SourceID     string    `db:"source_id"`
    TargetID     string    `db:"target_id"`
    EdgeType     string    `db:"edge_type"`
    Metadata     string    `db:"metadata"`
    CreatedAt    time.Time `db:"created_at"`
}
```

**字段说明**：
- `id` - UUID 主键
- `source_id` - 源符号 ID
- `target_id` - 目标符号 ID
- `edge_type` - 边类型（call, import, extends, implements 等）
- `metadata` - 额外元数据（JSON 格式）

### Vector 模型

```go
// vector.go
type Vector struct {
    SymbolID   string      `db:"symbol_id"`
    Embedding  []float32   `db:"embedding"`
    Dimensions int         `db:"dimensions"`
    UpdatedAt  time.Time   `db:"updated_at"`
}
```

**字段说明**：
- `symbol_id` - 符号 ID（主键）
- `embedding` - 向量嵌入（pgvector 类型）
- `dimensions` - 向量维度
- `updated_at` - 更新时间

---

## 仓库模式

### RepositoryRepository

```go
type RepositoryRepository struct {
    db *DB
}

func NewRepositoryRepository(db *DB) *RepositoryRepository {
    return &RepositoryRepository{db: db}
}

// Create 创建新仓库
func (r *RepositoryRepository) Create(ctx context.Context, repo *Repository) error

// GetByID 根据 ID 获取仓库
func (r *RepositoryRepository) GetByID(ctx context.Context, id string) (*Repository, error)

// GetByName 根据名称获取仓库
func (r *RepositoryRepository) GetByName(ctx context.Context, name string) (*Repository, error)

// GetAll 获取所有仓库
func (r *RepositoryRepository) GetAll(ctx context.Context) ([]*Repository, error)

// Update 更新仓库
func (r *RepositoryRepository) Update(ctx context.Context, repo *Repository) error

// Delete 删除仓库
func (r *RepositoryRepository) Delete(ctx context.Context, id string) error
```

### FileRepository

```go
type FileRepository struct {
    db *DB
}

func NewFileRepository(db *DB) *FileRepository {
    return &FileRepository{db: db}
}

// CreateBatch 批量创建文件
func (r *FileRepository) CreateBatch(ctx context.Context, files []*File) error

// GetByID 获取文件
func (r *FileRepository) GetByID(ctx context.Context, id string) (*File, error)

// GetByRepoID 获取仓库的所有文件
func (r *FileRepository) GetByRepoID(ctx context.Context, repoID string) ([]*File, error)

// GetByChecksum 根据校验和获取文件（用于增量索引）
func (r *FileRepository) GetByChecksum(ctx context.Context, repoID string, checksum string) (*File, error)

// Update 更新文件
func (r *FileRepository) Update(ctx context.Context, file *File) error

// DeleteByRepoID 删除仓库的所有文件
func (r *FileRepository) DeleteByRepoID(ctx context.Context, repoID string) error
```

### SymbolRepository

```go
type SymbolRepository struct {
    db *DB
}

func NewSymbolRepository(db *DB) *SymbolRepository {
    return &SymbolRepository{db: db}
}

// CreateBatch 批量创建符号
func (r *SymbolRepository) CreateBatch(ctx context.Context, symbols []*Symbol) error

// GetByID 获取符号
func (r *SymbolRepository) GetByID(ctx context.Context, id string) (*Symbol, error)

// GetByFileID 获取文件的所有符号
func (r *SymbolRepository) GetByFileID(ctx context.Context, fileID string) ([]*Symbol, error)

// SearchByName 根据名称搜索符号
func (r *SymbolRepository) SearchByName(ctx context.Context, name string) ([]*Symbol, error)

// GetByKind 根据类型获取符号
func (r *SymbolRepository) GetByKind(ctx context.Context, kind string) ([]*Symbol, error)
```

### EdgeRepository

```go
type EdgeRepository struct {
    db *DB
}

func NewEdgeRepository(db *DB) *EdgeRepository {
    return &EdgeRepository{db: db}
}

// CreateBatch 批量创建边
func (r *EdgeRepository) CreateBatch(ctx context.Context, edges []*Edge) error

// GetBySourceID 获取源符号的所有出边
func (r *EdgeRepository) GetBySourceID(ctx context.Context, sourceID string) ([]*Edge, error)

// GetByTargetID 获取目标符号的所有入边
func (r *EdgeRepository) GetByTargetID(ctx context.Context, targetID string) ([]*Edge, error)

// GetByType 根据类型获取边
func (r *EdgeRepository) GetByType(ctx context.Context, edgeType string) ([]*Edge, error)

// DeleteBySymbolID 删除符号相关的所有边
func (r *EdgeRepository) DeleteBySymbolID(ctx context.Context, symbolID string) error
```

### VectorRepository

```go
type VectorRepository struct {
    db *DB
}

func NewVectorRepository(db *DB) *VectorRepository {
    return &VectorRepository{db: db}
}

// Create 创建向量
func (r *VectorRepository) Create(ctx context.Context, vector *Vector) error

// CreateBatch 批量创建向量
func (r *VectorRepository) CreateBatch(ctx context.Context, vectors []*Vector) error

// Get 获取向量
func (r *VectorRepository) Get(ctx context.Context, symbolID string) (*Vector, error)

// Update 更新向量
func (r *VectorRepository) Update(ctx context.Context, vector *Vector) error

// Delete 删除向量
func (r *VectorRepository) Delete(ctx context.Context, symbolID string) error

// SemanticSearch 语义搜索（向量相似度搜索）
func (r *VectorRepository) SemanticSearch(
    ctx context.Context,
    queryVector []float32,
    limit int,
    threshold float32,
) ([]*SearchResult, error)
```

---

## 事务管理

### 事务

代码库使用 `database/sql` 原生事务（`*sql.Tx`），各 repository 提供
`BatchCreateTx(ctx, tx, items)` 形态的方法，在调用方管理的事务边界内写入，
保证索引管道多表写入的原子性。

```go
// 写入方示例（见 internal/indexer/indexer.go 的 writeDataWithTransaction）
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
// 失败时回滚（命名返回值 + defer 是惯用模式）
defer func() {
    if err != nil {
        _ = tx.Rollback()
    }
}()

if err := fileRepo.BatchCreateTx(ctx, tx, files); err != nil {
    return err
}
if err := symbolRepo.BatchCreateTx(ctx, tx, symbols); err != nil {
    return err
}
if err := edgeRepo.BatchCreateTx(ctx, tx, edges); err != nil {
    return err
}
return tx.Commit()
```

> 注：早期版本曾有独立的 `TransactionManager`/`WithTransaction` 抽象（基于
> pgx 风格的 `Tx`/`TxOptions` 接口），已在死代码清理中移除——实际写入路径
> 一律走 `*sql.Tx`，不再保留这层未接入的抽象。

---

## 数据库连接池

### 配置

```go
// db.go
type Config struct {
    Host            string
    Port            int
    User            string
    Password        string
    Database        string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
    ConnMaxIdleTime time.Duration
}

// 连接池配置
poolConfig, err := pgxpool.ParseConfig(fmt.Sprintf(
    "host=%s port=%d user=%s password=%s dbname=%s",
    config.Host,
    config.Port,
    config.User,
    config.Password,
    config.Database,
))

poolConfig.MaxConns = int32(config.MaxOpenConns)
poolConfig.MinConns = int32(config.MaxIdleConns)
poolConfig.MaxConnLifetime = config.ConnMaxLifetime
poolConfig.MaxConnIdleTime = config.ConnMaxIdleTime
```

### 推荐配置

```go
// 生产环境
config := &Config{
    MaxOpenConns:    25,  // 最大连接数
    MaxIdleConns:    5,   // 最小空闲连接数
    ConnMaxLifetime: 5 * time.Minute,  // 连接最大生命周期
    ConnMaxIdleTime: 1 * time.Minute,  // 空闲连接最大存活时间
}

// 开发环境
config := &Config{
    MaxOpenConns:    10,
    MaxIdleConns:    2,
    ConnMaxLifetime: 10 * time.Minute,
    ConnMaxIdleTime: 5 * time.Minute,
}
```

### 连接池监控

```go
// 获取连接池统计
stats := db.Pool.Stat()
fmt.Printf("Total connections: %d\n", stats.TotalConns())
fmt.Printf("Idle connections: %d\n", stats.IdleConns())
fmt.Printf("Acquired connections: %d\n", stats.AcquiredConns())
```

---

## pgvector 集成

### 安装 pgvector

```sql
-- 安装扩展
CREATE EXTENSION IF NOT EXISTS vector;

-- 创建向量列
CREATE TABLE vectors (
    symbol_id UUID PRIMARY KEY,
    embedding vector(1536),
    dimensions INTEGER,
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### 向量操作

```go
// vector.go
type Vector struct {
    SymbolID   string
    Embedding  []float32
    Dimensions int
}

// 插入向量
func (r *VectorRepository) Create(ctx context.Context, vector *Vector) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO vectors (symbol_id, embedding, dimensions)
        VALUES ($1, $2, $3)
        ON CONFLICT (symbol_id) DO UPDATE
        SET embedding = $2, updated_at = NOW()
    `, vector.SymbolID, pgvector.NewVector(vector.Embedding, vector.Dimensions), vector.Dimensions)
    return err
}

// 语义搜索
func (r *VectorRepository) SemanticSearch(
    ctx context.Context,
    queryVector []float32,
    limit int,
    threshold float32,
) ([]*SearchResult, error) {
    rows, err := r.db.Query(ctx, `
        SELECT
            s.id,
            s.name,
            s.kind,
            s.signature,
            f.path,
            v.embedding <=> $1 as distance
        FROM vectors v
        JOIN symbols s ON s.id = v.symbol_id
        JOIN files f ON f.id = s.file_id
        WHERE v.embedding <=> $1 < $2
        ORDER BY v.embedding <=> $1
        LIMIT $3
    `, pgvector.NewVector(queryVector, len(queryVector)), threshold, limit)
    // ...
}
```

### 距离函数

pgvector 提供多种距离函数：

| 函数 | 说明 | 用途 |
|------|------|------|
| `<=>` | 负欧几里得距离 | 最常用 |
| `<->` | 欧几里得距离 | 标准距离 |
| `<#>` | 负内积 | 归一化向量 |
| `<=>` | 余弦距离 | 角度相似度 |

---

## 代码图谱查询

> 注：已移除 Apache AGE 图数据库支持，改用关系表（edges/symbols/files）的 SQL 查询实现代码知识图谱。

代码关系（调用、依赖、继承等）存储在 `edges` 表中，通过标准 SQL JOIN 查询即可实现调用图、依赖链和路径查询：

```go
// 查询调用关系
func (r *EdgeRepository) GetCallers(ctx context.Context, symbolID string) ([]*Symbol, error) {
    rows, err := r.db.Query(ctx, `
        SELECT s.id, s.name, s.kind
        FROM edges e
        JOIN symbols s ON s.id = e.source_id
        WHERE e.target_id = $1 AND e.edge_type = 'call'
    `, symbolID)
    // ...
}
```

---

## 数据迁移

### Schema 版本管理

```go
// schema.go
type SchemaManager struct {
    db *DB
}

func (m *SchemaManager) EnsureLatestSchema(ctx context.Context) error {
    version, err := m.getCurrentVersion(ctx)
    if err != nil {
        return err
    }

    for _, migration := range migrations {
        if migration.Version > version {
            if err := migration.Up(ctx, m.db); err != nil {
                return fmt.Errorf("migration %d failed: %w", migration.Version, err)
            }
        }
    }

    return nil
}
```

### 迁移示例

```go
var migrations = []Migration{
    {
        Version: 1,
        Name:    "initial_schema",
        Up: func(ctx context.Context, db *DB) error {
            _, err := db.Exec(ctx, `
                CREATE TABLE IF NOT EXISTS repositories (
                    id UUID PRIMARY KEY,
                    name TEXT NOT NULL UNIQUE,
                    url TEXT,
                    branch TEXT DEFAULT 'main',
                    commit_hash TEXT,
                    created_at TIMESTAMP DEFAULT NOW(),
                    updated_at TIMESTAMP DEFAULT NOW()
                )
            `)
            return err
        },
    },
    {
        Version: 2,
        Name:    "add_vectors_table",
        Up: func(ctx context.Context, db *DB) error {
            _, err := db.Exec(ctx, `
                CREATE EXTENSION IF NOT EXISTS vector;

                CREATE TABLE IF NOT EXISTS vectors (
                    symbol_id UUID PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
                    embedding vector(1536),
                    dimensions INTEGER,
                    updated_at TIMESTAMP DEFAULT NOW()
                )
            `)
            return err
        },
    },
}
```

---

## 性能优化

### 1. 批量插入

```go
// 使用 COPY 命令批量插入
func (r *FileRepository) CreateBatch(ctx context.Context, files []*File) error {
    _, err := r.db.CopyFrom(
        ctx,
        pgx.Identifier{"files"},
        []string{"id", "repo_id", "path", "language", "checksum", "content"},
        pgx.CopyFromSlice(len(files), func(i int) ([]interface{}, error) {
            return []interface{}{
                files[i].ID,
                files[i].RepoID,
                files[i].Path,
                files[i].Language,
                files[i].Checksum,
                files[i].Content,
            }, nil
        }),
    )
    return err
}
```

### 2. 索引优化

```sql
-- 创建常用查询的索引
CREATE INDEX idx_files_repo_id ON files(repo_id);
CREATE INDEX idx_files_checksum ON files(checksum);
CREATE INDEX idx_symbols_file_id ON symbols(file_id);
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_kind ON symbols(kind);
CREATE INDEX idx_edges_source_id ON edges(source_id);
CREATE INDEX idx_edges_target_id ON edges(target_id);
CREATE INDEX idx_edges_type ON edges(edge_type);

-- 向量相似度索引（HNSW）
CREATE INDEX ON vectors USING hnsw (embedding vector_cosine_ops);
```

### 3. 查询优化

```go
// 使用预编译语句
func (r *SymbolRepository) GetByFileID(ctx context.Context, fileID string) ([]*Symbol, error) {
    rows, _ := r.db.Query(ctx, `
        SELECT id, file_id, name, kind, signature
        FROM symbols
        WHERE file_id = $1
    `, fileID)
    // ...
}

// 使用 JSON 聚合减少查询次数
func (r *FileRepository) GetWithSymbols(ctx context.Context, repoID string) ([]*FileWithSymbols, error) {
    rows, err := r.db.Query(ctx, `
        SELECT
            f.*,
            COALESCE(
                json_agg(
                    json_build_object(
                        'id', s.id,
                        'name', s.name,
                        'kind', s.kind
                    )
                ) FILTER (WHERE s.id IS NOT NULL),
                '[]'
            ) as symbols
        FROM files f
        LEFT JOIN symbols s ON s.file_id = f.id
        WHERE f.repo_id = $1
        GROUP BY f.id
    `, repoID)
    // ...
}
```

### 4. 连接池调优

```go
// 根据负载调整连接池
config := &Config{
    MaxOpenConns: 4 * runtime.NumCPU(),  // CPU 核心数的 4 倍
    MaxIdleConns: runtime.NumCPU(),
}
```

---

## 相关文档

- [数据库模式](../../docs/schema.md)
- [API 服务架构](../api/README.md)
- [配置指南](../../docs/configuration.md)
- [pgvector 文档](https://github.com/pgvector/pgvector)

---

**最后更新**: 2026-02-10
**维护者**: CodeAtlas 团队

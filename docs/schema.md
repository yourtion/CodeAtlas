# CodeAtlas Unified Schema Specification (Draft v0.1)

## 🎯 目标 (Objectives)

- 提供一个 统一的存储结构，覆盖项目、代码结构、知识增强三层。
- 同时支持 关系型查询 (PostgreSQL)、向量检索 (pgvector)、图遍历 (AGE)。
- 支持 增量更新（基于 git diff 和 tree-sitter），避免全量重算。
- 为 RAG 检索 和 语义/结构混合查询 提供支撑。
- 跨层次追溯：从 仓库 → 文件 → 符号 → AST Node → Token。


## 📦 存储层次 (Storage Layers)

### 1. 项目层 (Repository / Project)

- Repository
- repo_id (PK)
- name
- url
- branch
- commit_hash
- metadata (jsonb)
- Dependency
- dep_id (PK)
- repo_id (FK)
- package_name
- version
- source (registry / git / local)

（说明：Git 提交历史仅存储摘要映射，不存全量 diff，以降低成本）

### 2. 代码结构层 (Code Structure)

- File
- file_id (PK)
- repo_id (FK)
- path
- language
- checksum
- Symbol
- symbol_id (PK)
- file_id (FK)
- name
- kind (function / class / interface / variable / package / module)
- signature
- span (start_line, end_line)
- AST Node
- node_id (PK, 全局唯一锚点)
- file_id (FK)
- type (tree-sitter node type)
- span
- parent_id (FK self)
- extra (jsonb: attributes)
- Dependency Edge (Graph)
- edge_id (PK)
- src_symbol_id (FK)
- dst_symbol_id (FK)
- edge_type (import / call / extend / implement / reference)

### 3. 知识增强层 (Knowledge Layer)

- Docstring
- doc_id (PK)
- symbol_id (FK)
- content
- Embedding
- embed_id (PK)
- node_id (FK)
- embedding (vector)
- content (text for re-ranking/debugging)
- chunk_index
- Summary
- summary_id (PK)
- node_id (FK)
- summary_type (llm / prs / manual)
- content
- Graph (AGE)
- 节点 (node_id 对应 File / Symbol / AST Node)
- 边 (调用链、继承关系、跨文件引用)
- 类型 (CALL, IMPORT, EXTENDS, IMPLEMENTS, USES)

## 🔗 跨表锚点 (Cross-Layer Anchor)

- 所有实体（file_id / symbol_id / node_id）会映射到一个 全局唯一 node_id：
- 在 关系表 中：存元数据
- 在 向量表 中：存嵌入
- 在 图表 中：作为顶点引用

## 🛠 更新与增量 (Incremental Updates)

- 文件级别更新：通过 git diff 确认修改文件。
- AST 增量解析：只对修改过的文件调用 tree-sitter。
- 缓存利用：
- 文件 checksum 确保未变文件可复用
- 符号级别缓存（根据 span 和 signature）
- 向量/图更新：
- 当 node_id 变更时，触发对应 embedding & graph 更新。

## 🚀 查询场景 (RAG / Queries)

1.	语义检索：通过 pgvector 在 embedding 表中查找相关代码/注释。
2.	结构查询：通过 SQL 查询符号、文件、AST 元数据。
3.	图检索：通过 AGE 遍历调用链/依赖关系。
4.	混合查询：先向量召回，后通过 graph 过滤（例如 “找出调用该函数的上层模块”）。

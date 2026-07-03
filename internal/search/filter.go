// Package search 封装代码语义检索的过滤与排序逻辑。
//
// 本包现阶段为检索质量主线的接口预留层：将原本散落在 search_handler
// 内的内存过滤逻辑集中为可替换的组件，便于下一站把过滤下沉到 SQL 层
// （解决"先按 limit 取向量再过滤，过滤掉的不补位，导致结果数失真"），
// 而无需改动 handler 调用点。
package search

// ResultFilter 定义检索结果的过滤策略。
//
// 当前唯一实现 InMemoryFilter 在内存中按 kind/language/repo_id 过滤；
// 下一站可新增 SQLFilter，在向量检索时通过 JOIN/WHERE 直接过滤，
// 实现"过滤后仍满 limit"的正确语义。handler 通过 Filter() 调用，
// 不感知具体实现。
type ResultFilter interface {
	// Filter 对候选结果应用过滤，返回满足条件的结果。
	// 实现应保持输入顺序（按相似度降序）。
	Filter(candidates []Candidate) []Candidate
}

// Candidate 是过滤管道中的中间结果，融合了向量检索结果与符号/文件详情。
// 内存过滤实现只读此结构的字段；SQL 过滤实现会在检索阶段就应用条件，
// 构造 Candidate 时已满足过滤。
type Candidate struct {
	SymbolID   string
	Name       string
	Kind       string
	Signature  string
	FilePath   string
	Language   string
	RepoID     string
	Docstring  string
	Similarity float64
}

// FilterCriteria 是过滤条件。空字段表示不过滤该维度。
type FilterCriteria struct {
	Kind     []string // 任一匹配即保留（OR 语义）
	Language string   // 精确匹配
	RepoID   string   // 精确匹配
}

// InMemoryFilter 在内存中按 FilterCriteria 过滤候选结果。
// 这是从 search_handler 抽出的原有逻辑，行为保持不变。
type InMemoryFilter struct {
	Criteria FilterCriteria
}

// NewInMemoryFilter 创建一个内存过滤器。
func NewInMemoryFilter(criteria FilterCriteria) *InMemoryFilter {
	return &InMemoryFilter{Criteria: criteria}
}

// Filter 按相似度保序过滤候选结果。
func (f *InMemoryFilter) Filter(candidates []Candidate) []Candidate {
	out := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if !f.Criteria.matches(c) {
			continue
		}
		out = append(out, c)
	}
	return out
}

// matches 检查候选结果是否满足所有过滤维度。
func (fc FilterCriteria) matches(c Candidate) bool {
	// kind: 任一匹配即保留（空列表表示不过滤）
	if len(fc.Kind) > 0 {
		kindMatch := false
		for _, k := range fc.Kind {
			if c.Kind == k {
				kindMatch = true
				break
			}
		}
		if !kindMatch {
			return false
		}
	}

	// language: 精确匹配
	if fc.Language != "" && c.Language != fc.Language {
		return false
	}

	// repo_id: 精确匹配
	if fc.RepoID != "" && c.RepoID != fc.RepoID {
		return false
	}

	return true
}

package retrieval

import (
	"context"
	"reflect"
	"testing"

	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// fakeVectorSearcher 实现 VectorSearcher 接口，供单测注入。
//
// 仅 keyword 路径在 keyword 测试用例里被走到；vector/hybrid 路径返回 error，
// 避免误用未设置的结果时静默通过。
//
// receivedFilters 记录最近一次被调用时收到的 filters（拷贝），用于断言
// RepoIDs 等参数是否被 Query 正确透传。
type fakeVectorSearcher struct {
	keywordResults  []*models.VectorSearchResult
	receivedFilters models.VectorSearchFilters
}

func (f *fakeVectorSearcher) KeywordSearch(ctx context.Context, query string, filters models.VectorSearchFilters) ([]*models.VectorSearchResult, error) {
	f.receivedFilters = filters
	// 拷贝 RepoIDs 切片，避免被调用方后续就地修改污染断言。
	if filters.RepoIDs != nil {
		f.receivedFilters.RepoIDs = append([]string(nil), filters.RepoIDs...)
	}
	return f.keywordResults, nil
}

func (f *fakeVectorSearcher) HybridSearch(ctx context.Context, query string, emb []float32, filters models.VectorSearchFilters, wv, wk float64) ([]*models.HybridSearchResult, error) {
	panic("fakeVectorSearcher.HybridSearch should not be called in these tests")
}

func (f *fakeVectorSearcher) SimilaritySearchWithFilters(ctx context.Context, emb []float32, filters models.VectorSearchFilters) ([]*models.VectorSearchResult, error) {
	panic("fakeVectorSearcher.SimilaritySearchWithFilters should not be called in these tests")
}

func (f *fakeVectorSearcher) GetByVectorIDs(ctx context.Context, ids []string) ([]*models.Vector, error) {
	panic("fakeVectorSearcher.GetByVectorIDs should not be called in these tests")
}

// 编译期断言：确保 fakeVectorSearcher 满足 VectorSearcher 接口。
// 一旦接口签名变更导致不再满足，编译立即失败。
var _ VectorSearcher = (*fakeVectorSearcher)(nil)

// fakeEdgeExpander 实现 EdgeExpander 接口，用 map 存预设的 callers/callees。
type fakeEdgeExpander struct {
	callers map[string][]*models.EdgeWithDetails // key = symbolID
	callees map[string][]*models.EdgeWithDetails // key = symbolID
}

func (e *fakeEdgeExpander) GetCallersWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error) {
	return e.callers[symbolID], nil
}

func (e *fakeEdgeExpander) GetCalleesWithDetails(ctx context.Context, symbolID string) ([]*models.EdgeWithDetails, error) {
	return e.callees[symbolID], nil
}

// 编译期断言。
var _ EdgeExpander = (*fakeEdgeExpander)(nil)

// makeKeywordHit 构造一个关键词检索命中结果。
func makeKeywordHit(symbolID, name string, sim float64) *models.VectorSearchResult {
	return &models.VectorSearchResult{
		VectorID:   "vec-" + symbolID,
		EntityID:   symbolID,
		EntityType: "symbol",
		Name:       name,
		Kind:       "function",
		Similarity: sim,
	}
}

// makeEdgeDetail 构造一条带详情的邻居边。
func makeEdgeDetail(symbolID, name string) *models.EdgeWithDetails {
	return &models.EdgeWithDetails{
		EdgeType: "calls",
		SymbolID: symbolID,
		Name:     name,
		Kind:     "function",
	}
}

// TestHybridRetriever_Query_RepoIDsFilterPassedThrough 验证 Query 把
// RetrievalRequest.RepoIDs 透传到 VectorSearcher 收到的 filters 中。
//
// 用 keyword 模式：该路径不调用 embedder，故 embedder 可传 nil。
func TestHybridRetriever_Query_RepoIDsFilterPassedThrough(t *testing.T) {
	vs := &fakeVectorSearcher{
		keywordResults: []*models.VectorSearchResult{
			makeKeywordHit("sym-1", "DoThing", 0.5),
		},
	}
	r := NewHybridRetriever(vs, &fakeEdgeExpander{}, nil, DefaultHybridRetrieverConfig())

	blocks, err := r.Query(context.Background(), RetrievalRequest{
		Query:   "do thing",
		RepoIDs: []string{"repo-a", "repo-b"},
		Mode:    "keyword",
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	// 断言 RepoIDs 被透传且长度正确。
	if got := len(vs.receivedFilters.RepoIDs); got != 2 {
		t.Fatalf("expected receivedFilters.RepoIDs len 2, got %d", got)
	}
	// 断言 RepoIDs 内容被原样透传（顺序、值均一致）。
	if !reflect.DeepEqual(vs.receivedFilters.RepoIDs, []string{"repo-a", "repo-b"}) {
		t.Errorf("expected receivedFilters.RepoIDs [repo-a repo-b], got %v", vs.receivedFilters.RepoIDs)
	}
	// 断言其它过滤字段也被正确下沉。
	if vs.receivedFilters.EntityType != "symbol" {
		t.Errorf("expected EntityType=symbol, got %q", vs.receivedFilters.EntityType)
	}
	if !vs.receivedFilters.WithDetails {
		t.Error("expected WithDetails=true")
	}
}

// TestHybridRetriever_NeighborLimitTop5 验证 callers/callees 邻居被限流到 Top-5，
// 且按符号名字典序排序后截断。
//
// 构造 8 个 callers，名字乱序输入以同时验证排序与截断。
func TestHybridRetriever_NeighborLimitTop5(t *testing.T) {
	const hitSymbol = "sym-1"
	edges := &fakeEdgeExpander{
		callers: map[string][]*models.EdgeWithDetails{
			// 故意乱序：字典序升序应为 callA..callH，Top-5 = callA..callE。
			hitSymbol: {
				makeEdgeDetail("c8", "callH"),
				makeEdgeDetail("c1", "callA"),
				makeEdgeDetail("c5", "callE"),
				makeEdgeDetail("c3", "callC"),
				makeEdgeDetail("c7", "callG"),
				makeEdgeDetail("c2", "callB"),
				makeEdgeDetail("c6", "callF"),
				makeEdgeDetail("c4", "callD"),
			},
		},
	}
	vs := &fakeVectorSearcher{
		keywordResults: []*models.VectorSearchResult{
			makeKeywordHit(hitSymbol, "DoThing", 0.5),
		},
	}
	r := NewHybridRetriever(vs, edges, nil, DefaultHybridRetrieverConfig())

	blocks, err := r.Query(context.Background(), RetrievalRequest{
		Query:         "do thing",
		Mode:          "keyword",
		ExpandHops:    1,
		ExpandCallers: true,
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	callers := blocks[0].Callers
	if got := len(callers); got != 5 {
		t.Fatalf("expected 5 callers (Top-5), got %d", got)
	}

	// 断言按字典序升序，且为前 5 个（callA..callE）。
	want := []string{"callA", "callB", "callC", "callD", "callE"}
	for i, w := range want {
		if callers[i].Name != w {
			t.Errorf("callers[%d].Name = %q, want %q", i, callers[i].Name, w)
		}
	}

	// Callees 未开（零值 false），应为空。
	if len(blocks[0].Callees) != 0 {
		t.Errorf("expected empty callees (ExpandCallees=false), got %d", len(blocks[0].Callees))
	}
}

// TestHybridRetriever_ExpandSwitches 验证 ExpandCallers/ExpandCallees 开关：
// ExpandCallers=false → Callers 为空；ExpandCallees=true → Callees 有结果。
func TestHybridRetriever_ExpandSwitches(t *testing.T) {
	const hitSymbol = "sym-1"
	edges := &fakeEdgeExpander{
		callers: map[string][]*models.EdgeWithDetails{
			hitSymbol: {makeEdgeDetail("caller-1", "CallerOne")},
		},
		callees: map[string][]*models.EdgeWithDetails{
			hitSymbol: {makeEdgeDetail("callee-1", "CalleeOne")},
		},
	}
	vs := &fakeVectorSearcher{
		keywordResults: []*models.VectorSearchResult{
			makeKeywordHit(hitSymbol, "DoThing", 0.5),
		},
	}
	r := NewHybridRetriever(vs, edges, nil, DefaultHybridRetrieverConfig())

	blocks, err := r.Query(context.Background(), RetrievalRequest{
		Query:         "do thing",
		Mode:          "keyword",
		ExpandHops:    1,
		ExpandCallers: false, // 显式关闭 callers
		ExpandCallees: true,  // 显式开启 callees
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	// Callers 应为空：开关关闭即便 expander 有数据也不应拉取。
	if len(blocks[0].Callers) != 0 {
		t.Errorf("expected empty Callers (ExpandCallers=false), got %d", len(blocks[0].Callers))
	}
	// Callees 应有结果。
	if len(blocks[0].Callees) != 1 {
		t.Fatalf("expected 1 Callee, got %d", len(blocks[0].Callees))
	}
	if blocks[0].Callees[0].Name != "CalleeOne" {
		t.Errorf("Callees[0].Name = %q, want %q", blocks[0].Callees[0].Name, "CalleeOne")
	}
}

package models

import (
	"math"
	"testing"
)

// TestFuseHybridResults 钉住 HybridSearch 的纯函数核心：
// 归一化后的两路分数 → 加权融合 → 去重（同 EntityID 合并）→ 降序 → 截断。
//
// 该测试不依赖 DB，验证语义不变量：
//   - 同 entity 在向量与关键词两路都命中时，两路分数都保留
//   - 最终 Similarity = VectorScore*wVec + KeywordScore*wKw
//   - 按总分降序
//   - limit 截断生效
func TestFuseHybridResults(t *testing.T) {
	// 构造 merge：分数已"除以本路 max"归一化完毕（这是 HybridSearch 的输入约定）
	// - "a" 在两路都命中（去重合并场景）
	// - "b" 仅向量命中
	// - "c" 仅关键词命中
	merge := map[string]*HybridSearchResult{
		"a": {VectorSearchResult: VectorSearchResult{EntityID: "a"}, VectorScore: 0.8, KeywordScore: 0.5},
		"b": {VectorSearchResult: VectorSearchResult{EntityID: "b"}, VectorScore: 1.0, KeywordScore: 0.0},
		"c": {VectorSearchResult: VectorSearchResult{EntityID: "c"}, VectorScore: 0.0, KeywordScore: 1.0},
	}

	// 默认权重 0.7/0.3
	got := fuseHybridResults(merge, 0.7, 0.3, 0)

	if len(got) != 3 {
		t.Fatalf("expected 3 fused results, got %d", len(got))
	}

	// 期望分数
	want := map[string]float64{
		"a": 0.8*0.7 + 0.5*0.3, // 0.71
		"b": 1.0*0.7 + 0.0*0.3, // 0.70
		"c": 0.0*0.7 + 1.0*0.3, // 0.30
	}
	for _, h := range got {
		w, ok := want[h.EntityID]
		if !ok {
			t.Errorf("unexpected entity %q in results", h.EntityID)
			continue
		}
		if math.Abs(h.Similarity-w) > 1e-9 {
			t.Errorf("entity %q: similarity = %.6f, want %.6f", h.EntityID, h.Similarity, w)
		}
	}

	// 期望降序：a(0.71) > b(0.70) > c(0.30)
	if got[0].EntityID != "a" || got[1].EntityID != "b" || got[2].EntityID != "c" {
		t.Errorf("expected order a,b,c; got %s,%s,%s", got[0].EntityID, got[1].EntityID, got[2].EntityID)
	}
}

// TestFuseHybridResults_DedupPreservesBothScores 验证同 entity 两路命中时，
// 向量主体保留、关键词分数补到 KeywordScore，二者都进最终融合（不丢分）。
func TestFuseHybridResults_DedupPreservesBothScores(t *testing.T) {
	merge := map[string]*HybridSearchResult{
		"shared": {VectorSearchResult: VectorSearchResult{EntityID: "shared"}, VectorScore: 0.9, KeywordScore: 0.6},
	}
	got := fuseHybridResults(merge, 0.7, 0.3, 0)
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].VectorScore != 0.9 || got[0].KeywordScore != 0.6 {
		t.Errorf("both scores must be preserved: got vec=%v kw=%v", got[0].VectorScore, got[0].KeywordScore)
	}
	wantSim := 0.9*0.7 + 0.6*0.3
	if math.Abs(got[0].Similarity-wantSim) > 1e-9 {
		t.Errorf("fused similarity = %.6f, want %.6f", got[0].Similarity, wantSim)
	}
}

// TestFuseHybridResults_LimitTruncation 验证 limit 截断：取总分最高的前 N 个。
func TestFuseHybridResults_LimitTruncation(t *testing.T) {
	merge := map[string]*HybridSearchResult{
		"low":  {VectorSearchResult: VectorSearchResult{EntityID: "low"}, VectorScore: 0.1, KeywordScore: 0.0},
		"high": {VectorSearchResult: VectorSearchResult{EntityID: "high"}, VectorScore: 1.0, KeywordScore: 1.0},
		"mid":  {VectorSearchResult: VectorSearchResult{EntityID: "mid"}, VectorScore: 0.5, KeywordScore: 0.5},
	}
	got := fuseHybridResults(merge, 0.7, 0.3, 2)
	if len(got) != 2 {
		t.Fatalf("limit=2 expected 2 results, got %d", len(got))
	}
	// high(1.0) > mid(0.5) > low(0.1)
	if got[0].EntityID != "high" || got[1].EntityID != "mid" {
		t.Errorf("expected [high, mid], got [%s, %s]", got[0].EntityID, got[1].EntityID)
	}
}

// TestFuseHybridResults_NoLimit 不传 limit（=0）时返回全部，不截断。
func TestFuseHybridResults_NoLimit(t *testing.T) {
	merge := map[string]*HybridSearchResult{
		"x": {VectorSearchResult: VectorSearchResult{EntityID: "x"}, VectorScore: 1.0},
	}
	got := fuseHybridResults(merge, 1.0, 0.0, 0)
	if len(got) != 1 {
		t.Fatalf("limit=0 should not truncate; got %d results", len(got))
	}
}

// TestFuseHybridResults_WeightNormalization 验证权重会被 HybridSearch
// 归一化后再传入（这里直接传归一化后的值，验证乘法正确）。
// 当 keyword 权重为 1.0、vector 为 0 时，结果应等于 KeywordScore。
func TestFuseHybridResults_WeightNormalization(t *testing.T) {
	merge := map[string]*HybridSearchResult{
		"k": {VectorSearchResult: VectorSearchResult{EntityID: "k"}, VectorScore: 0.4, KeywordScore: 0.9},
	}
	got := fuseHybridResults(merge, 0.0, 1.0, 0)
	if math.Abs(got[0].Similarity-0.9) > 1e-9 {
		t.Errorf("kw-only weight: similarity=%.6f, want 0.9", got[0].Similarity)
	}
}

// TestFuseHybridResults_TieBreakEqualScores 等分时不应 panic，结果数应正确。
// sort.Slice 非稳定，但等分项的相对顺序对正确性无影响（分数相同）。
func TestFuseHybridResults_TieBreakEqualScores(t *testing.T) {
	merge := map[string]*HybridSearchResult{
		"p": {VectorSearchResult: VectorSearchResult{EntityID: "p"}, VectorScore: 0.5, KeywordScore: 0.5},
		"q": {VectorSearchResult: VectorSearchResult{EntityID: "q"}, VectorScore: 0.5, KeywordScore: 0.5},
	}
	got := fuseHybridResults(merge, 0.5, 0.5, 0)
	if len(got) != 2 {
		t.Fatalf("expected 2 results with equal scores, got %d", len(got))
	}
	for _, h := range got {
		if math.Abs(h.Similarity-0.5) > 1e-9 {
			t.Errorf("tie score: similarity=%.6f, want 0.5", h.Similarity)
		}
	}
}

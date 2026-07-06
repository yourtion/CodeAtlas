package qa

import (
	"strings"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

func TestBuildPrompt_Format(t *testing.T) {
	blocks := []retrieval.ContextBlock{
		{
			Symbol:     retrieval.ContextSymbol{Name: "FuncA", FilePath: "a.go:10", Signature: "func FuncA()"},
			Similarity: 0.9,
			Callers:    []retrieval.ContextSymbol{{Name: "Caller1", FilePath: "c.go:1"}},
		},
	}
	prompt, truncated := BuildPrompt("how does FuncA work", []string{"repo-1"}, blocks, nil, DefaultPromptBuildOptions())
	if truncated {
		t.Error("expected not truncated for small input")
	}
	mustContain := []string{"# Code Context", "## Question", "FuncA", "0.90", "Caller1", "repo-1", "## Repositories", "## Relevant Symbols", "func FuncA()"}
	for _, s := range mustContain {
		if !strings.Contains(prompt, s) {
			t.Errorf("prompt missing %q\ngot:\n%s", s, prompt)
		}
	}
}

func TestBuildPrompt_EmptyRepoIDs(t *testing.T) {
	// repoIDs 为空时不应出现 Repositories 段。
	blocks := []retrieval.ContextBlock{
		{Symbol: retrieval.ContextSymbol{Name: "FuncA"}, Similarity: 0.5},
	}
	prompt, _ := BuildPrompt("q", nil, blocks, nil, DefaultPromptBuildOptions())
	if strings.Contains(prompt, "## Repositories") {
		t.Error("should not emit Repositories section when repoIDs is empty")
	}
	if strings.Contains(prompt, "## Relevant Symbols") == false {
		t.Error("should still emit Relevant Symbols section")
	}
}

func TestBuildPrompt_TruncationDropsLowScoreNeighbors(t *testing.T) {
	// 验证差异化截断：预算恰好让"砍掉邻居后的高分 block 放得下、
	// 带邻居放不下、低分 block 即使砍掉邻居也放不下"。
	//
	// formatBlockSection 实测长度（见下方断言依赖）：
	//   high 带邻居 = 102 chars，砍邻居后 = 58 chars
	//   low  带邻居 = 100 chars，砍邻居后 = 57 chars
	//
	// 选 MaxTokens=20 → charBudget=80：
	//   1. high 全段 102 > 80 → 砍掉 Callers/Callees 重算为 58 ≤ 80 → 写入，邻居被丢弃
	//   2. 写入 high 后剩余 80-58=22；low 砍邻居后仍有 57 > 22 → 整个 block 跳过
	// 从而同时验证三件事：高分符号保留、高分邻居被砍、低分 block 整体丢弃、truncated=true。
	high := retrieval.ContextBlock{
		Symbol:     retrieval.ContextSymbol{Name: "HighScore", FilePath: "h.go:1"},
		Similarity: 0.95,
		Callers:    []retrieval.ContextSymbol{{Name: "CallerHigh", FilePath: "ch.go:1"}},
	}
	low := retrieval.ContextBlock{
		Symbol:     retrieval.ContextSymbol{Name: "LowScore", FilePath: "l.go:1"},
		Similarity: 0.1,
		Callers:    []retrieval.ContextSymbol{{Name: "CallerLow", FilePath: "cl.go:1"}},
	}

	// 前置条件：fixture 与预算的关系必须符合上面的算式，否则测试本身失效。
	highFull := len(formatBlockSection(1, high, nil, false))
	highStripped := retrieval.ContextBlock{Symbol: high.Symbol, Similarity: high.Similarity}
	highStrippedLen := len(formatBlockSection(1, highStripped, nil, false))
	lowStripped := retrieval.ContextBlock{Symbol: low.Symbol, Similarity: low.Similarity}
	lowStrippedLen := len(formatBlockSection(2, lowStripped, nil, false))
	charBudget := 80
	if !(highFull > charBudget && highStrippedLen <= charBudget && charBudget-highStrippedLen < lowStrippedLen) {
		t.Fatalf("fixture/budget invariant broken: highFull=%d highStripped=%d lowStripped=%d budget=%d",
			highFull, highStrippedLen, lowStrippedLen, charBudget)
	}

	prompt, truncated := BuildPrompt("q", nil, []retrieval.ContextBlock{high, low}, nil, PromptBuildOptions{MaxTokens: 20})
	if !truncated {
		t.Fatal("expected truncated=true: high's neighbors were stripped and low was dropped")
	}

	// 高分 block 被保留（符号本身仍在 prompt 里），但它的邻居被砍掉了。
	if !strings.Contains(prompt, "HighScore") {
		t.Errorf("expected high-score block symbol to survive truncation\ngot:\n%s", prompt)
	}
	if strings.Contains(prompt, "CallerHigh") {
		t.Errorf("expected high-score block's caller to be dropped to fit budget\ngot:\n%s", prompt)
	}

	// 低分 block 整个被丢弃（既不含符号名，也不含其 caller）。
	if strings.Contains(prompt, "LowScore") {
		t.Errorf("expected low-score block to be dropped entirely (not enough budget)\ngot:\n%s", prompt)
	}
	if strings.Contains(prompt, "CallerLow") {
		t.Errorf("expected low-score block's caller to be absent\ngot:\n%s", prompt)
	}

	if !strings.HasPrefix(prompt, "# Code Context") {
		t.Error("prompt should always start with header")
	}
}

func TestBuildPrompt_IncludeSource(t *testing.T) {
	blocks := []retrieval.ContextBlock{
		{
			Symbol:     retrieval.ContextSymbol{Name: "FuncB", FilePath: "b.go:5"},
			Similarity: 0.8,
			ChunkID:    "chunk-b",
		},
	}
	sources := map[string]string{
		"chunk-b": "func FuncB() {\n\treturn\n}",
	}
	prompt, truncated := BuildPrompt("explain FuncB", []string{"repo-2"}, blocks, sources, PromptBuildOptions{MaxTokens: 8000, IncludeSource: true})
	if truncated {
		t.Error("expected not truncated")
	}
	if !strings.Contains(prompt, "```\nfunc FuncB()") {
		t.Errorf("expected fenced source block in prompt\ngot:\n%s", prompt)
	}
	if !strings.Contains(prompt, "func FuncB() {\n\treturn\n}") {
		t.Error("expected source text to appear verbatim inside fenced block")
	}
}

func TestBuildPrompt_IncludeSourceMissingChunk(t *testing.T) {
	// IncludeSource=true 但 sources 中无对应 chunkID：不应 panic，不应渲染 fenced block。
	blocks := []retrieval.ContextBlock{
		{
			Symbol:     retrieval.ContextSymbol{Name: "FuncC", FilePath: "c.go:5"},
			Similarity: 0.8,
			ChunkID:    "chunk-c",
		},
	}
	prompt, _ := BuildPrompt("q", nil, blocks, map[string]string{}, PromptBuildOptions{MaxTokens: 8000, IncludeSource: true})
	if strings.Contains(prompt, "```") {
		t.Error("should not emit fenced block when source missing")
	}
}

func TestBuildPrompt_DefaultOptsWhenZero(t *testing.T) {
	// MaxTokens=0 应回退到默认（不截断小输入）。
	blocks := []retrieval.ContextBlock{
		{Symbol: retrieval.ContextSymbol{Name: "FuncA"}, Similarity: 0.5},
	}
	prompt, truncated := BuildPrompt("q", nil, blocks, nil, PromptBuildOptions{})
	if truncated {
		t.Error("zero MaxTokens should fall back to default and not truncate small input")
	}
	if !strings.Contains(prompt, "FuncA") {
		t.Error("expected FuncA in prompt")
	}
}

func TestBuildPrompt_CalleesRendered(t *testing.T) {
	blocks := []retrieval.ContextBlock{
		{
			Symbol:     retrieval.ContextSymbol{Name: "FuncA", FilePath: "a.go:1"},
			Similarity: 0.7,
			Callees:    []retrieval.ContextSymbol{{Name: "Callee1", FilePath: "call.go:9"}},
		},
	}
	prompt, _ := BuildPrompt("q", nil, blocks, nil, DefaultPromptBuildOptions())
	if !strings.Contains(prompt, "Callee1") {
		t.Errorf("expected Callee1 in prompt\ngot:\n%s", prompt)
	}
	if !strings.Contains(prompt, "## Calls") && !strings.Contains(prompt, "- **Calls**") {
		t.Error("expected Calls section")
	}
}

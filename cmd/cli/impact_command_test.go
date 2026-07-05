package main

import (
	"strings"
	"testing"
	"time"

	"github.com/yourtionguo/CodeAtlas/pkg/client"
)

// TestRenderImpactTree_GroupsByDepth 验证符号按 depth 分层、同层按名排序。
func TestRenderImpactTree_GroupsByDepth(t *testing.T) {
	resp := &client.TransitiveResponse{
		Symbols: []client.ReachableSymbol{
			{SymbolID: "s2", Name: "validateInput", Kind: "function", FilePath: "main.go", Depth: 1},
			{SymbolID: "s1", Name: "processData", Kind: "function", FilePath: "main.go", Depth: 1},
			{SymbolID: "s3", Name: "saveToDB", Kind: "function", FilePath: "db.go", Depth: 2},
		},
		Total: 3,
		Depth: 5,
	}

	out := renderImpactTree(resp, "callees", 10*time.Millisecond)

	// 头部含符号数与最大深度
	if !strings.Contains(out, "3 symbols") {
		t.Errorf("output should mention total symbols, got: %s", out)
	}
	if !strings.Contains(out, "max depth=5") {
		t.Errorf("output should mention max depth, got: %s", out)
	}

	// depth=1 段：processData 应在 validateInput 之前（按名排序）
	idxProcess := strings.Index(out, "processData")
	idxValidate := strings.Index(out, "validateInput")
	if idxProcess < 0 || idxValidate < 0 {
		t.Fatalf("missing expected symbols in output: %s", out)
	}
	if idxProcess > idxValidate {
		t.Errorf("processData should appear before validateInput (sorted by name), got output:\n%s", out)
	}

	// depth=2 段：saveToDB 应出现且在 depth=1 段之后
	idxSave := strings.Index(out, "saveToDB")
	if idxSave < 0 {
		t.Fatalf("missing saveToDB in output: %s", out)
	}
	if idxSave < idxValidate {
		t.Errorf("saveToDB (depth=2) should appear after depth=1 symbols, got output:\n%s", out)
	}
}

// TestRenderImpactTree_CallersDirection 验证 direction=callers 的措辞。
func TestRenderImpactTree_CallersDirection(t *testing.T) {
	resp := &client.TransitiveResponse{
		Symbols: []client.ReachableSymbol{
			{SymbolID: "s1", Name: "main", Kind: "function", FilePath: "main.go", Depth: 1},
		},
		Total: 1,
		Depth: 3,
	}

	out := renderImpactTree(resp, "callers", 5*time.Millisecond)
	if !strings.Contains(out, "Transitive callers") {
		t.Errorf("callers direction should say 'Transitive callers', got: %s", out)
	}
	if !strings.Contains(out, "caller (depth=1") {
		t.Errorf("callers should use 'caller' noun per depth, got: %s", out)
	}
}

// TestRenderImpactTree_EmptyResponse 验证空结果输出友好提示。
func TestRenderImpactTree_EmptyResponse(t *testing.T) {
	out := renderImpactTree(nil, "callees", 0)
	if !strings.Contains(out, "No transitive callees found") {
		t.Errorf("nil response should print friendly empty message, got: %s", out)
	}

	out = renderImpactTree(&client.TransitiveResponse{}, "callers", 0)
	if !strings.Contains(out, "No transitive callers found") {
		t.Errorf("empty response should print friendly empty message, got: %s", out)
	}
}

// TestRenderImpactTree_SkipsMissingDepthLayers 验证中间 depth 缺层时跳过、不打印空段。
func TestRenderImpactTree_SkipsMissingDepthLayers(t *testing.T) {
	// 只有 depth=1 和 depth=3，无 depth=2
	resp := &client.TransitiveResponse{
		Symbols: []client.ReachableSymbol{
			{SymbolID: "s1", Name: "a", Kind: "function", FilePath: "a.go", Depth: 1},
			{SymbolID: "s3", Name: "c", Kind: "function", FilePath: "c.go", Depth: 3},
		},
		Total: 2,
		Depth: 3,
	}

	out := renderImpactTree(resp, "callees", 0)
	if strings.Contains(out, "depth=2, 0") {
		t.Errorf("should not print empty depth=2 layer, got: %s", out)
	}
	if !strings.Contains(out, "depth=1") || !strings.Contains(out, "depth=3") {
		t.Errorf("should print depth=1 and depth=3 layers, got: %s", out)
	}
}

package search

import (
	"testing"
)

// 候选结果保持相似度降序的样本数据，用于验证过滤后保序。
func sampleCandidates() []Candidate {
	return []Candidate{
		{SymbolID: "s1", Name: "FuncA", Kind: "function", Language: "go", RepoID: "repo-1", Similarity: 0.95},
		{SymbolID: "s2", Name: "ClassB", Kind: "class", Language: "python", RepoID: "repo-1", Similarity: 0.90},
		{SymbolID: "s3", Name: "FuncC", Kind: "function", Language: "go", RepoID: "repo-2", Similarity: 0.85},
		{SymbolID: "s4", Name: "VarD", Kind: "variable", Language: "go", RepoID: "repo-1", Similarity: 0.80},
	}
}

func TestInMemoryFilter_NoCriteria(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{})
	got := f.Filter(sampleCandidates())
	if len(got) != 4 {
		t.Fatalf("no filter should keep all 4, got %d", len(got))
	}
}

func TestInMemoryFilter_ByKind(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{Kind: []string{"function"}})
	got := f.Filter(sampleCandidates())
	if len(got) != 2 {
		t.Fatalf("kind=function should match 2, got %d", len(got))
	}
	for _, c := range got {
		if c.Kind != "function" {
			t.Errorf("expected kind function, got %s", c.Kind)
		}
	}
}

func TestInMemoryFilter_ByLanguage(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{Language: "go"})
	got := f.Filter(sampleCandidates())
	if len(got) != 3 {
		t.Fatalf("language=go should match 3, got %d", len(got))
	}
}

func TestInMemoryFilter_ByRepo(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{RepoID: "repo-2"})
	got := f.Filter(sampleCandidates())
	if len(got) != 1 || got[0].SymbolID != "s3" {
		t.Fatalf("repo-2 should match only s3, got %v", got)
	}
}

// TestInMemoryFilter_Combined 验证多维度组合（AND 语义）。
func TestInMemoryFilter_Combined(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{Kind: []string{"function"}, Language: "go", RepoID: "repo-1"})
	got := f.Filter(sampleCandidates())
	// function + go + repo-1: 只剩 s1
	if len(got) != 1 || got[0].SymbolID != "s1" {
		t.Fatalf("combined filter should match only s1, got %v", got)
	}
}

// TestInMemoryFilter_PreservesOrder 验证过滤后保持相似度降序。
func TestInMemoryFilter_PreservesOrder(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{Language: "go"})
	got := f.Filter(sampleCandidates())
	// go 候选: s1(0.95), s3(0.85), s4(0.80)
	if len(got) != 3 {
		t.Fatalf("expected 3 go candidates, got %d", len(got))
	}
	if got[0].SymbolID != "s1" || got[1].SymbolID != "s3" || got[2].SymbolID != "s4" {
		t.Errorf("order not preserved: got %s %s %s", got[0].SymbolID, got[1].SymbolID, got[2].SymbolID)
	}
}

// TestInMemoryFilter_KindOR 验证 kind 是 OR 语义（任一匹配即保留）。
func TestInMemoryFilter_KindOR(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{Kind: []string{"class", "variable"}})
	got := f.Filter(sampleCandidates())
	// class + variable: s2, s4
	if len(got) != 2 {
		t.Fatalf("kind OR should match 2, got %d", len(got))
	}
}

// TestInMemoryFilter_EmptyInput 验证空输入返回空（不 panic）。
func TestInMemoryFilter_EmptyInput(t *testing.T) {
	f := NewInMemoryFilter(FilterCriteria{Kind: []string{"function"}})
	got := f.Filter(nil)
	if len(got) != 0 {
		t.Errorf("nil input should return empty, got %d", len(got))
	}
}

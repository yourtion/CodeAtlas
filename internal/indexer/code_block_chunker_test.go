package indexer

import (
	"strings"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

func mkSymbol(id, fileID, name string, kind schema.SymbolKind, startLine, endLine int, signature string) schema.Symbol {
	return schema.Symbol{
		SymbolID:  id,
		FileID:    fileID,
		Name:      name,
		Kind:      kind,
		Signature: signature,
		Span:      schema.Span{StartLine: startLine, EndLine: endLine},
	}
}

// TestCodeBlockChunker_GroupsAdjacentSymbols 验证相邻符号被合并为一块。
func TestCodeBlockChunker_GroupsAdjacentSymbols(t *testing.T) {
	symbols := []schema.Symbol{
		mkSymbol("s1", "f1", "helperA", schema.SymbolFunction, 10, 15, "func helperA()"),
		mkSymbol("s2", "f1", "helperB", schema.SymbolFunction, 18, 25, "func helperB()"),
	}
	c := NewCodeBlockChunker(30)
	got := c.Chunk(symbols)
	if len(got) != 1 {
		t.Fatalf("adjacent symbols (gap<=threshold) should merge into 1 block, got %d", len(got))
	}
	// 内容应包含两个符号的签名
	if !strings.Contains(got[0].Content, "helperA") || !strings.Contains(got[0].Content, "helperB") {
		t.Errorf("block content should contain both signatures, got: %s", got[0].Content)
	}
}

// TestCodeBlockChunker_SplitsDistantSymbols 验证行距超阈值的符号分到不同块。
func TestCodeBlockChunker_SplitsDistantSymbols(t *testing.T) {
	symbols := []schema.Symbol{
		mkSymbol("s1", "f1", "near", schema.SymbolFunction, 10, 15, "func near()"),
		mkSymbol("s2", "f1", "far", schema.SymbolFunction, 100, 110, "func far()"),
	}
	c := NewCodeBlockChunker(30)
	got := c.Chunk(symbols)
	if len(got) != 2 {
		t.Fatalf("distant symbols (gap>threshold) should split into 2 blocks, got %d", len(got))
	}
}

// TestCodeBlockChunker_SeparatesFiles 验证不同文件的符号不会合并。
func TestCodeBlockChunker_SeparatesFiles(t *testing.T) {
	symbols := []schema.Symbol{
		mkSymbol("s1", "f1", "a", schema.SymbolFunction, 10, 15, "func a()"),
		mkSymbol("s2", "f2", "b", schema.SymbolFunction, 12, 16, "func b()"),
	}
	c := NewCodeBlockChunker(30)
	got := c.Chunk(symbols)
	if len(got) != 2 {
		t.Fatalf("symbols in different files should not merge, got %d blocks", len(got))
	}
}

// TestCodeBlockChunker_PrimarySymbolPriority 验证块的 entity_id 取主要符号
// （function 优先于 variable）。
func TestCodeBlockChunker_PrimarySymbolPriority(t *testing.T) {
	symbols := []schema.Symbol{
		mkSymbol("v1", "f1", "config", schema.SymbolVariable, 10, 10, "var config"),
		mkSymbol("fn1", "f1", "process", schema.SymbolFunction, 12, 20, "func process()"),
	}
	c := NewCodeBlockChunker(30)
	got := c.Chunk(symbols)
	if len(got) != 1 {
		t.Fatalf("expected 1 block, got %d", len(got))
	}
	if got[0].EntityID != "fn1" {
		t.Errorf("primary entity should be function fn1, got %s", got[0].EntityID)
	}
}

// TestCodeBlockChunker_SkipsEmptyContent 验证无内容符号被跳过。
func TestCodeBlockChunker_SkipsEmptyContent(t *testing.T) {
	empty := mkSymbol("s1", "f1", "x", schema.SymbolFunction, 10, 15, "")
	empty.Docstring = ""
	empty.SemanticSummary = ""
	rich := mkSymbol("s2", "f1", "y", schema.SymbolFunction, 12, 16, "func y()")
	c := NewCodeBlockChunker(30)
	got := c.Chunk([]schema.Symbol{empty, rich})
	if len(got) != 1 {
		t.Fatalf("empty-content symbol should be skipped, expected 1 block, got %d", len(got))
	}
	if got[0].EntityID != "s2" {
		t.Errorf("only the rich symbol should produce a block, got entity %s", got[0].EntityID)
	}
}

// TestCodeBlockChunker_EmptyInput 验证空输入不 panic。
func TestCodeBlockChunker_EmptyInput(t *testing.T) {
	c := NewCodeBlockChunker(30)
	if got := c.Chunk(nil); len(got) != 0 {
		t.Errorf("nil input should return empty, got %d", len(got))
	}
}

// TestSymbolChunker_Baseline 对照测试：确认默认 SymbolChunker 仍逐符号产出。
func TestSymbolChunker_Baseline(t *testing.T) {
	symbols := []schema.Symbol{
		mkSymbol("s1", "f1", "a", schema.SymbolFunction, 10, 15, "func a()"),
		mkSymbol("s2", "f1", "b", schema.SymbolFunction, 12, 16, "func b()"),
	}
	got := SymbolChunker{}.Chunk(symbols)
	if len(got) != 2 {
		t.Fatalf("SymbolChunker should produce 1 input per symbol, got %d", len(got))
	}
}

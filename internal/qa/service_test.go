package qa

import (
	"context"
	"testing"

	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

type fakeRetriever struct {
	blocks []retrieval.ContextBlock
	err    error
}

func (f *fakeRetriever) Query(ctx context.Context, req retrieval.RetrievalRequest) ([]retrieval.ContextBlock, error) {
	return f.blocks, f.err
}

type fakeSourceFetcher struct {
	data map[string]string
	err  error
}

func (f *fakeSourceFetcher) GetByVectorIDs(ctx context.Context, ids []string) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.data, nil
}

func TestService_Ask_BasicFlow(t *testing.T) {
	fr := &fakeRetriever{
		blocks: []retrieval.ContextBlock{
			{
				Symbol:     retrieval.ContextSymbol{SymbolID: "sym-1", Name: "Foo", Kind: "function", FilePath: "a/foo.go"},
				Similarity: 0.92,
				MatchMode:  "hybrid",
				ChunkID:    "chunk-1",
				Callers: []retrieval.ContextSymbol{
					{SymbolID: "sym-2", Name: "Bar", Kind: "function", FilePath: "a/bar.go"},
				},
			},
		},
	}
	svc := NewService(fr, nil, DefaultPromptBuildOptions())

	resp, err := svc.Ask(context.Background(), AskRequest{Query: "how does Foo work?"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Query 回显
	if resp.Query != "how does Foo work?" {
		t.Errorf("Query = %q, want %q", resp.Query, "how does Foo work?")
	}

	// Blocks 数量
	if len(resp.Blocks) != 1 {
		t.Fatalf("len(Blocks) = %d, want 1", len(resp.Blocks))
	}
	if resp.Blocks[0].Symbol.Name != "Foo" {
		t.Errorf("Block[0].Symbol.Name = %q, want %q", resp.Blocks[0].Symbol.Name, "Foo")
	}
	if resp.Blocks[0].ChunkID != "chunk-1" {
		t.Errorf("Block[0].ChunkID = %q, want %q", resp.Blocks[0].ChunkID, "chunk-1")
	}
	if len(resp.Blocks[0].Callers) != 1 {
		t.Errorf("Block[0] Callers len = %d, want 1", len(resp.Blocks[0].Callers))
	}

	// ChunkIDs 汇总
	if len(resp.ChunkIDs) != 1 || resp.ChunkIDs[0] != "chunk-1" {
		t.Errorf("ChunkIDs = %v, want [chunk-1]", resp.ChunkIDs)
	}

	// Prompt 非空
	if resp.Prompt == "" {
		t.Error("Prompt is empty")
	}
}

func TestService_Ask_EmptyQueryReturnsError(t *testing.T) {
	svc := NewService(&fakeRetriever{}, nil, DefaultPromptBuildOptions())

	_, err := svc.Ask(context.Background(), AskRequest{Query: ""})
	if err == nil {
		t.Fatal("expected error for empty query, got nil")
	}
}

func TestService_Ask_IncludeSourceFillsSource(t *testing.T) {
	fr := &fakeRetriever{
		blocks: []retrieval.ContextBlock{
			{
				Symbol:     retrieval.ContextSymbol{SymbolID: "sym-1", Name: "Foo", Kind: "function"},
				Similarity: 0.9,
				ChunkID:    "chunk-1",
			},
		},
	}
	fsf := &fakeSourceFetcher{
		data: map[string]string{"chunk-1": "func Foo() {}"},
	}
	svc := NewService(fr, fsf, DefaultPromptBuildOptions())

	resp, err := svc.Ask(context.Background(), AskRequest{Query: "Foo?", IncludeSource: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Blocks[0].Source != "func Foo() {}" {
		t.Errorf("Block[0].Source = %q, want %q", resp.Blocks[0].Source, "func Foo() {}")
	}
}

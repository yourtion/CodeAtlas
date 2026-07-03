package indexer

import (
	"context"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/yourtionguo/CodeAtlas/internal/schema"
)

// fakeExecutor 是 pipelineExecutor 的测试替身，记录每个阶段被调用的次数，
// 用于断言索引管道的副作用行为（无需真实数据库）。
type fakeExecutor struct {
	mu sync.Mutex

	writeRepositoryCalls   int
	writeDataCalls         int
	buildGraphCalls        int
	generateEmbeddingsCalls int

	// 返回值（默认零值结果，调用方可按需覆盖）
	writeRepoErr     error
	writeDataResult  *WriteResult
	writeDataErr     error
	graphResult      *GraphBuildResult
	embedResult      *EmbedResult
}

func (f *fakeExecutor) writeRepository(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeRepositoryCalls++
	return f.writeRepoErr
}

func (f *fakeExecutor) writeData(ctx context.Context, files []schema.File, edges []schema.DependencyEdge) (*WriteResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writeDataCalls++
	if f.writeDataResult == nil {
		return &WriteResult{}, f.writeDataErr
	}
	return f.writeDataResult, f.writeDataErr
}

func (f *fakeExecutor) buildGraph(ctx context.Context, files []schema.File, edges []schema.DependencyEdge) *GraphBuildResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buildGraphCalls++
	if f.graphResult == nil {
		return &GraphBuildResult{}
	}
	return f.graphResult
}

func (f *fakeExecutor) generateEmbeddings(ctx context.Context, files []schema.File) *EmbedResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.generateEmbeddingsCalls++
	if f.embedResult == nil {
		return &EmbedResult{}
	}
	return f.embedResult
}

// newIndexerWithFake 构造一个走 fake executor 的 Indexer，
// 绕过真实数据库依赖。SkipVectors=true 避免构造 embedder。
func newIndexerWithFake(t *testing.T) (*Indexer, *fakeExecutor) {
	t.Helper()
	config := DefaultIndexerConfig()
	config.RepoID = uuid.New().String()
	config.RepoName = "test-repo"
	config.SkipVectors = true

	idx := NewIndexer(nil, config)
	fake := &fakeExecutor{}
	idx.SetExecutor(fake)
	return idx, fake
}

// TestIndexWithProgress_DoesNotRunPipelineTwice 钉住阶段一修复的 Bug #1：
// IndexWithProgress 末尾曾错误地调用 idx.Index(ctx, input)，
// 导致整个索引管道（writeRepository/writeData/buildGraph/generateEmbeddings）
// 被执行两次。修复后每个阶段应只被调用一次。
//
// 若此测试失败（计数为 2），说明双跑 bug 复发。
func TestIndexWithProgress_DoesNotRunPipelineTwice(t *testing.T) {
	idx, fake := newIndexerWithFake(t)

	input := &schema.ParseOutput{
		Metadata: schema.ParseMetadata{Version: "test-1.0"},
		Files: []schema.File{
			{
				FileID:   uuid.New().String(),
				Path:     "main.go",
				Checksum: "abc123",
				Language: "go",
			},
		},
	}

	ctx := context.Background()
	result, err := idx.IndexWithProgress(ctx, input, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// 核心断言：每个阶段恰好执行一次（双跑 bug 会导致各为 2 次）
	if fake.writeRepositoryCalls != 1 {
		t.Errorf("writeRepository should be called once, got %d (double-run bug?)", fake.writeRepositoryCalls)
	}
	if fake.writeDataCalls != 1 {
		t.Errorf("writeData should be called once, got %d (double-run bug?)", fake.writeDataCalls)
	}
	if fake.buildGraphCalls != 1 {
		t.Errorf("buildGraph should be called once, got %d (double-run bug?)", fake.buildGraphCalls)
	}
	// generateEmbeddings 在 SkipVectors=true 时不进入该分支，预期 0 次
	if fake.generateEmbeddingsCalls != 0 {
		t.Errorf("generateEmbeddings should not be called when SkipVectors=true, got %d", fake.generateEmbeddingsCalls)
	}
}

// TestIndex_RunsPipelineOnce 验证 Index 正常路径下每个阶段只执行一次，
// 作为 IndexWithProgress 测试的对照组。
func TestIndex_RunsPipelineOnce(t *testing.T) {
	idx, fake := newIndexerWithFake(t)

	input := &schema.ParseOutput{
		Metadata: schema.ParseMetadata{Version: "test-1.0"},
		Files: []schema.File{
			{
				FileID:   uuid.New().String(),
				Path:     "main.go",
				Checksum: "abc123",
				Language: "go",
			},
		},
	}

	ctx := context.Background()
	result, err := idx.Index(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Status != "success" {
		t.Errorf("expected status 'success', got %q", result.Status)
	}

	if fake.writeRepositoryCalls != 1 {
		t.Errorf("writeRepository should be called once, got %d", fake.writeRepositoryCalls)
	}
	if fake.writeDataCalls != 1 {
		t.Errorf("writeData should be called once, got %d", fake.writeDataCalls)
	}
	if fake.buildGraphCalls != 1 {
		t.Errorf("buildGraph should be called once, got %d", fake.buildGraphCalls)
	}
}

// TestSetExecutor_NilRestoresDefault 验证 SetExecutor(nil) 恢复为 Indexer 自身执行，
// 防止 nil 注入导致空指针。
func TestSetExecutor_NilRestoresDefault(t *testing.T) {
	config := DefaultIndexerConfig()
	config.SkipVectors = true
	idx := NewIndexer(nil, config)

	idx.SetExecutor(nil)
	idx.ensureExecutor()
	if idx.executor == nil {
		t.Fatal("executor should default to Indexer itself, got nil")
	}
	if idx.executor != idx {
		t.Error("executor should be the Indexer itself after SetExecutor(nil)")
	}
}

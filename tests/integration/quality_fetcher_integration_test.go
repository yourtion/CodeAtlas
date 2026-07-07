package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/indexer"
	"github.com/yourtionguo/CodeAtlas/internal/quality"
	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// TestDefaultGraphFetcher_RealDB 验证 DefaultGraphFetcher 在真 DB 上各方法正确。
func TestDefaultGraphFetcher_RealDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testDB := SetupTestDB(t)
	defer testDB.TeardownTestDB(t)

	ctx := context.Background()

	// 索引带边的数据（复用 createTestParseOutputWithRelationships）
	// fixture 已知数据：
	//   3 个符号：main(src/main.go), helper(src/main.go), Utility(src/utils.go)
	//   2 条 call 边：main→helper(同文件), main→Utility(跨文件)
	//   两条边 target_id 均非空，无悬空边；无孤立符号。
	parseOutput := createTestParseOutputWithRelationships()
	repoID := uuid.New().String()
	config := &indexer.IndexerConfig{
		RepoID: repoID, RepoName: "fetcher-test", BatchSize: 10,
		WorkerCount: 2, SkipVectors: true, UseTransactions: true,
	}
	idx := indexer.NewIndexer(testDB.DB, config)
	_, err := idx.Index(ctx, parseOutput)
	require.NoError(t, err)

	edgeRepo := models.NewEdgeRepository(testDB.DB)
	symbolRepo := models.NewSymbolRepository(testDB.DB)
	fetcher := quality.NewDefaultGraphFetcher(edgeRepo, symbolRepo)

	// 1. CountEdgesByType
	byType, err := fetcher.CountEdgesByType(ctx, repoID)
	require.NoError(t, err)
	assert.NotEmpty(t, byType)
	assert.Equal(t, 2, byType["call"], "应有 2 条 call 边")

	// 2. CountDanglingEdges
	dangling, err := fetcher.CountDanglingEdges(ctx, repoID)
	require.NoError(t, err)
	// createTestParseOutputWithRelationships 的边都有 target，应无悬空
	assert.Empty(t, dangling, "fixture 边都应解析到 target")

	// 3. CountTotalSymbols
	total, err := fetcher.CountTotalSymbols(ctx, repoID)
	require.NoError(t, err)
	assert.Equal(t, 3, total)

	// 4. CountOrphanSymbols
	orphans, err := fetcher.CountOrphanSymbols(ctx, repoID)
	require.NoError(t, err)
	assert.Equal(t, 0, orphans, "所有符号都参与边，应无孤立符号")

	// 5. CountCrossFileEdges
	crossFile, err := fetcher.CountCrossFileEdges(ctx, repoID)
	require.NoError(t, err)
	assert.Equal(t, 1, crossFile, "应有 1 条跨文件边 (main.go→utils.go)")

	// 6. ListExtractedEdges
	extracted, err := fetcher.ListExtractedEdges(ctx, repoID)
	require.NoError(t, err)
	assert.NotEmpty(t, extracted)
	assert.Len(t, extracted, 2)
	// 验证结构体字段
	for _, e := range extracted {
		assert.NotEmpty(t, e.SourceName)
		assert.NotEmpty(t, e.EdgeType)
	}

	// 7. CheckCallChainConnectivity（空 chains 应返回 0,0）
	ok, totalChains, err := fetcher.CheckCallChainConnectivity(ctx, repoID, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, ok)
	assert.Equal(t, 0, totalChains)

	// 8. CheckCallChainConnectivity（真值链：main→helper / main→Utility 均 reachable）
	chains := []quality.ExpectedChain{
		{StartName: "main", EndName: "helper", StartFile: "src/main.go", EndFile: "src/main.go"},
		{StartName: "main", EndName: "Utility", StartFile: "src/main.go", EndFile: "src/utils.go"},
	}
	ok, totalChains, err = fetcher.CheckCallChainConnectivity(ctx, repoID, chains)
	require.NoError(t, err)
	assert.Equal(t, 2, ok, "两条链都应 reachable")
	assert.Equal(t, 2, totalChains)

	// 9. CheckCallChainConnectivity（不可达链：helper→main 反向不存在）
	unreachable := []quality.ExpectedChain{
		{StartName: "helper", EndName: "main", StartFile: "src/main.go", EndFile: "src/main.go"},
	}
	ok, totalChains, err = fetcher.CheckCallChainConnectivity(ctx, repoID, unreachable)
	require.NoError(t, err)
	assert.Equal(t, 0, ok, "helper→main 不存在 call 边，应不可达")
	assert.Equal(t, 1, totalChains)
}

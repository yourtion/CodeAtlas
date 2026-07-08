package quality

import (
	"context"
	"fmt"

	"github.com/yourtionguo/CodeAtlas/pkg/models"
)

// DefaultGraphFetcher 是 GraphDataFetcher 的默认实现，
// 组合 EdgeRepository + SymbolRepository，调用 pkg/models 的聚合查询方法。
type DefaultGraphFetcher struct {
	edgeRepo   *models.EdgeRepository
	symbolRepo *models.SymbolRepository
}

// NewDefaultGraphFetcher 构造默认 fetcher。
func NewDefaultGraphFetcher(edgeRepo *models.EdgeRepository, symbolRepo *models.SymbolRepository) *DefaultGraphFetcher {
	return &DefaultGraphFetcher{edgeRepo: edgeRepo, symbolRepo: symbolRepo}
}

func (f *DefaultGraphFetcher) CountEdgesByType(ctx context.Context, repoID string) (map[string]int, error) {
	return models.CountEdgesByType(ctx, f.edgeRepo, repoID)
}

func (f *DefaultGraphFetcher) CountDanglingEdges(ctx context.Context, repoID string) (map[string]int, error) {
	return models.CountDanglingEdges(ctx, f.edgeRepo, repoID)
}

func (f *DefaultGraphFetcher) CountOrphanSymbols(ctx context.Context, repoID string) (int, error) {
	return models.CountOrphanSymbols(ctx, f.symbolRepo, repoID)
}

func (f *DefaultGraphFetcher) CountCrossFileEdges(ctx context.Context, repoID string) (int, error) {
	return models.CountCrossFileEdges(ctx, f.edgeRepo, repoID)
}

func (f *DefaultGraphFetcher) CountTotalSymbols(ctx context.Context, repoID string) (int, error) {
	return models.CountTotalSymbols(ctx, f.symbolRepo, repoID)
}

func (f *DefaultGraphFetcher) CheckCallChainConnectivity(ctx context.Context, repoID string, chains []ExpectedChain) (int, int, error) {
	if len(chains) == 0 {
		return 0, 0, nil
	}
	ok := 0
	for _, c := range chains {
		connected, err := models.CheckSingleChainConnectivity(ctx, f.edgeRepo, repoID, models.ChainSpec{
			StartName: c.StartName,
			EndName:   c.EndName,
			StartFile: c.StartFile,
			EndFile:   c.EndFile,
		})
		if err != nil {
			return 0, 0, fmt.Errorf("check chain %s->%s: %w", c.StartName, c.EndName, err)
		}
		if connected {
			ok++
		}
	}
	return ok, len(chains), nil
}

func (f *DefaultGraphFetcher) ListExtractedEdges(ctx context.Context, repoID string) ([]ExtractedEdge, error) {
	rawEdges, err := models.ListExtractedEdges(ctx, f.edgeRepo, repoID)
	if err != nil {
		return nil, err
	}
	result := make([]ExtractedEdge, len(rawEdges))
	for i, e := range rawEdges {
		result[i] = ExtractedEdge{
			SourceID:   e.SourceID,
			SourceName: e.SourceName,
			EdgeType:   e.EdgeType,
			TargetID:   e.TargetID,
			TargetName: e.TargetName,
		}
	}
	return result, nil
}

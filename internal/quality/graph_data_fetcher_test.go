package quality

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// DefaultGraphFetcher 是适配器，绝大部分方法直接转发到 pkg/models 的聚合查询，
// 后者依赖真实 *EdgeRepository / *SymbolRepository（内含 *sql.DB），无法在单元测试里
// 用 mock 替换。这里只覆盖两条不触 DB 的路径：
//   1. NewDefaultGraphFetcher 构造正确（返回非 nil）。
//   2. CheckCallChainConnectivity 的空 chains 快速路径（len(chains)==0 直接返回）。

func TestNewDefaultGraphFetcher_NonNil(t *testing.T) {
	f := NewDefaultGraphFetcher(nil, nil)
	assert.NotNil(t, f)
}

func TestNewDefaultGraphFetcher_WithRepos(t *testing.T) {
	// 构造时即便传入 nil，结构体字段也应是 nil 而非触发 panic；
	// 这里主要确保构造函数不 panic 且返回值类型正确。
	f := NewDefaultGraphFetcher(nil, nil)
	// 通过断言类型确认返回的是 *DefaultGraphFetcher
	assert.IsType(t, &DefaultGraphFetcher{}, f)
	// nil repo 字段下，空 chains 快速路径不应访问 repo
	ok, total, err := f.CheckCallChainConnectivity(context.Background(), "repo-1", nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, ok)
	assert.Equal(t, 0, total)
}

func TestDefaultGraphFetcher_CheckCallChainConnectivity_EmptyChains(t *testing.T) {
	f := NewDefaultGraphFetcher(nil, nil)
	ok, total, err := f.CheckCallChainConnectivity(context.Background(), "repo-1", nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, ok)
	assert.Equal(t, 0, total)
}

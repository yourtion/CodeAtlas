package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCallAnalysisGroundTruth_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, CallAnalysisGroundTruth, "应至少有一个 fixture 真值")
	for _, gt := range CallAnalysisGroundTruth {
		assert.NotEmpty(t, gt.FixtureFile, "FixtureFile 不能为空")
		assert.NotEmpty(t, gt.Edges, "%s 的 Edges 不能为空", gt.FixtureFile)
	}
}

func TestCallAnalysisGroundTruth_HasChains(t *testing.T) {
	totalChains := 0
	for _, gt := range CallAnalysisGroundTruth {
		totalChains += len(gt.Chains)
	}
	assert.Greater(t, totalChains, 0, "应至少有一条调用链真值")
}

func TestCallAnalysisGroundTruth_EdgeFieldsValid(t *testing.T) {
	for _, gt := range CallAnalysisGroundTruth {
		for _, e := range gt.Edges {
			assert.NotEmpty(t, e.SourceName, "%s: Edge.SourceName 不能为空", gt.FixtureFile)
			assert.NotEmpty(t, e.EdgeType, "%s: Edge.EdgeType 不能为空", gt.FixtureFile)
			// target_name 为空的情形（合法悬空边）：
			//   - import 边指向外部依赖（无对应符号），target_name 回退到 target_module（多数为空）。
			//   - Optional=true 的 call/extends 等边，target 为标准库/外部运行时函数
			//     （strlen/printf/UIViewController 等），跨文件消解后仍悬空——这是合法状态，
			//     真值标注它以免拉低 precision。
			if e.EdgeType == "import" || e.Optional {
				continue
			}
			assert.NotEmpty(t, e.TargetName, "%s: Edge.TargetName 不能为空（非 Optional 边应可消解）", gt.FixtureFile)
		}
	}
}

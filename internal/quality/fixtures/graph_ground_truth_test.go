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
			assert.NotEmpty(t, e.TargetName, "%s: Edge.TargetName 不能为空", gt.FixtureFile)
		}
	}
}

package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRetrievalGroundTruth_Coverage(t *testing.T) {
	assert.GreaterOrEqual(t, len(RetrievalGroundTruths), 6, "至少 6 个 query 真值")

	crossLang := 0
	singleLang := 0
	for _, gt := range RetrievalGroundTruths {
		assert.NotEmpty(t, gt.Query)
		assert.NotEmpty(t, gt.RelevantSymbols)
		if len(gt.Repos) > 1 {
			crossLang++
		} else {
			singleLang++
		}
	}
	assert.Greater(t, crossLang, 0, "至少一个跨语言 query")
	assert.Greater(t, singleLang, 0, "至少一个单语言 query")
}

package quality

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourtionguo/CodeAtlas/internal/retrieval"
)

func TestEvaluate_FixtureMode_RunsBothCategories(t *testing.T) {
	graphEval := NewGraphEvaluator(&stubGraphFetcher{
		byType: map[string]int{"call": 10}, dangling: map[string]int{"call": 1},
		orphans: 2, crossFile: 5, totalSymbols: 20,
		chainOK: 9, chainTotal: 10,
		extracted: []ExtractedEdge{{SourceName: "A", EdgeType: "call", TargetName: "B"}},
	}, &GraphGroundTruth{Edges: []ExpectedEdge{{SourceName: "A", EdgeType: "call", TargetName: "B"}}})

	retrievalEval := NewRetrievalEvaluator(&stubRetrievalRunner{
		blocksByQueryMode: map[string]map[string][]retrieval.ContextBlock{},
	}, nil, []string{"hybrid"})

	report, err := Evaluate(context.Background(), EvaluateConfig{
		Mode:         EvalModeFixture,
		FixtureSet:   "test",
		RunRetrieval: true,
		RepoID:       "repo-1",
	}, graphEval, retrievalEval)
	require.NoError(t, err)

	assert.Equal(t, EvalModeFixture, report.Mode)
	assert.Equal(t, "test", report.FixtureSet)
	assert.NotEmpty(t, report.Metrics)
	assert.Equal(t, len(report.Metrics), report.Summary.Total)
}

func TestEvaluate_RepoMode_SkipsRetrieval(t *testing.T) {
	graphEval := NewGraphEvaluator(&stubGraphFetcher{
		byType: map[string]int{"call": 10}, totalSymbols: 5,
	}, nil)

	report, err := Evaluate(context.Background(), EvaluateConfig{
		Mode:   EvalModeRepo,
		RepoID: "repo-1",
	}, graphEval, nil)
	require.NoError(t, err)

	assert.Equal(t, EvalModeRepo, report.Mode)
	for _, m := range report.Metrics {
		assert.Equal(t, CategoryGraph, m.Category, "repo 模式不应有检索指标")
	}
}

func TestReport_JSONMarshal(t *testing.T) {
	r := &Report{
		Mode: EvalModeFixture,
		Metrics: []MetricValue{
			{Name: "recall", Category: CategoryRetrieval, Value: 0.75, Threshold: 0.7, HigherIsBetter: true, Passed: true},
		},
	}
	r.Summary = ComputeSummary(r.Metrics)

	data, err := r.JSONMarshal()
	require.NoError(t, err)

	var back Report
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, "recall", back.Metrics[0].Name)
	assert.Equal(t, 0.75, back.Metrics[0].Value)
}

func TestReport_OverrideThreshold(t *testing.T) {
	r := &Report{
		Metrics: []MetricValue{
			{Name: "recall", Value: 0.65, Threshold: 0.70, HigherIsBetter: true, Passed: false},
		},
	}
	r.Summary = ComputeSummary(r.Metrics)
	assert.Equal(t, 1, r.Summary.Failed)

	r.OverrideThreshold("recall", 0.60)
	assert.True(t, r.Metrics[0].Passed)
	assert.Equal(t, 0, r.Summary.Failed)
}

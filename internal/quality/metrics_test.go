package quality

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricValue_IsPassed(t *testing.T) {
	tests := []struct {
		name       string
		mv         MetricValue
		wantPassed bool
	}{
		{
			name:       "无阈值恒通过",
			mv:         MetricValue{Name: "observed", Value: 0.99, Threshold: 0},
			wantPassed: true,
		},
		{
			name:       "越高越好的指标达标",
			mv:         MetricValue{Name: "recall", Value: 0.75, Threshold: 0.70, HigherIsBetter: true},
			wantPassed: true,
		},
		{
			name:       "越高越好的指标未达标",
			mv:         MetricValue{Name: "recall", Value: 0.65, Threshold: 0.70, HigherIsBetter: true},
			wantPassed: false,
		},
		{
			name:       "越低越好的指标达标",
			mv:         MetricValue{Name: "dangling", Value: 0.20, Threshold: 0.30, HigherIsBetter: false},
			wantPassed: true,
		},
		{
			name:       "越低越好的指标未达标",
			mv:         MetricValue{Name: "dangling", Value: 0.35, Threshold: 0.30, HigherIsBetter: false},
			wantPassed: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mv.EvaluatePassed()
			assert.Equal(t, tt.wantPassed, tt.mv.Passed)
		})
	}
}

func TestSummary_ComputeFromMetrics(t *testing.T) {
	metrics := []MetricValue{
		{Name: "a", Threshold: 0.7, Value: 0.8, HigherIsBetter: true, Passed: true},
		{Name: "b", Threshold: 0.7, Value: 0.6, HigherIsBetter: true, Passed: false},
		{Name: "c", Threshold: 0, Value: 0.5, Passed: true}, // 仅观察
	}
	s := ComputeSummary(metrics)
	assert.Equal(t, 3, s.Total)
	assert.Equal(t, 2, s.Passed)
	assert.Equal(t, 1, s.Failed)
	assert.Equal(t, 1, s.NoThreshold)
}

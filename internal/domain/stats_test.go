package domain

import "testing"

func TestMetric_Valid(t *testing.T) {
	for _, m := range []Metric{MetricCoins, MetricCollection, MetricValue, MetricStars} {
		if !m.Valid() {
			t.Errorf("%s should be valid", m)
		}
	}
	if Metric("fame").Valid() || Metric("").Valid() {
		t.Error("unknown metrics must be invalid")
	}
}

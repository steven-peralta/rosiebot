package domain

import "time"

type RollRecord struct {
	Slug string
	Name string
	Kind RollKind
	Cost int64
	At   time.Time
}

type Metric string

const (
	MetricCoins      Metric = "coins"
	MetricCollection Metric = "collection"
	MetricValue      Metric = "value"
	MetricStars      Metric = "stars"
)

func (m Metric) Valid() bool {
	switch m {
	case MetricCoins, MetricCollection, MetricValue, MetricStars:
		return true
	}
	return false
}

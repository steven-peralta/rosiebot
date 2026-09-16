package app

import (
	"sort"
	"strings"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type SortField string

const (
	SortNone  SortField = ""
	SortLikes SortField = "likes"
	SortTrash SortField = "trash"
	SortTotal SortField = "total"
	SortName  SortField = "name"
	SortRank  SortField = "rank"
	SortStars SortField = "stars"
)

type RankLookup func(slug string) (domain.RankedWaifu, bool)

func NoRanking(string) (domain.RankedWaifu, bool) { return domain.RankedWaifu{}, false }

func LookupFrom(p RankingProvider) RankLookup {
	return func(slug string) (domain.RankedWaifu, bool) {
		if p == nil {
			return domain.RankedWaifu{}, false
		}
		return p.Current().Lookup(slug)
	}
}

type Filter struct {
	Field SortField
	Op    string
	Value int
}

type Query struct {
	Term       string
	SortBy     SortField
	Descending bool
	Filters    []Filter
}

func (q Query) Empty() bool { return strings.TrimSpace(q.Term) == "" }

const OpAbsent = "none"

func (q Query) WithFilter(field SortField, op string, value int) Query {
	q.Filters = append(q.Filters, Filter{Field: field, Op: op, Value: value})
	return q
}

func (q Query) WantsUnranked() bool {
	for _, f := range q.Filters {
		if f.Field == SortRank && f.Op == OpAbsent {
			return true
		}
	}
	return false
}

func fieldValue(w domain.WaifuSummary, f SortField, lookup RankLookup) (int, bool) {
	switch f {
	case SortLikes:
		return w.Likes, true
	case SortTrash:
		return w.Trash, true
	case SortTotal:
		return w.TotalVotes(), true
	case SortRank:
		if r, ok := lookup(w.Slug); ok {
			return r.Position, true
		}
		return 0, false
	case SortStars:
		if r, ok := lookup(w.Slug); ok {
			return r.Stars, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func (f Filter) matches(w domain.WaifuSummary, lookup RankLookup) bool {
	v, ok := fieldValue(w, f.Field, lookup)
	if f.Op == OpAbsent {
		return !ok
	}
	if !ok {
		return false
	}
	switch f.Op {
	case "<":
		return v < f.Value
	case "<=":
		return v <= f.Value
	case ">":
		return v > f.Value
	case ">=":
		return v >= f.Value
	default:
		return v == f.Value
	}
}

func (q Query) Apply(items []domain.WaifuSummary, lookup RankLookup) []domain.WaifuSummary {
	if lookup == nil {
		lookup = NoRanking
	}
	out := make([]domain.WaifuSummary, 0, len(items))
	for _, it := range items {
		keep := true
		for _, f := range q.Filters {
			if !f.matches(it, lookup) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, it)
		}
	}
	if q.SortBy == SortNone {
		return out
	}
	if q.SortBy == SortName {
		sort.SliceStable(out, func(i, j int) bool {
			a, b := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
			if q.Descending {
				return b < a
			}
			return a < b
		})
		return out
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, aok := fieldValue(out[i], q.SortBy, lookup)
		b, bok := fieldValue(out[j], q.SortBy, lookup)
		if aok != bok {
			return aok
		}
		if q.Descending {
			return b < a
		}
		return a < b
	})
	return out
}

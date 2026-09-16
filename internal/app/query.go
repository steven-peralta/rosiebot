package app

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

var ErrBadQuery = errors.New("bad query")

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

func (q Query) Empty() bool { return q.Term == "" }

func ParseQuery(raw string) (Query, error) {
	var q Query
	var words []string
	for _, tok := range strings.Fields(raw) {
		field, value, hasColon := strings.Cut(tok, ":")
		if !hasColon {
			words = append(words, tok)
			continue
		}
		field = strings.ToLower(field)
		if field == "sortby" {
			desc := false
			switch {
			case strings.HasPrefix(value, "-"):
				desc, value = true, value[1:]
			case strings.HasPrefix(value, "+"):
				value = value[1:]
			}
			sf, ok := parseSortField(value)
			if !ok {
				return Query{}, fmt.Errorf("%w: unknown sort field %q (use likes, trash, total, rank, stars, or name)", ErrBadQuery, value)
			}
			q.SortBy, q.Descending = sf, desc
			continue
		}
		sf, ok := parseSortField(field)
		if !ok || sf == SortName {
			return Query{}, fmt.Errorf("%w: unknown filter %q (use likes, trash, total, rank, or stars)", ErrBadQuery, field)
		}
		op := "="
		for _, candidate := range []string{"<=", ">=", "<", ">", "="} {
			if strings.HasPrefix(value, candidate) {
				op, value = candidate, value[len(candidate):]
				break
			}
		}
		n, err := strconv.Atoi(value)
		if err != nil {
			return Query{}, fmt.Errorf("%w: %s needs a number, got %q", ErrBadQuery, field, value)
		}
		q.Filters = append(q.Filters, Filter{Field: sf, Op: op, Value: n})
	}
	q.Term = strings.Join(words, " ")
	return q, nil
}

func parseSortField(s string) (SortField, bool) {
	switch strings.ToLower(s) {
	case "likes", "like":
		return SortLikes, true
	case "trash":
		return SortTrash, true
	case "total", "votes":
		return SortTotal, true
	case "name":
		return SortName, true
	case "rank", "position":
		return SortRank, true
	case "stars", "star", "tier":
		return SortStars, true
	default:
		return SortNone, false
	}
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

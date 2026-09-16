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
)

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
				return Query{}, fmt.Errorf("%w: unknown sort field %q (use likes, trash, total, or name)", ErrBadQuery, value)
			}
			q.SortBy, q.Descending = sf, desc
			continue
		}
		sf, ok := parseSortField(field)
		if !ok || sf == SortName {
			return Query{}, fmt.Errorf("%w: unknown filter %q (use likes, trash, or total)", ErrBadQuery, field)
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
	default:
		return SortNone, false
	}
}

func fieldValue(w domain.WaifuSummary, f SortField) int {
	switch f {
	case SortLikes:
		return w.Likes
	case SortTrash:
		return w.Trash
	case SortTotal:
		return w.TotalVotes()
	default:
		return 0
	}
}

func (f Filter) matches(w domain.WaifuSummary) bool {
	v := fieldValue(w, f.Field)
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

func (q Query) Apply(items []domain.WaifuSummary) []domain.WaifuSummary {
	out := make([]domain.WaifuSummary, 0, len(items))
	for _, it := range items {
		keep := true
		for _, f := range q.Filters {
			if !f.matches(it) {
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
	less := func(a, b domain.WaifuSummary) bool {
		if q.SortBy == SortName {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return fieldValue(a, q.SortBy) < fieldValue(b, q.SortBy)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if q.Descending {
			return less(out[j], out[i])
		}
		return less(out[i], out[j])
	})
	return out
}

package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	MaxSearchPages      = 3
	MaxListPages        = 3
	MaxSeriesCharPages  = 10
	searchTermMaxLength = 100
)

type SeriesResult struct {
	Series domain.Series
	Waifus []domain.WaifuSummary
}

type SearchService struct {
	source  WaifuSource
	ranking RankingProvider
}

func NewSearchService(source WaifuSource, ranking RankingProvider) *SearchService {
	return &SearchService{source: source, ranking: ranking}
}

func (s *SearchService) Waifus(ctx context.Context, query Query) ([]domain.WaifuSummary, error) {
	query.Term = cleanTerm(query.Term)
	var (
		results []domain.WaifuSummary
		err     error
	)
	if query.Empty() {
		results, err = s.collect(ctx, MaxListPages, func(page int) (SearchPage, error) { return s.source.ListCharacters(ctx, page) })
		if err != nil {
			return nil, fmt.Errorf("list characters: %w", err)
		}
	} else {
		results, err = s.collect(ctx, MaxSearchPages, func(page int) (SearchPage, error) { return s.source.SearchWaifus(ctx, query.Term, page) })
		if err != nil {
			return nil, fmt.Errorf("search waifus %q: %w", query.Term, err)
		}
	}
	results = query.Apply(results, LookupFrom(s.ranking))
	if len(results) == 0 {
		return nil, ErrNotFound
	}
	return results, nil
}

func (s *SearchService) collect(_ context.Context, maxPages int, fetch func(page int) (SearchPage, error)) ([]domain.WaifuSummary, error) {
	var results []domain.WaifuSummary
	for page := 1; page <= maxPages; page++ {
		res, err := fetch(page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", page, err)
		}
		results = append(results, res.Items...)
		if page >= res.LastPage || len(res.Items) == 0 {
			break
		}
	}
	return results, nil
}

func (s *SearchService) Random(ctx context.Context) (domain.Waifu, error) {
	summary, err := s.source.Random(ctx)
	if err != nil {
		return domain.Waifu{}, fmt.Errorf("random waifu: %w", err)
	}
	return s.source.Get(ctx, summary.Slug)
}

func (s *SearchService) Detail(ctx context.Context, slug string) (domain.Waifu, error) {
	return s.source.Get(ctx, slug)
}

func (s *SearchService) Suggest(prefix string, limit int) []domain.RankedWaifu {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if s.ranking == nil {
		return nil
	}
	current := s.ranking.Current()
	if current == nil || limit <= 0 {
		return nil
	}
	var out []domain.RankedWaifu
	for _, row := range current.Rows() {
		if prefix != "" && !matchesName(row.WaifuSummary, prefix) {
			continue
		}
		out = append(out, row)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func matchesName(w domain.WaifuSummary, prefix string) bool {
	for _, n := range []string{w.Name, w.OriginalName, w.RomajiName, w.Slug} {
		if n != "" && strings.Contains(strings.ToLower(n), prefix) {
			return true
		}
	}
	return false
}

func (s *SearchService) Series(ctx context.Context, term string) (SeriesResult, error) {
	term = cleanTerm(term)
	if term == "" {
		return SeriesResult{}, ErrNotFound
	}
	works, err := s.source.SearchWorks(ctx, term)
	if err != nil {
		return SeriesResult{}, fmt.Errorf("search works %q: %w", term, err)
	}
	if len(works) == 0 {
		return SeriesResult{}, ErrNotFound
	}
	series := works[0]

	var waifus []domain.WaifuSummary
	for page := 1; page <= MaxSeriesCharPages; page++ {
		res, err := s.source.WorkCharacters(ctx, series.Slug, page)
		if err != nil {
			return SeriesResult{}, fmt.Errorf("characters of %q page %d: %w", series.Slug, page, err)
		}
		waifus = append(waifus, res.Items...)
		if page >= res.LastPage || len(res.Items) == 0 {
			break
		}
	}
	sort.SliceStable(waifus, func(i, j int) bool { return waifus[i].Likes > waifus[j].Likes })
	return SeriesResult{Series: series, Waifus: waifus}, nil
}

func cleanTerm(term string) string {
	term = strings.TrimSpace(term)
	if len(term) > searchTermMaxLength {
		term = term[:searchTermMaxLength]
	}
	return term
}

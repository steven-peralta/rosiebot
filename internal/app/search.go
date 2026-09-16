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
	MaxSeriesCharPages  = 10
	searchTermMaxLength = 100
)

type SeriesResult struct {
	Series domain.Series
	Waifus []domain.WaifuSummary
}

type SearchService struct {
	source WaifuSource
}

func NewSearchService(source WaifuSource) *SearchService {
	return &SearchService{source: source}
}

func (s *SearchService) Waifus(ctx context.Context, term string) ([]domain.WaifuSummary, error) {
	term = cleanTerm(term)
	if term == "" {
		return nil, ErrNotFound
	}
	var results []domain.WaifuSummary
	for page := 1; page <= MaxSearchPages; page++ {
		res, err := s.source.SearchWaifus(ctx, term, page)
		if err != nil {
			return nil, fmt.Errorf("search waifus %q page %d: %w", term, page, err)
		}
		results = append(results, res.Items...)
		if page >= res.LastPage || len(res.Items) == 0 {
			break
		}
	}
	if len(results) == 0 {
		return nil, ErrNotFound
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

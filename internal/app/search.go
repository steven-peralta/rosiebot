package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	MaxSearchPages      = 3
	MaxListPages        = 3
	MaxSeriesCharPages  = 10
	MaxBrowseResults    = 500
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
	query = query.WithDefaultSort(SortRank, false)
	if query.Empty() {
		if ranked := s.rankedSummaries(); len(ranked) > 0 && !query.WantsUnranked() {
			results = ranked
		} else {
			results, err = s.collect(ctx, MaxListPages, func(page int) (SearchPage, error) { return s.source.ListCharacters(ctx, page) })
			if err != nil {
				return nil, fmt.Errorf("list characters: %w", err)
			}
		}
	} else {
		results, err = s.collect(ctx, MaxSearchPages, func(page int) (SearchPage, error) { return s.source.SearchWaifus(ctx, query.Term, page) })
		if err != nil {
			return nil, fmt.Errorf("search waifus %q: %w", query.Term, err)
		}
	}
	found := len(results)
	results = query.Apply(results, LookupFrom(s.ranking))
	if len(results) == 0 {
		if found > 0 {
			return nil, &FilteredOutError{Found: found}
		}
		return nil, ErrNotFound
	}
	if len(results) > MaxBrowseResults {
		results = results[:MaxBrowseResults]
	}
	return results, nil
}

func (s *SearchService) rankedSummaries() []domain.WaifuSummary {
	if s.ranking == nil {
		return nil
	}
	current := s.ranking.Current()
	if current.Len() == 0 {
		return nil
	}
	rows := current.Rows()
	out := make([]domain.WaifuSummary, len(rows))
	for i, r := range rows {
		out[i] = r.WaifuSummary
	}
	return out
}

func (s *SearchService) Matches(w domain.WaifuSummary, query Query) bool {
	return len(query.Apply([]domain.WaifuSummary{w}, LookupFrom(s.ranking))) == 1
}

func (s *SearchService) Rank(slug string) (domain.RankedWaifu, bool) {
	return LookupFrom(s.ranking)(slug)
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

const minSeriesSuggestLength = 2

func (s *SearchService) Series(ctx context.Context, term string, q Query) (SeriesResult, error) {
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
	return s.charactersOf(ctx, works[0], q)
}

func (s *SearchService) SeriesBySlug(ctx context.Context, slug string, q Query) (SeriesResult, error) {
	series, err := s.source.Work(ctx, slug)
	if err != nil {
		return SeriesResult{}, err
	}
	return s.charactersOf(ctx, series, q)
}

func (s *SearchService) SuggestSeries(ctx context.Context, term string, limit int) ([]domain.Series, error) {
	term = cleanTerm(term)
	if len([]rune(term)) < minSeriesSuggestLength || limit <= 0 {
		return nil, nil
	}
	works, err := s.source.SearchWorks(ctx, term)
	if err != nil {
		return nil, fmt.Errorf("suggest series %q: %w", term, err)
	}
	if len(works) > limit {
		works = works[:limit]
	}
	return works, nil
}

func (s *SearchService) charactersOf(ctx context.Context, series domain.Series, q Query) (SeriesResult, error) {
	waifus, err := s.collect(ctx, MaxSeriesCharPages, func(page int) (SearchPage, error) {
		return s.source.WorkCharacters(ctx, series.Slug, page)
	})
	if err != nil {
		return SeriesResult{}, fmt.Errorf("characters of %q: %w", series.Slug, err)
	}
	q = q.WithDefaultSort(SortRank, false)
	if term := strings.ToLower(cleanTerm(q.Term)); term != "" {
		kept := waifus[:0]
		for _, w := range waifus {
			if matchesName(w, term) {
				kept = append(kept, w)
			}
		}
		waifus = kept
	}
	found := len(waifus)
	waifus = q.Apply(waifus, LookupFrom(s.ranking))
	if len(waifus) == 0 && found > 0 {
		return SeriesResult{Series: series}, &FilteredOutError{Found: found}
	}
	return SeriesResult{Series: series, Waifus: waifus}, nil
}

func cleanTerm(term string) string {
	term = strings.TrimSpace(term)
	if len(term) > searchTermMaxLength {
		term = term[:searchTermMaxLength]
	}
	return term
}

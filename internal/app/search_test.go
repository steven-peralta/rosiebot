package app_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestSearchService_EmptyTermListsCatalog(t *testing.T) {
	f := newFixture(t)
	for page := 1; page <= app.MaxListPages; page++ {
		f.source.EXPECT().ListCharacters(mock.Anything, page).Return(app.SearchPage{Page: page, LastPage: 5113, Items: []domain.WaifuSummary{summary(fmt.Sprintf("c%d", page), page*10, 1)}}, nil).Once()
	}
	got, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, "   ")
	if err != nil || len(got) != app.MaxListPages || got[0].Slug != "c1" {
		t.Fatalf("empty query = %v, %v", got, err)
	}
	f.source.AssertNotCalled(t, "SearchWaifus", mock.Anything, mock.Anything, mock.Anything)

	f.source.EXPECT().ListCharacters(mock.Anything, 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, ""); err == nil {
		t.Error("list error should propagate")
	}
}

func TestSearchService_SortAndFilterTokens(t *testing.T) {
	f := newFixture(t)
	items := []domain.WaifuSummary{summary("a", 50, 5), summary("b", 500, 10), summary("c", 5, 0)}
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: items}, nil).Times(3)
	svc := app.NewSearchService(f.source, f.ranking)

	got, err := svc.Waifus(f.ctx, "rem sortby:-likes")
	if err != nil || got[0].Slug != "b" || got[2].Slug != "c" {
		t.Errorf("sortby:-likes = %v %v", got, err)
	}
	got, err = svc.Waifus(f.ctx, "sortby:+trash rem")
	if err != nil || got[0].Slug != "c" || got[2].Slug != "b" {
		t.Errorf("sortby:+trash = %v %v", got, err)
	}
	got, err = svc.Waifus(f.ctx, "rem likes:>=50 trash:<10")
	if err != nil || len(got) != 1 || got[0].Slug != "a" {
		t.Errorf("filters = %v %v", got, err)
	}

	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: items}, nil).Once()
	if _, err := svc.Waifus(f.ctx, "rem likes:>1000"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("filter matching nothing = %v", err)
	}
	f.ranking.Set(rankingOf(10))
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("x", 1, 0), summary("ranked-003", 1, 0)}}, nil).Once()
	got, err = svc.Waifus(f.ctx, "rem sortby:rank")
	if err != nil || got[0].Slug != "ranked-003" {
		t.Errorf("service rank sort = %v %v", got, err)
	}
	if _, err := svc.Waifus(f.ctx, "rem sortby:height"); !errors.Is(err, app.ErrBadQuery) {
		t.Errorf("bad sort field = %v", err)
	}
	if _, err := svc.Waifus(f.ctx, "rem likes:many"); !errors.Is(err, app.ErrBadQuery) {
		t.Errorf("bad filter value = %v", err)
	}
	if _, err := svc.Waifus(f.ctx, "rem name:rem"); !errors.Is(err, app.ErrBadQuery) {
		t.Errorf("name filter = %v", err)
	}
}

func TestParseQuery(t *testing.T) {
	q, err := app.ParseQuery("  shinji  ikari sortby:-total likes:>=100 trash:<5 total:=200 votes:>1 ")
	if err != nil {
		t.Fatal(err)
	}
	if q.Term != "shinji ikari" || q.SortBy != app.SortTotal || !q.Descending || len(q.Filters) != 4 || q.Filters[2].Op != "=" || q.Filters[2].Value != 200 {
		t.Errorf("parsed = %+v", q)
	}
	ranked := rankingOf(100)
	rankedQ, err := app.ParseQuery("sortby:rank")
	if err != nil {
		t.Fatal(err)
	}
	mixed := []domain.WaifuSummary{summary("nobody", 1, 0), summary("ranked-050", 1, 0), summary("ranked-002", 1, 0)}
	byRank := rankedQ.Apply(mixed, app.LookupFrom(memory.NewRankingHolder(ranked)))
	if byRank[0].Slug != "ranked-002" || byRank[1].Slug != "ranked-050" || byRank[2].Slug != "nobody" {
		t.Errorf("rank sort = %v", byRank)
	}
	rankedQ, _ = app.ParseQuery("sortby:-rank")
	byRank = rankedQ.Apply(mixed, app.LookupFrom(memory.NewRankingHolder(ranked)))
	if byRank[0].Slug != "ranked-050" || byRank[2].Slug != "nobody" {
		t.Errorf("desc rank sort should still put unranked last: %v", byRank)
	}
	starsQ, _ := app.ParseQuery("tier:4 sortby:-stars")
	if got := starsQ.Apply(mixed, app.LookupFrom(memory.NewRankingHolder(ranked))); len(got) != 1 || got[0].Slug != "ranked-002" {
		t.Errorf("stars filter = %v", got)
	}
	if got := starsQ.Apply(mixed, nil); len(got) != 0 {
		t.Errorf("rank filters without a ranking match nothing: %v", got)
	}
	if got := starsQ.Apply(mixed, app.LookupFrom(nil)); len(got) != 0 {
		t.Errorf("nil provider matches nothing: %v", got)
	}
	rankQ, _ := app.ParseQuery("rank:<=10")
	if got := rankQ.Apply(mixed, app.LookupFrom(memory.NewRankingHolder(ranked))); len(got) != 1 || got[0].Slug != "ranked-002" {
		t.Errorf("rank filter = %v", got)
	}

	q, err = app.ParseQuery("sortby:name")
	if err != nil || q.SortBy != app.SortName || q.Descending || !q.Empty() {
		t.Errorf("name sort = %+v %v", q, err)
	}
	sorted := q.Apply([]domain.WaifuSummary{{Name: "b"}, {Name: "A"}, {Name: "c"}}, nil)
	if sorted[0].Name != "A" || sorted[2].Name != "c" {
		t.Errorf("name sort order = %v", sorted)
	}
	q, _ = app.ParseQuery("sortby:-name")
	sorted = q.Apply([]domain.WaifuSummary{{Name: "b"}, {Name: "A"}, {Name: "c"}}, nil)
	if sorted[0].Name != "c" || sorted[2].Name != "A" {
		t.Errorf("desc name sort order = %v", sorted)
	}
	if _, err := app.ParseQuery("SORTBY:LIKE"); err != nil {
		t.Errorf("case-insensitive sort: %v", err)
	}
}

func TestSearchService_WaifusNoResults(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "nobody", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, "nobody"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestSearchService_WaifusWalksPagesUpToCap(t *testing.T) {
	f := newFixture(t)
	for page := 1; page <= app.MaxSearchPages; page++ {
		f.source.EXPECT().SearchWaifus(mock.Anything, "rem", page).Return(app.SearchPage{
			Page: page, LastPage: app.MaxSearchPages + 5,
			Items: []domain.WaifuSummary{summary("p"+string(rune('0'+page)), page, 0)},
		}, nil).Once()
	}
	got, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, "  rem ")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != app.MaxSearchPages {
		t.Errorf("got %d results, want one per page up to the cap", len(got))
	}
}

func TestSearchService_WaifusStopsAtLastPage(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("rem", 1, 0)}}, nil).Once()
	got, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, "rem")
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestSearchService_WaifusTruncatesLongTerms(t *testing.T) {
	f := newFixture(t)
	long := strings.Repeat("a", 150)
	f.source.EXPECT().SearchWaifus(mock.Anything, strings.Repeat("a", 100), 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("a", 1, 0)}}, nil).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, long); err != nil {
		t.Fatal(err)
	}
}

func TestSearchService_WaifusPropagatesErrors(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, "rem"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSearchService_RandomFetchesDetail(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()
	got, err := app.NewSearchService(f.source, f.ranking).Random(f.ctx)
	if err != nil || got.Slug != "rem" {
		t.Fatalf("got %+v, %v", got, err)
	}

	f.source.EXPECT().Random(mock.Anything).Return(domain.WaifuSummary{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Random(f.ctx); err == nil {
		t.Fatal("expected error")
	}

	f.source.EXPECT().Get(mock.Anything, "ram").Return(detail("ram"), nil).Once()
	if got, err := app.NewSearchService(f.source, f.ranking).Detail(f.ctx, "ram"); err != nil || got.Slug != "ram" {
		t.Fatalf("Detail = %+v, %v", got, err)
	}
}

func TestSearchService_SeriesNotFound(t *testing.T) {
	f := newFixture(t)
	svc := app.NewSearchService(f.source, f.ranking)
	if _, err := svc.Series(f.ctx, ""); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("empty term err = %v", err)
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "nothing").Return(nil, nil).Once()
	if _, err := svc.Series(f.ctx, "nothing"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("no works err = %v", err)
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	if _, err := svc.Series(f.ctx, "boom"); err == nil {
		t.Error("expected error")
	}
}

func TestSearchService_SeriesSortsCharactersByLikesAcrossPages(t *testing.T) {
	f := newFixture(t)
	series := domain.Series{Slug: "re-zero", Name: "Re:Zero"}
	f.source.EXPECT().SearchWorks(mock.Anything, "re zero").Return([]domain.Series{series, {Slug: "other"}}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 2, Items: []domain.WaifuSummary{summary("ram", 100, 0), summary("emilia", 300, 0)}}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 2).Return(app.SearchPage{Page: 2, LastPage: 2, Items: []domain.WaifuSummary{summary("rem", 900, 0)}}, nil).Once()

	res, err := app.NewSearchService(f.source, f.ranking).Series(f.ctx, "re zero")
	if err != nil {
		t.Fatal(err)
	}
	if res.Series.Slug != "re-zero" {
		t.Errorf("series = %+v", res.Series)
	}
	want := []string{"rem", "emilia", "ram"}
	for i, w := range want {
		if res.Waifus[i].Slug != w {
			t.Errorf("waifus[%d] = %s, want %s", i, res.Waifus[i].Slug, w)
		}
	}
}

func TestSearchService_SeriesCharacterErrorPropagates(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWorks(mock.Anything, "x").Return([]domain.Series{{Slug: "x"}}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "x", 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Series(f.ctx, "x"); err == nil {
		t.Fatal("expected error")
	}
}

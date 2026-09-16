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
	got, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, app.Query{Term: "   "})
	if err != nil || len(got) != app.MaxListPages || got[0].Slug != "c1" {
		t.Fatalf("empty query = %v, %v", got, err)
	}
	f.source.AssertNotCalled(t, "SearchWaifus", mock.Anything, mock.Anything, mock.Anything)

	f.source.EXPECT().ListCharacters(mock.Anything, 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, app.Query{}); err == nil {
		t.Error("list error should propagate")
	}
}

func TestSearchService_SortAndFilters(t *testing.T) {
	f := newFixture(t)
	items := []domain.WaifuSummary{summary("a", 50, 5), summary("b", 500, 10), summary("c", 5, 0)}
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: items}, nil).Times(5)
	svc := app.NewSearchService(f.source, f.ranking)

	got, err := svc.Waifus(f.ctx, app.Query{Term: "rem", SortBy: app.SortLikes, Descending: true})
	if err != nil || got[0].Slug != "b" || got[2].Slug != "c" {
		t.Errorf("likes desc = %v %v", got, err)
	}
	got, err = svc.Waifus(f.ctx, app.Query{Term: "rem", SortBy: app.SortTrash})
	if err != nil || got[0].Slug != "c" || got[2].Slug != "b" {
		t.Errorf("trash asc = %v %v", got, err)
	}
	got, err = svc.Waifus(f.ctx, app.Query{Term: "rem"}.WithFilter(app.SortLikes, ">=", 50).WithFilter(app.SortTrash, "<", 10))
	if err != nil || len(got) != 1 || got[0].Slug != "a" {
		t.Errorf("filters = %v %v", got, err)
	}
	if _, err := svc.Waifus(f.ctx, app.Query{Term: "rem"}.WithFilter(app.SortLikes, ">", 1000)); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("filter matching nothing = %v", err)
	}

	f.ranking.Set(rankingOf(10))
	got, err = svc.Waifus(f.ctx, app.Query{Term: "rem", SortBy: app.SortRank})
	if err != nil || len(got) != 3 {
		t.Fatalf("rank sort without ranked results = %v %v", got, err)
	}
	f.source.EXPECT().SearchWaifus(mock.Anything, "ranked", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("x", 1, 0), summary("ranked-003", 1, 0)}}, nil).Times(2)
	got, err = svc.Waifus(f.ctx, app.Query{Term: "ranked", SortBy: app.SortRank})
	if err != nil || got[0].Slug != "ranked-003" {
		t.Errorf("service rank sort = %v %v", got, err)
	}
	got, err = svc.Waifus(f.ctx, app.Query{Term: "ranked"}.WithFilter(app.SortRank, ">=", 1))
	if err != nil || len(got) != 1 || got[0].Slug != "ranked-003" {
		t.Errorf("ranked-only filter = %v %v", got, err)
	}
}

func TestQuery_Apply(t *testing.T) {
	ranked := rankingOf(100)
	lookup := app.LookupFrom(memory.NewRankingHolder(ranked))
	mixed := []domain.WaifuSummary{summary("nobody", 1, 0), summary("ranked-050", 1, 0), summary("ranked-002", 1, 0)}

	byRank := app.Query{SortBy: app.SortRank}.Apply(mixed, lookup)
	if byRank[0].Slug != "ranked-002" || byRank[1].Slug != "ranked-050" || byRank[2].Slug != "nobody" {
		t.Errorf("rank sort = %v", byRank)
	}
	byRank = app.Query{SortBy: app.SortRank, Descending: true}.Apply(mixed, lookup)
	if byRank[0].Slug != "ranked-050" || byRank[2].Slug != "nobody" {
		t.Errorf("desc rank sort should still put unranked last: %v", byRank)
	}
	stars := app.Query{SortBy: app.SortStars, Descending: true}.WithFilter(app.SortStars, ">=", 4)
	if got := stars.Apply(mixed, lookup); len(got) != 1 || got[0].Slug != "ranked-002" {
		t.Errorf("stars filter = %v", got)
	}
	if got := stars.Apply(mixed, nil); len(got) != 0 {
		t.Errorf("rank filters without a ranking match nothing: %v", got)
	}
	if got := stars.Apply(mixed, app.LookupFrom(nil)); len(got) != 0 {
		t.Errorf("nil provider matches nothing: %v", got)
	}
	if got := (app.Query{}).WithFilter(app.SortRank, "<=", 10).Apply(mixed, lookup); len(got) != 1 || got[0].Slug != "ranked-002" {
		t.Errorf("rank filter = %v", got)
	}
	if got := (app.Query{}).WithFilter(app.SortTotal, "=", 1).Apply(mixed, lookup); len(got) != 3 {
		t.Errorf("equality filter = %v", got)
	}
	names := []domain.WaifuSummary{{Name: "b"}, {Name: "A"}, {Name: "c"}}
	if got := (app.Query{SortBy: app.SortName}).Apply(names, nil); got[0].Name != "A" || got[2].Name != "c" {
		t.Errorf("name sort = %v", got)
	}
	if got := (app.Query{SortBy: app.SortName, Descending: true}).Apply(names, nil); got[0].Name != "c" || got[2].Name != "A" {
		t.Errorf("desc name sort = %v", got)
	}
	if !(app.Query{Term: "  "}).Empty() || (app.Query{Term: "x"}).Empty() {
		t.Error("Empty")
	}
}

func TestSearchService_Suggest(t *testing.T) {
	f := newFixture(t)
	svc := app.NewSearchService(f.source, f.ranking)
	if got := svc.Suggest("rank", 5); len(got) != 0 {
		t.Errorf("no ranking yet should suggest nothing: %v", got)
	}
	f.ranking.Set(rankingOf(50))
	got := svc.Suggest("RANKED-00", 3)
	if len(got) != 3 || got[0].Slug != "ranked-000" || got[2].Slug != "ranked-002" {
		t.Errorf("suggest = %v", got)
	}
	if got := svc.Suggest("", 2); len(got) != 2 {
		t.Errorf("empty prefix lists top ranked: %v", got)
	}
	if got := svc.Suggest("zzz", 5); len(got) != 0 {
		t.Errorf("no match = %v", got)
	}
	if got := svc.Suggest("x", 0); got != nil {
		t.Errorf("zero limit = %v", got)
	}
	if got := app.NewSearchService(f.source, nil).Suggest("x", 5); got != nil {
		t.Errorf("nil provider = %v", got)
	}
}

func TestSearchService_WaifusNoResults(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "nobody", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, app.Query{Term: "nobody"}); !errors.Is(err, app.ErrNotFound) {
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
	got, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, app.Query{Term: "  rem "})
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
	got, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, app.Query{Term: "rem"})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestSearchService_WaifusTruncatesLongTerms(t *testing.T) {
	f := newFixture(t)
	long := strings.Repeat("a", 150)
	f.source.EXPECT().SearchWaifus(mock.Anything, strings.Repeat("a", 100), 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("a", 1, 0)}}, nil).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, app.Query{Term: long}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchService_WaifusPropagatesErrors(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source, f.ranking).Waifus(f.ctx, app.Query{Term: "rem"}); err == nil {
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
	if _, err := svc.Series(f.ctx, "", app.Query{}); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("empty term err = %v", err)
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "nothing").Return(nil, nil).Once()
	if _, err := svc.Series(f.ctx, "nothing", app.Query{}); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("no works err = %v", err)
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	if _, err := svc.Series(f.ctx, "boom", app.Query{}); err == nil {
		t.Error("expected error")
	}
}

func TestSearchService_SeriesSortsCharactersByLikesAcrossPages(t *testing.T) {
	f := newFixture(t)
	series := domain.Series{Slug: "re-zero", Name: "Re:Zero"}
	f.source.EXPECT().SearchWorks(mock.Anything, "re zero").Return([]domain.Series{series, {Slug: "other"}}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 2, Items: []domain.WaifuSummary{summary("ram", 100, 0), summary("emilia", 300, 0)}}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 2).Return(app.SearchPage{Page: 2, LastPage: 2, Items: []domain.WaifuSummary{summary("rem", 900, 0)}}, nil).Once()

	res, err := app.NewSearchService(f.source, f.ranking).Series(f.ctx, "re zero", app.Query{})
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
	if _, err := app.NewSearchService(f.source, f.ranking).Series(f.ctx, "x", app.Query{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestSearchService_SeriesBySlugAndOptions(t *testing.T) {
	f := newFixture(t)
	svc := app.NewSearchService(f.source, f.ranking)
	series := domain.Series{Slug: "re-zero", Name: "Re:Zero"}
	f.source.EXPECT().Work(mock.Anything, "re-zero").Return(series, nil).Twice()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("ram", 100, 0), summary("rem", 900, 50), summary("emilia", 300, 0)}}, nil).Twice()

	res, err := svc.SeriesBySlug(f.ctx, "re-zero", app.Query{})
	if err != nil || res.Series.Slug != "re-zero" || res.Waifus[0].Slug != "rem" || res.Waifus[2].Slug != "ram" {
		t.Fatalf("by slug default order = %+v %v", res, err)
	}
	res, err = svc.SeriesBySlug(f.ctx, "re-zero", app.Query{SortBy: app.SortName}.WithFilter(app.SortTrash, "=", 0))
	if err != nil || len(res.Waifus) != 2 || res.Waifus[0].Slug != "emilia" {
		t.Errorf("by slug with options = %+v %v", res, err)
	}

	f.source.EXPECT().Work(mock.Anything, "ghost").Return(domain.Series{}, app.ErrNotFound).Once()
	if _, err := svc.SeriesBySlug(f.ctx, "ghost", app.Query{}); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("missing slug = %v", err)
	}
}

func TestSearchService_SuggestSeries(t *testing.T) {
	f := newFixture(t)
	svc := app.NewSearchService(f.source, f.ranking)
	if got, err := svc.SuggestSeries(f.ctx, "r", 5); err != nil || got != nil {
		t.Errorf("short term = %v %v", got, err)
	}
	if got, err := svc.SuggestSeries(f.ctx, "re zero", 0); err != nil || got != nil {
		t.Errorf("zero limit = %v %v", got, err)
	}
	many := []domain.Series{{Slug: "a"}, {Slug: "b"}, {Slug: "c"}}
	f.source.EXPECT().SearchWorks(mock.Anything, "re").Return(many, nil).Once()
	got, err := svc.SuggestSeries(f.ctx, " re ", 2)
	if err != nil || len(got) != 2 || got[1].Slug != "b" {
		t.Errorf("capped = %v %v", got, err)
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	if _, err := svc.SuggestSeries(f.ctx, "boom", 5); err == nil {
		t.Error("expected error")
	}
}

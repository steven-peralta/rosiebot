package app_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestSearchService_WaifusEmptyTerm(t *testing.T) {
	f := newFixture(t)
	if _, err := app.NewSearchService(f.source).Waifus(f.ctx, "   "); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestSearchService_WaifusNoResults(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "nobody", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	if _, err := app.NewSearchService(f.source).Waifus(f.ctx, "nobody"); !errors.Is(err, app.ErrNotFound) {
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
	got, err := app.NewSearchService(f.source).Waifus(f.ctx, "  rem ")
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
	got, err := app.NewSearchService(f.source).Waifus(f.ctx, "rem")
	if err != nil || len(got) != 1 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestSearchService_WaifusTruncatesLongTerms(t *testing.T) {
	f := newFixture(t)
	long := strings.Repeat("a", 150)
	f.source.EXPECT().SearchWaifus(mock.Anything, strings.Repeat("a", 100), 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("a", 1, 0)}}, nil).Once()
	if _, err := app.NewSearchService(f.source).Waifus(f.ctx, long); err != nil {
		t.Fatal(err)
	}
}

func TestSearchService_WaifusPropagatesErrors(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source).Waifus(f.ctx, "rem"); err == nil {
		t.Fatal("expected error")
	}
}

func TestSearchService_RandomFetchesDetail(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()
	got, err := app.NewSearchService(f.source).Random(f.ctx)
	if err != nil || got.Slug != "rem" {
		t.Fatalf("got %+v, %v", got, err)
	}

	f.source.EXPECT().Random(mock.Anything).Return(domain.WaifuSummary{}, errors.New("boom")).Once()
	if _, err := app.NewSearchService(f.source).Random(f.ctx); err == nil {
		t.Fatal("expected error")
	}

	f.source.EXPECT().Get(mock.Anything, "ram").Return(detail("ram"), nil).Once()
	if got, err := app.NewSearchService(f.source).Detail(f.ctx, "ram"); err != nil || got.Slug != "ram" {
		t.Fatalf("Detail = %+v, %v", got, err)
	}
}

func TestSearchService_SeriesNotFound(t *testing.T) {
	f := newFixture(t)
	svc := app.NewSearchService(f.source)
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

	res, err := app.NewSearchService(f.source).Series(f.ctx, "re zero")
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
	if _, err := app.NewSearchService(f.source).Series(f.ctx, "x"); err == nil {
		t.Fatal("expected error")
	}
}

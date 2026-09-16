package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type cacheFixture struct {
	*fixture
	store  *memory.WaifuCache
	cached *app.CachedSource
}

func newCacheFixture(t *testing.T) *cacheFixture {
	f := newFixture(t)
	store := memory.NewWaifuCache(f.clock)
	cached := app.NewCachedSource(f.source, store, f.clock, app.CacheConfig{}, nil)
	t.Cleanup(cached.Flush)
	return &cacheFixture{fixture: f, store: store, cached: cached}
}

func TestCachedSource_GetHitMissAndStale(t *testing.T) {
	f := newCacheFixture(t)
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()

	first, err := f.cached.Get(f.ctx, "rem")
	if err != nil || first.Slug != "rem" {
		t.Fatalf("miss = %+v %v", first, err)
	}
	second, err := f.cached.Get(f.ctx, "rem")
	if err != nil || second.Slug != "rem" {
		t.Fatalf("hit = %+v %v", second, err)
	}
	f.source.AssertNumberOfCalls(t, "Get", 1)

	f.clock.Advance(app.DefaultWaifuTTL + time.Minute)
	refreshed := detail("rem")
	refreshed.Likes = 999
	f.source.EXPECT().Get(mock.MatchedBy(app.IsBackground), "rem").Return(refreshed, nil).Once()
	stale, err := f.cached.Get(f.ctx, "rem")
	if err != nil || stale.Likes != 10 {
		t.Fatalf("stale read should return the old entry immediately: %+v %v", stale, err)
	}
	f.cached.Flush()
	fresh, err := f.cached.Get(f.ctx, "rem")
	if err != nil || fresh.Likes != 999 {
		t.Fatalf("after refresh = %+v %v", fresh, err)
	}
	f.source.AssertNumberOfCalls(t, "Get", 2)
}

func TestCachedSource_StaleRefreshIsSingleFlight(t *testing.T) {
	f := newCacheFixture(t)
	if err := f.store.PutWaifu(f.ctx, detail("rem"), f.clock.now.Add(-2*app.DefaultWaifuTTL)); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	f.source.EXPECT().Get(mock.Anything, "rem").RunAndReturn(func(context.Context, string) (domain.Waifu, error) {
		<-release
		return detail("rem"), nil
	}).Once()
	for range 5 {
		if _, err := f.cached.Get(f.ctx, "rem"); err != nil {
			t.Fatal(err)
		}
	}
	close(release)
	f.cached.Flush()
	f.source.AssertNumberOfCalls(t, "Get", 1)
}

func TestCachedSource_RefreshFailureKeepsStale(t *testing.T) {
	f := newCacheFixture(t)
	if err := f.store.PutWaifu(f.ctx, detail("rem"), f.clock.now.Add(-2*app.DefaultWaifuTTL)); err != nil {
		t.Fatal(err)
	}
	f.source.EXPECT().Get(mock.Anything, "rem").Return(domain.Waifu{}, errors.New("down")).Twice()
	got, err := f.cached.Get(f.ctx, "rem")
	if err != nil || got.Slug != "rem" {
		t.Fatalf("stale = %+v %v", got, err)
	}
	f.cached.Flush()
	again, err := f.cached.Get(f.ctx, "rem")
	if err != nil || again.Slug != "rem" {
		t.Fatalf("stale entry must survive a failed refresh: %+v %v", again, err)
	}
}

func TestCachedSource_MissErrorsPropagateAndAreNotCached(t *testing.T) {
	f := newCacheFixture(t)
	f.source.EXPECT().Get(mock.Anything, "ghost").Return(domain.Waifu{}, app.ErrNotFound).Twice()
	for range 2 {
		if _, err := f.cached.Get(f.ctx, "ghost"); !errors.Is(err, app.ErrNotFound) {
			t.Fatalf("err = %v", err)
		}
	}
	if f.store.Len() != 0 {
		t.Error("failed lookups must not be cached")
	}
}

type brokenCache struct{ app.WaifuCache }

func (brokenCache) GetWaifu(context.Context, string) (app.CachedWaifu, error) {
	return app.CachedWaifu{}, errors.New("disk on fire")
}

func (brokenCache) PutWaifu(context.Context, domain.Waifu, time.Time) error {
	return errors.New("disk on fire")
}

func (brokenCache) GetPage(context.Context, string) (app.CachedPage, error) {
	return app.CachedPage{}, errors.New("disk on fire")
}

func (brokenCache) PutPage(context.Context, string, app.SearchPage, time.Time) error {
	return errors.New("disk on fire")
}

func (brokenCache) GetSeries(context.Context, string) (app.CachedSeries, error) {
	return app.CachedSeries{}, errors.New("disk on fire")
}

func (brokenCache) PutSeries(context.Context, string, []domain.Series, int, time.Time) error {
	return errors.New("disk on fire")
}

func (brokenCache) Prune(context.Context, time.Time) (int64, error) {
	return 0, errors.New("disk on fire")
}

func TestCachedSource_FallsThroughWhenCacheBroken(t *testing.T) {
	f := newFixture(t)
	cached := app.NewCachedSource(f.source, brokenCache{}, f.clock, app.CacheConfig{}, nil)
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()
	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	if _, err := cached.Get(f.ctx, "rem"); err != nil {
		t.Fatal(err)
	}
	if _, err := cached.SearchWaifus(f.ctx, "rem", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := cached.Prune(f.ctx); err == nil {
		t.Error("prune should surface store errors")
	}
}

func TestCachedSource_PagesCachedPerKey(t *testing.T) {
	f := newCacheFixture(t)
	page1 := app.SearchPage{Page: 1, LastPage: 2, Items: []domain.WaifuSummary{summary("shinji", 1, 0)}}
	f.source.EXPECT().SearchWaifus(mock.Anything, "Shinji", 1).Return(page1, nil).Once()
	f.source.EXPECT().SearchWaifus(mock.Anything, "shinji", 2).Return(app.SearchPage{Page: 2, LastPage: 2}, nil).Once()
	f.source.EXPECT().ListCharacters(mock.Anything, 1).Return(app.SearchPage{Page: 1, LastPage: 5}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()

	for _, term := range []string{"Shinji", "  shinji ", "SHINJI"} {
		got, err := f.cached.SearchWaifus(f.ctx, term, 1)
		if err != nil || len(got.Items) != 1 {
			t.Fatalf("%q = %+v %v", term, got, err)
		}
	}
	if _, err := f.cached.SearchWaifus(f.ctx, "shinji", 2); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := f.cached.ListCharacters(f.ctx, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := f.cached.WorkCharacters(f.ctx, "re-zero", 1); err != nil {
			t.Fatal(err)
		}
	}
	f.source.AssertNumberOfCalls(t, "SearchWaifus", 2)
	f.source.AssertNumberOfCalls(t, "ListCharacters", 1)
	f.source.AssertNumberOfCalls(t, "WorkCharacters", 1)

	f.clock.Advance(app.DefaultSearchTTL + time.Second)
	f.source.EXPECT().SearchWaifus(mock.MatchedBy(app.IsBackground), "shinji", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("shinji", 1, 0), summary("shinji-2", 1, 0)}}, nil).Once()
	stale, err := f.cached.SearchWaifus(f.ctx, "shinji", 1)
	if err != nil || len(stale.Items) != 1 {
		t.Fatalf("stale page = %+v %v", stale, err)
	}
	f.cached.Flush()
	fresh, err := f.cached.SearchWaifus(f.ctx, "shinji", 1)
	if err != nil || len(fresh.Items) != 2 {
		t.Fatalf("refreshed page = %+v %v", fresh, err)
	}

	f.source.EXPECT().SearchWaifus(mock.Anything, "boom", 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	if _, err := f.cached.SearchWaifus(f.ctx, "boom", 1); err == nil {
		t.Error("miss errors should propagate")
	}
}

func TestCachedSource_PassThroughs(t *testing.T) {
	f := newCacheFixture(t)
	f.source.EXPECT().Random(mock.Anything).Return(summary("r", 1, 0), nil).Twice()
	f.source.EXPECT().Daily(mock.Anything).Return(summary("d", 1, 0), nil).Twice()
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(app.PopularPage{Page: 1}, nil).Twice()
	f.source.EXPECT().Work(mock.Anything, "x").Return(domain.Series{Slug: "x"}, nil).Twice()
	for range 2 {
		if _, err := f.cached.Random(f.ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := f.cached.Daily(f.ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := f.cached.PopularPage(f.ctx, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := f.cached.Work(f.ctx, "x"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCachedSource_SeriesSearchCached(t *testing.T) {
	f := newCacheFixture(t)
	f.source.EXPECT().SearchWorks(mock.Anything, "Re Zero").Return([]domain.Series{{Slug: "re-zero", Name: "Re:Zero"}}, nil).Once()
	for _, term := range []string{"Re Zero", "re  zero", "RE ZERO"} {
		got, err := f.cached.SearchWorks(f.ctx, term)
		if err != nil || len(got) != 1 || got[0].Slug != "re-zero" {
			t.Fatalf("%q = %v %v", term, got, err)
		}
	}
	f.source.AssertNumberOfCalls(t, "SearchWorks", 1)

	f.clock.Advance(app.DefaultSearchTTL + time.Second)
	f.source.EXPECT().SearchWorks(mock.MatchedBy(app.IsBackground), "re zero").Return([]domain.Series{{Slug: "re-zero"}, {Slug: "re-zero-2"}}, nil).Once()
	stale, err := f.cached.SearchWorks(f.ctx, "re zero")
	if err != nil || len(stale) != 1 {
		t.Fatalf("stale = %v %v", stale, err)
	}
	f.cached.Flush()
	fresh, err := f.cached.SearchWorks(f.ctx, "re zero")
	if err != nil || len(fresh) != 2 {
		t.Fatalf("refreshed = %v %v", fresh, err)
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	if _, err := f.cached.SearchWorks(f.ctx, "boom"); err == nil {
		t.Error("miss errors should propagate")
	}

	broken := app.NewCachedSource(f.source, brokenCache{}, f.clock, app.CacheConfig{}, nil)
	f.source.EXPECT().SearchWorks(mock.Anything, "x").Return([]domain.Series{{Slug: "x"}}, nil).Once()
	if got, err := broken.SearchWorks(f.ctx, "x"); err != nil || len(got) != 1 {
		t.Errorf("broken cache should fall through: %v %v", got, err)
	}
}

package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

var weekAtFixture = 4*24*time.Hour + 22*time.Hour

func series(slug string) domain.Series {
	return domain.Series{Slug: slug, Name: "Series " + slug, URL: "https://www.mywaifulist.moe/series/" + slug, PictureURL: "https://img/series/" + slug}
}

func detailIn(slug string, s domain.Series) domain.Waifu {
	d := detail(slug)
	d.Appearances = []domain.Series{s}
	return d
}

func members(page, lastPage int, slugs ...string) app.SearchPage {
	items := make([]domain.WaifuSummary, len(slugs))
	for i, slug := range slugs {
		items[i] = summary(slug, 1, 0)
	}
	return app.SearchPage{Items: items, Page: page, LastPage: lastPage}
}

func (f *fixture) bannerWith(cfg app.BannerConfig) *app.BannerService {
	return app.NewBannerService(f.banners, f.ranking, f.source, f.clock, f.rng, f.loc, cfg, nil)
}

func TestBannerService_CurrentReturnsStoredAndRefreshIn(t *testing.T) {
	f := newFixture(t)
	f.seedBanner("ranked-050", "ranked-000", "ranked-020", "ranked-005", "ranked-100")

	res, err := f.banner().Current(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.RefreshIn != weekAtFixture {
		t.Errorf("RefreshIn = %v, want %v", res.RefreshIn, weekAtFixture)
	}
	if got := res.Banner.Slugs(); len(got) != 5 || got[0] != "ranked-000" || got[4] != "ranked-100" {
		t.Errorf("characters = %v, want sorted by rank", got)
	}
	if res.Banner.Series.Slug != "re-zero" || !res.Banner.WeekStart.Equal(domain.BannerWeekStart(f.clock.now, f.loc)) {
		t.Errorf("banner header = %+v", res.Banner)
	}
}

func TestBannerService_CurrentWithoutBannerIsErrNoBanner(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	if _, err := f.banner().Current(f.ctx); !errors.Is(err, app.ErrNoBanner) {
		t.Fatalf("err = %v, want ErrNoBanner", err)
	}
	f.source.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestBannerService_EnsurePicksEligibleSeries(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.script(3)
	f.source.EXPECT().Get(mock.MatchedBy(app.IsBackground), "ranked-003").Return(detailIn("ranked-003", series("re-zero")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.MatchedBy(app.IsBackground), "re-zero", 1).
		Return(members(1, 1, "ranked-003", "ghost", "ranked-050", "ranked-000", "ranked-150", "ranked-010", "ranked-003"), nil).Once()

	got, err := f.banner().Ensure(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ranked-000", "ranked-003", "ranked-010", "ranked-050", "ranked-150"}
	if slugs := got.Slugs(); len(slugs) != len(want) {
		t.Fatalf("characters = %v, want %v", slugs, want)
	} else {
		for i := range want {
			if slugs[i] != want[i] {
				t.Errorf("characters[%d] = %s, want %s", i, slugs[i], want[i])
			}
		}
	}
	if got.Series.Slug != "re-zero" || got.Series.PictureURL == "" || got.Characters[0].Stars != 5 {
		t.Errorf("banner = %+v", got)
	}
	stored, err := f.banners.Get(f.ctx, domain.BannerWeekStart(f.clock.now, f.loc))
	if err != nil || stored.Series.Slug != "re-zero" {
		t.Errorf("banner not persisted: %+v %v", stored, err)
	}
	f.source.AssertNotCalled(t, "Work", mock.Anything, mock.Anything)
}

func TestBannerService_PickerSkipsIneligibleAndDuplicateSeries(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.script(0, 1, 2, 4)
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(detailIn("ranked-000", series("tiny")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "tiny", 1).Return(members(1, 1, "ranked-000", "ranked-001"), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-001").Return(detailIn("ranked-001", series("tiny")), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-002").Return(detailIn("ranked-002", series("dull")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "dull", 1).Return(members(1, 1, "ranked-100", "ranked-101", "ranked-102", "ranked-103", "ranked-104", "ranked-105"), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-004").Return(detailIn("ranked-004", series("big")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "big", 1).Return(members(1, 1, "ranked-004", "ranked-040", "ranked-041", "ranked-042", "ranked-043"), nil).Once()

	got, err := f.banner().Ensure(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Series.Slug != "big" || len(got.Characters) != 5 {
		t.Errorf("banner = %+v", got)
	}
}

func TestBannerService_PickerExcludesPreviousWeek(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	lastWeek := domain.BannerWeekStart(f.clock.now, f.loc).AddDate(0, 0, -7)
	if _, err := f.banners.Put(f.ctx, domain.NewBanner(lastWeek, series("last"), nil)); err != nil {
		t.Fatal(err)
	}
	f.script(0, 1)
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(detailIn("ranked-000", series("last")), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-001").Return(detailIn("ranked-001", series("fresh")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "fresh", 1).Return(members(1, 1, "ranked-001", "ranked-011", "ranked-012", "ranked-013", "ranked-014"), nil).Once()

	got, err := f.banner().Ensure(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got.Series.Slug != "fresh" {
		t.Errorf("series = %s, want the previous week's series skipped", got.Series.Slug)
	}
	f.source.AssertNotCalled(t, "WorkCharacters", mock.Anything, "last", mock.Anything)
}

func TestBannerService_PickerWalksPagesUpToCap(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.script(0)
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(detailIn("ranked-000", series("huge")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "huge", 1).Return(members(1, 5, "ranked-000", "ranked-001", "ranked-002"), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "huge", 2).Return(members(2, 5, "ranked-003", "ranked-004", "ranked-005"), nil).Once()

	got, err := f.bannerWith(app.BannerConfig{MaxPages: 2}).Ensure(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Characters) != 6 {
		t.Errorf("characters = %v, want the first two pages only", got.Slugs())
	}
	f.source.AssertNotCalled(t, "WorkCharacters", mock.Anything, "huge", 3)

	g := newFixture(t)
	g.ranking.Set(rankingOf(200))
	g.script(0)
	g.source.EXPECT().Get(mock.Anything, "ranked-000").Return(detailIn("ranked-000", series("short")), nil).Once()
	g.source.EXPECT().WorkCharacters(mock.Anything, "short", 1).Return(members(1, 9, "ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004"), nil).Once()
	g.source.EXPECT().WorkCharacters(mock.Anything, "short", 2).Return(members(2, 9), nil).Once()
	if got, err := g.banner().Ensure(g.ctx); err != nil || len(got.Characters) != 5 {
		t.Errorf("empty page should stop the walk: %v %v", got.Slugs(), err)
	}
}

func TestBannerService_PickerFillsMissingPicture(t *testing.T) {
	bare := series("plain")
	bare.PictureURL = ""
	for name, workErr := range map[string]error{"work ok": nil, "work fails": errors.New("boom")} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.ranking.Set(rankingOf(200))
			f.script(0)
			f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(detailIn("ranked-000", bare), nil).Once()
			f.source.EXPECT().WorkCharacters(mock.Anything, "plain", 1).Return(members(1, 1, "ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004"), nil).Once()
			f.source.EXPECT().Work(mock.MatchedBy(app.IsBackground), "plain").Return(series("plain"), workErr).Once()

			got, err := f.banner().Ensure(f.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if (got.Series.PictureURL != "") != (workErr == nil) {
				t.Errorf("picture = %q with workErr=%v", got.Series.PictureURL, workErr)
			}
		})
	}
}

func TestBannerService_PickerBoundedAttemptsWritesNothing(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.script(0, 1, 2)
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(domain.Waifu{}, errors.New("down")).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-001").Return(detail("ranked-001"), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-002").Return(detailIn("ranked-002", series("broken")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "broken", 1).Return(app.SearchPage{}, errors.New("down")).Once()

	_, err := f.bannerWith(app.BannerConfig{MaxAttempts: 3}).Ensure(f.ctx)
	if !errors.Is(err, app.ErrNoEligibleSeries) {
		t.Fatalf("err = %v, want ErrNoEligibleSeries", err)
	}
	if _, err := f.banners.Get(f.ctx, domain.BannerWeekStart(f.clock.now, f.loc)); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("nothing should be stored, got %v", err)
	}
}

func TestBannerService_PickerFallsBackToAnyRankedSeed(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(10))
	f.script(4)
	f.source.EXPECT().Get(mock.Anything, "ranked-004").Return(detailIn("ranked-004", series("small")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "small", 1).Return(members(1, 1, "ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004"), nil).Once()

	_, err := f.bannerWith(app.BannerConfig{MaxAttempts: 1}).Ensure(f.ctx)
	if !errors.Is(err, app.ErrNoEligibleSeries) {
		t.Fatalf("a ranking with no 4-star rows should still seed from any ranked row and then fail eligibility, got %v", err)
	}
}

func TestBannerService_EnsureWithoutRankingIsErrNoRanking(t *testing.T) {
	f := newFixture(t)
	if _, err := f.banner().Ensure(f.ctx); !errors.Is(err, app.ErrNoRanking) {
		t.Fatalf("err = %v, want ErrNoRanking", err)
	}
}

func TestBannerService_EnsureReturnsExistingWithoutPicking(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	seeded := f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	got, err := f.banner().Ensure(f.ctx)
	if err != nil || got.Series.Slug != seeded.Series.Slug {
		t.Fatalf("Ensure = %+v %v", got, err)
	}
	f.source.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestBannerService_EnsureCancelledContext(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if _, err := f.banner().Ensure(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

type failingBanners struct {
	getErr error
	putErr error
}

func (b failingBanners) Get(context.Context, time.Time) (domain.Banner, error) {
	if b.getErr != nil {
		return domain.Banner{}, b.getErr
	}
	return domain.Banner{}, app.ErrNotFound
}

func (b failingBanners) Put(context.Context, domain.Banner) (domain.Banner, error) {
	return domain.Banner{}, b.putErr
}

func TestBannerService_StoreErrorsPropagate(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	broken := app.NewBannerService(failingBanners{getErr: errors.New("get boom")}, f.ranking, f.source, f.clock, f.rng, nil, app.BannerConfig{}, nil)
	if _, err := broken.Current(f.ctx); err == nil || errors.Is(err, app.ErrNoBanner) {
		t.Errorf("Current should surface the store error, got %v", err)
	}
	if _, err := broken.Ensure(f.ctx); err == nil {
		t.Error("Ensure should surface the get error")
	}

	f.script(0)
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(detailIn("ranked-000", series("ok")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "ok", 1).Return(members(1, 1, "ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004"), nil).Once()
	putFails := app.NewBannerService(failingBanners{putErr: errors.New("put boom")}, f.ranking, f.source, f.clock, f.rng, nil, app.BannerConfig{}, nil)
	if _, err := putFails.Ensure(f.ctx); err == nil {
		t.Error("Ensure should surface the put error")
	}
}

func TestBannerService_ConcurrentPutKeepsFirstWinner(t *testing.T) {
	f := newFixture(t)
	week := domain.BannerWeekStart(f.clock.now, f.loc)
	first, err := f.banners.Put(f.ctx, domain.NewBanner(week, series("first"), nil))
	if err != nil || first.Series.Slug != "first" {
		t.Fatal(err)
	}
	second, err := f.banners.Put(f.ctx, domain.NewBanner(week, series("second"), nil))
	if err != nil || second.Series.Slug != "first" {
		t.Errorf("Put should return the existing winner, got %+v", second)
	}
}

func TestBannerService_RunEnsuresThenSleepsUntilBoundary(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	f.script(3)
	f.source.EXPECT().Get(mock.Anything, "ranked-003").Return(detailIn("ranked-003", series("next")), nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "next", 1).Return(members(1, 1, "ranked-003", "ranked-030", "ranked-031", "ranked-032", "ranked-033"), nil).Once()

	svc := f.banner()
	ctx, cancel := context.WithCancel(f.ctx)
	var sleeps []time.Duration
	svc.SetSleepForTest(func(_ context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		f.clock.Advance(d)
		if len(sleeps) == 2 {
			cancel()
			return context.Canceled
		}
		return nil
	})
	if err := svc.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned %v", err)
	}
	if len(sleeps) != 2 || sleeps[0] != weekAtFixture || sleeps[1] != 7*24*time.Hour {
		t.Errorf("sleeps = %v, want until Monday 10:00 then a full week", sleeps)
	}
	next, err := f.banners.Get(f.ctx, domain.BannerWeekStart(f.clock.now, f.loc).AddDate(0, 0, -7))
	if err != nil || next.Series.Slug != "next" {
		t.Errorf("second week's banner = %+v %v", next, err)
	}
}

func TestBannerService_RunRetriesAfterFailureCappedAtBoundary(t *testing.T) {
	cases := map[string]struct {
		cfg     app.BannerConfig
		ranking bool
		want    time.Duration
	}{
		"waiting for ranking": {cfg: app.BannerConfig{}, want: 30 * time.Second},
		"pick failed":         {cfg: app.BannerConfig{MaxAttempts: 1}, ranking: true, want: 10 * time.Minute},
		"boundary caps":       {cfg: app.BannerConfig{RankingWait: 400 * time.Hour}, want: weekAtFixture},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			if c.ranking {
				f.ranking.Set(rankingOf(200))
				f.script(0)
				f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(domain.Waifu{}, errors.New("down")).Once()
			}
			svc := f.bannerWith(c.cfg)
			ctx, cancel := context.WithCancel(f.ctx)
			var sleeps []time.Duration
			svc.SetSleepForTest(func(_ context.Context, d time.Duration) error {
				sleeps = append(sleeps, d)
				cancel()
				return context.Canceled
			})
			if err := svc.Run(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("Run returned %v", err)
			}
			if len(sleeps) != 1 || sleeps[0] != c.want {
				t.Errorf("sleeps = %v, want %v", sleeps, c.want)
			}
		})
	}
}

func TestBannerService_RunReturnsWhenCancelled(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if err := f.banner().Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned %v", err)
	}

	g := newFixture(t)
	g.ranking.Set(rankingOf(200))
	ctx, cancel = context.WithCancel(g.ctx)
	g.source.EXPECT().Get(mock.Anything, "ranked-000").RunAndReturn(func(context.Context, string) (domain.Waifu, error) {
		cancel()
		return domain.Waifu{}, errors.New("down")
	}).Once()
	g.script(0)
	if err := g.banner().Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run after mid-pick cancel returned %v", err)
	}
}

func TestBannerConfig_Defaults(t *testing.T) {
	d := app.DefaultBannerConfig()
	if d.MaxAttempts != 12 || d.MaxPages != 30 || d.MinRanked != domain.BannerMinRanked || d.MinStars != domain.BannerMinStars || d.RetryInterval != 10*time.Minute || d.RankingWait != 30*time.Second {
		t.Errorf("defaults = %+v", d)
	}
}

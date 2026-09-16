package app_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type rankingFixture struct {
	*fixture
	store  *memory.RankingStore
	sleeps []time.Duration
	svc    *app.RankingService
}

func newRankingFixture(t *testing.T, cfg app.RankingConfig) *rankingFixture {
	f := &rankingFixture{fixture: newFixture(t), store: memory.NewRankingStore()}
	f.svc = app.NewRankingService(f.store, f.source, f.clock, cfg, nil)
	f.svc.SetSleepForTest(func(_ context.Context, d time.Duration) error {
		f.sleeps = append(f.sleeps, d)
		f.clock.Advance(d)
		return nil
	})
	return f
}

func page(n, lastPage int, totals ...int) app.PopularPage {
	rows := make([]domain.WaifuSummary, len(totals))
	for i, total := range totals {
		rows[i] = summary(fmt.Sprintf("p%d-%d", n, i), total-1, 1)
	}
	return app.PopularPage{Rows: rows, Page: n, LastPage: lastPage}
}

func TestRankingWalker_StopsAtCutoff(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{})
	f.source.EXPECT().PopularPage(mock.MatchedBy(app.IsBackground), 1).Return(page(1, 5000, 900, 800), nil).Once()
	f.source.EXPECT().PopularPage(mock.Anything, 2).Return(page(2, 5000, 300, 200), nil).Once()
	f.source.EXPECT().PopularPage(mock.Anything, 3).Return(page(3, 5000, 101, 100), nil).Once()

	if err := f.svc.Refresh(f.ctx); err != nil {
		t.Fatal(err)
	}
	r := f.svc.Current()
	if r == nil || r.Len() != 5 || r.CutoffPage != 3 {
		t.Fatalf("ranking = len %d cutoff %d", r.Len(), r.CutoffPage)
	}
	if _, ok := r.Lookup("p3-1"); ok {
		t.Error("row with exactly 100 votes must not be ranked")
	}
	if !r.FetchedAt.Equal(f.clock.now) {
		t.Errorf("FetchedAt = %v", r.FetchedAt)
	}
	saved, err := f.store.LoadLatest(f.ctx)
	if err != nil || saved.Len() != 5 {
		t.Errorf("snapshot not saved: %v %v", saved, err)
	}
	if got := f.svc.NextRefresh(); !got.Equal(f.clock.now.Add(24 * time.Hour)) {
		t.Errorf("NextRefresh = %v", got)
	}
}

func TestRankingWalker_StopsAtLastPage(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{})
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(page(1, 2, 900), nil).Once()
	f.source.EXPECT().PopularPage(mock.Anything, 2).Return(page(2, 2, 800), nil).Once()
	if err := f.svc.Refresh(f.ctx); err != nil {
		t.Fatal(err)
	}
	if f.svc.Current().Len() != 2 || f.svc.Current().CutoffPage != 2 {
		t.Errorf("ranking = %+v", f.svc.Current())
	}
}

func TestRankingWalker_StopsOnEmptyPageAndCap(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{})
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(page(1, 0, 900), nil).Once()
	f.source.EXPECT().PopularPage(mock.Anything, 2).Return(app.PopularPage{Page: 2}, nil).Once()
	if err := f.svc.Refresh(f.ctx); err != nil {
		t.Fatal(err)
	}
	if f.svc.Current().Len() != 1 || f.svc.Current().CutoffPage != 2 {
		t.Errorf("ranking after empty page = %+v", f.svc.Current())
	}

	capped := newRankingFixture(t, app.RankingConfig{MaxPages: 2})
	capped.source.EXPECT().PopularPage(mock.Anything, 1).Return(page(1, 0, 900), nil).Once()
	capped.source.EXPECT().PopularPage(mock.Anything, 2).Return(page(2, 0, 800), nil).Once()
	if err := capped.svc.Refresh(capped.ctx); err != nil {
		t.Fatal(err)
	}
	if capped.svc.Current().Len() != 2 || capped.svc.Current().CutoffPage != 2 {
		t.Errorf("ranking at cap = %+v", capped.svc.Current())
	}
}

func TestRankingWalker_ResumesAfterTransientError(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{BackoffBase: time.Second, BackoffMax: 3 * time.Second})
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(page(1, 0, 900), nil).Once()
	f.source.EXPECT().PopularPage(mock.Anything, 2).Return(app.PopularPage{}, errors.New("flaky")).Times(3)
	f.source.EXPECT().PopularPage(mock.Anything, 2).Return(page(2, 0, 101, 50), nil).Once()

	if err := f.svc.Refresh(f.ctx); err != nil {
		t.Fatal(err)
	}
	if f.svc.Current().Len() != 2 {
		t.Errorf("ranking = %+v", f.svc.Current())
	}
	want := []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}
	if len(f.sleeps) != 3 || f.sleeps[0] != want[0] || f.sleeps[1] != want[1] || f.sleeps[2] != want[2] {
		t.Errorf("backoff sleeps = %v, want %v", f.sleeps, want)
	}
}

func TestRankingWalker_AbandonKeepsOldTable(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{PageRetries: 2})
	old := rankingOf(10)
	if err := f.store.Save(f.ctx, old); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Load(f.ctx); err != nil {
		t.Fatal(err)
	}
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(app.PopularPage{}, errors.New("down")).Times(2)

	err := f.svc.Refresh(f.ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if f.svc.Current() != old {
		t.Error("failed refresh must keep the previous table")
	}
	saved, _ := f.store.LoadLatest(f.ctx)
	if saved != old {
		t.Error("failed refresh must not write a snapshot")
	}
}

func TestRankingWalker_SaveFailure(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{})
	failing := app.NewRankingService(failingRankingStore{}, f.source, f.clock, app.RankingConfig{}, nil)
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(page(1, 1, 900), nil).Once()
	if err := failing.Refresh(f.ctx); err == nil {
		t.Fatal("expected save error")
	}
	if failing.Current() != nil {
		t.Error("table must not be published when the save fails")
	}
	if err := failing.Load(f.ctx); err == nil {
		t.Error("expected load error")
	}
}

type failingRankingStore struct{}

func (failingRankingStore) LoadLatest(context.Context) (*domain.Ranking, error) {
	return nil, errors.New("disk on fire")
}

func (failingRankingStore) Save(context.Context, *domain.Ranking) error {
	return errors.New("disk on fire")
}

func TestRankingWalker_ContextCancelledDuringRetry(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{})
	ctx, cancel := context.WithCancel(f.ctx)
	f.source.EXPECT().PopularPage(mock.Anything, 1).RunAndReturn(func(context.Context, int) (app.PopularPage, error) {
		cancel()
		return app.PopularPage{}, errors.New("boom")
	}).Once()
	if err := f.svc.Refresh(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
	if len(f.sleeps) != 0 {
		t.Error("should not back off after cancellation")
	}
}

func TestRankingService_LoadPublishesStaleSnapshot(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{})
	old := rankingOf(10)
	if err := f.store.Save(f.ctx, old); err != nil {
		t.Fatal(err)
	}
	if f.svc.Current() != nil {
		t.Fatal("precondition")
	}
	if !f.svc.NextRefresh().Equal(f.clock.now) {
		t.Error("with no table the next refresh should be now")
	}
	if err := f.svc.Load(f.ctx); err != nil {
		t.Fatal(err)
	}
	if f.svc.Current() != old {
		t.Error("Load should publish the stored snapshot even if stale")
	}
	if !f.svc.NextRefresh().Equal(old.FetchedAt.Add(24 * time.Hour)) {
		t.Errorf("NextRefresh = %v", f.svc.NextRefresh())
	}
	if err := f.svc.Load(f.ctx); err != nil {
		t.Fatal(err)
	}
}

func TestRankingService_RefreshRejectsOverlap(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{})
	release := make(chan struct{})
	entered := make(chan struct{})
	f.source.EXPECT().PopularPage(mock.Anything, 1).RunAndReturn(func(context.Context, int) (app.PopularPage, error) {
		close(entered)
		<-release
		return page(1, 1, 900), nil
	}).Once()

	done := make(chan error, 1)
	go func() { done <- f.svc.Refresh(f.ctx) }()
	<-entered
	if err := f.svc.Refresh(f.ctx); !errors.Is(err, app.ErrRefreshInProgress) {
		t.Errorf("overlapping refresh err = %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if f.svc.Current().Len() != 1 {
		t.Error("first refresh should complete")
	}
}

func TestRankingService_RunRefreshesThenSleepsUntilNext(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{RefreshInterval: 6 * time.Hour})
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(page(1, 1, 900), nil).Once()
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(page(1, 1, 950, 900), nil).Once()

	ctx, cancel := context.WithCancel(f.ctx)
	refreshes := 0
	f.svc.SetSleepForTest(func(_ context.Context, d time.Duration) error {
		f.sleeps = append(f.sleeps, d)
		f.clock.Advance(d)
		refreshes++
		if refreshes == 2 {
			cancel()
			return context.Canceled
		}
		return nil
	})

	if err := f.svc.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned %v", err)
	}
	if len(f.sleeps) != 2 || f.sleeps[0] != 6*time.Hour || f.sleeps[1] != 6*time.Hour {
		t.Errorf("sleeps = %v, want two full intervals", f.sleeps)
	}
	if f.svc.Current().Len() != 2 {
		t.Errorf("second refresh should have published, len=%d", f.svc.Current().Len())
	}
}

func TestRankingService_RunRetriesAfterFailure(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{PageRetries: 1, FailureRetry: 30 * time.Minute})
	f.source.EXPECT().PopularPage(mock.Anything, 1).Return(app.PopularPage{}, errors.New("down")).Once()
	ctx, cancel := context.WithCancel(f.ctx)
	f.svc.SetSleepForTest(func(_ context.Context, d time.Duration) error {
		f.sleeps = append(f.sleeps, d)
		cancel()
		return context.Canceled
	})
	if err := f.svc.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run returned %v", err)
	}
	if len(f.sleeps) != 1 || f.sleeps[0] != 30*time.Minute {
		t.Errorf("sleeps = %v, want the failure retry delay", f.sleeps)
	}
}

func TestRankingService_RunReturnsWhenCancelledMidRefresh(t *testing.T) {
	f := newRankingFixture(t, app.RankingConfig{PageRetries: 1})
	ctx, cancel := context.WithCancel(f.ctx)
	f.source.EXPECT().PopularPage(mock.Anything, 1).RunAndReturn(func(context.Context, int) (app.PopularPage, error) {
		cancel()
		return app.PopularPage{}, errors.New("boom")
	}).Once()
	if err := f.svc.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Run returned %v", err)
	}
}

func TestRankingConfig_Defaults(t *testing.T) {
	d := app.DefaultRankingConfig()
	if d.RefreshInterval != 24*time.Hour || d.MinVotes != 100 || d.MaxPages != 1500 || d.PageRetries != 5 || d.FailureRetry != time.Hour {
		t.Errorf("defaults = %+v", d)
	}
}

func TestRankingService_RealSleepHonoursCancellation(t *testing.T) {
	f := newFixture(t)
	store := memory.NewRankingStore()
	old := rankingOf(10)
	if err := store.Save(f.ctx, old); err != nil {
		t.Fatal(err)
	}
	clock := app.ClockFunc(func() time.Time { return old.FetchedAt })
	svc := app.NewRankingService(store, f.source, clock, app.RankingConfig{}, nil)
	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	if err := svc.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Run on cancelled ctx = %v", err)
	}

	ready := app.NewRankingService(store, f.source, clock, app.RankingConfig{}, nil)
	if err := ready.Load(f.ctx); err != nil {
		t.Fatal(err)
	}
	slow, cancelSlow := context.WithTimeout(f.ctx, 50*time.Millisecond)
	defer cancelSlow()
	if err := ready.Run(slow); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run should return when the sleep is interrupted: %v", err)
	}
}

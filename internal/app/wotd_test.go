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

func TestWotd_SameForWholeDayAndPersisted(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.script(17)

	first, err := f.wotd().Today(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if first.RefreshIn != 12*time.Hour {
		t.Errorf("RefreshIn at noon = %v, want 12h", first.RefreshIn)
	}

	f.clock.Advance(11 * time.Hour)
	again, err := f.wotd().Today(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if again.Waifu.Slug != first.Waifu.Slug || again.RefreshIn != time.Hour {
		t.Errorf("second call = %+v, first = %+v", again, first)
	}

	restarted := app.NewWotdService(f.daily, memory.NewRankingHolder(nil), f.clock, f.rng, f.loc, nil)
	afterRestart, err := restarted.Today(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if afterRestart.Waifu.Slug != first.Waifu.Slug {
		t.Error("waifu of the day should come from the store after a restart")
	}

	f.clock.Advance(2 * time.Hour)
	f.script(3)
	tomorrow, err := f.wotd().Today(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tomorrow.Day.Equal(first.Day) {
		t.Error("day should have rolled over")
	}
}

func TestWotd_PickHasBetween1And4Stars(t *testing.T) {
	f := newFixture(t)
	ranking := rankingOf(1000)
	f.ranking.Set(ranking)
	for i := range 50 {
		f.daily = memory.NewDailyStore()
		f.script(i * 7)
		res, err := f.wotd().Today(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		row, ok := ranking.Lookup(res.Waifu.Slug)
		if !ok || row.Stars < 1 || row.Stars > 4 {
			t.Fatalf("pick %s has stars=%d", res.Waifu.Slug, row.Stars)
		}
	}
}

func TestWotd_NoRankingMeansNoPickAndNothingStored(t *testing.T) {
	f := newFixture(t)
	if _, err := f.wotd().Today(f.ctx); !errors.Is(err, app.ErrNoRanking) {
		t.Fatalf("err = %v, want ErrNoRanking", err)
	}
	if _, err := f.daily.Get(f.ctx, domain.WotdDay(f.clock.now, f.loc)); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("nothing should be stored without a ranking, got %v", err)
	}
	f.source.AssertNotCalled(t, "Daily", mock.Anything)
}

func TestWotd_ReplacesStoredUnrankedPick(t *testing.T) {
	f := newFixture(t)
	day := domain.WotdDay(f.clock.now, f.loc)
	if _, err := f.daily.Put(f.ctx, day, summary("unranked", 50, 5)); err != nil {
		t.Fatal(err)
	}
	res, err := f.wotd().Today(f.ctx)
	if err != nil || res.Waifu.Slug != "unranked" {
		t.Fatalf("without a ranking the stored pick stands: %+v %v", res, err)
	}

	f.ranking.Set(rankingOf(200))
	f.script(17)
	res, err = f.wotd().Today(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if res.Waifu.Slug == "unranked" {
		t.Fatal("an unranked stored pick should be replaced once the ranking is available")
	}
	if row, ok := rankingOf(200).Lookup(res.Waifu.Slug); !ok || !domain.WotdEligible(row) {
		t.Errorf("replacement %s is not an eligible ranked pick", res.Waifu.Slug)
	}
	stored, err := f.daily.Get(f.ctx, day)
	if err != nil || stored.Slug != res.Waifu.Slug {
		t.Errorf("replacement not persisted: %+v %v", stored, err)
	}
	again, err := f.wotd().Today(f.ctx)
	if err != nil || again.Waifu.Slug != res.Waifu.Slug {
		t.Errorf("a ranked stored pick must stay put: %+v %v", again, err)
	}
}

type failingDaily struct {
	getErr error
	putErr error
}

func (d failingDaily) Get(context.Context, time.Time) (domain.WaifuSummary, error) {
	if d.getErr != nil {
		return domain.WaifuSummary{}, d.getErr
	}
	return domain.WaifuSummary{}, app.ErrNotFound
}

func (d failingDaily) Put(context.Context, time.Time, domain.WaifuSummary) (domain.WaifuSummary, error) {
	return domain.WaifuSummary{}, d.putErr
}

func (d failingDaily) Replace(context.Context, time.Time, domain.WaifuSummary) error {
	return d.putErr
}

type unrankedDaily struct{ failingDaily }

func (unrankedDaily) Get(context.Context, time.Time) (domain.WaifuSummary, error) {
	return domain.WaifuSummary{Slug: "unranked"}, nil
}

func TestWotd_StoreErrorsPropagate(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(100))
	svc := app.NewWotdService(failingDaily{getErr: errors.New("get boom")}, f.ranking, f.clock, f.rng, nil, nil)
	if _, err := svc.Today(f.ctx); err == nil {
		t.Error("expected get error")
	}
	f.script(0)
	svc = app.NewWotdService(failingDaily{putErr: errors.New("put boom")}, f.ranking, f.clock, f.rng, nil, nil)
	if _, err := svc.Today(f.ctx); err == nil {
		t.Error("expected put error")
	}
	f.script(0)
	svc = app.NewWotdService(unrankedDaily{failingDaily{putErr: errors.New("replace boom")}}, f.ranking, f.clock, f.rng, nil, nil)
	if _, err := svc.Today(f.ctx); err == nil {
		t.Error("expected replace error")
	}
}

func TestWotd_ConcurrentPutKeepsFirstWinner(t *testing.T) {
	f := newFixture(t)
	day := domain.WotdDay(f.clock.now, f.loc)
	first, err := f.daily.Put(f.ctx, day, summary("first", 1, 0))
	if err != nil || first.Slug != "first" {
		t.Fatal(err)
	}
	second, err := f.daily.Put(f.ctx, day, summary("second", 1, 0))
	if err != nil || second.Slug != "first" {
		t.Errorf("Put should return the existing winner, got %+v", second)
	}
}

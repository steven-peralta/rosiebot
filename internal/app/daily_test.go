package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func (f *fixture) dailySvc() *app.DailyService {
	return app.NewDailyService(f.players, f.clock, f.rng, f.loc)
}

func TestDailyService_ClaimOncePerWindow(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))

	res, err := f.dailySvc().Claim(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Coins != domain.DailyCoins || res.Multiplier != 1 || res.Critical() || res.Balance != domain.StartingCoins+domain.DailyCoins {
		t.Errorf("first claim = %+v", res)
	}

	f.clock.Advance(time.Hour)
	_, err = f.dailySvc().Claim(f.ctx, alice)
	var already *app.DailyAlreadyClaimedError
	if !errors.As(err, &already) {
		t.Fatalf("second claim err = %v, want DailyAlreadyClaimedError", err)
	}
	if already.RefreshIn != 21*time.Hour {
		t.Errorf("RefreshIn = %v, want 21h (13:00 -> next day 10:00)", already.RefreshIn)
	}
	if already.Error() == "" {
		t.Error("error should render")
	}

	f.clock.Advance(21 * time.Hour)
	f.script(d100(50))
	if _, err := f.dailySvc().Claim(f.ctx, alice); err != nil {
		t.Fatalf("claim in next window: %v", err)
	}
}

func TestDailyService_Multipliers(t *testing.T) {
	cases := []struct {
		roll, wantMult int
		wantCoins      int64
	}{
		{roll: 1, wantMult: 5, wantCoins: 2000},
		{roll: 2, wantMult: 2, wantCoins: 800},
		{roll: 21, wantMult: 2, wantCoins: 800},
		{roll: 22, wantMult: 1, wantCoins: 400},
		{roll: 100, wantMult: 1, wantCoins: 400},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.script(d100(c.roll))
		res, err := f.dailySvc().Claim(f.ctx, alice)
		if err != nil {
			t.Fatal(err)
		}
		if res.Multiplier != c.wantMult || res.Coins != c.wantCoins || res.Critical() != (c.wantMult > 1) {
			t.Errorf("roll %d: result = %+v", c.roll, res)
		}
	}
}

func TestDailyService_ClaimBeforeTenUsesPreviousWindow(t *testing.T) {
	f := newFixture(t)
	f.clock.now = time.Date(2026, 9, 16, 9, 0, 0, 0, f.loc)
	f.script(d100(50))
	if _, err := f.dailySvc().Claim(f.ctx, alice); err != nil {
		t.Fatal(err)
	}
	f.clock.now = time.Date(2026, 9, 16, 10, 0, 0, 0, f.loc)
	f.script(d100(50))
	if _, err := f.dailySvc().Claim(f.ctx, alice); err != nil {
		t.Fatalf("claim at 10:00 after a 09:00 claim should be allowed: %v", err)
	}
}

type dailyRaceStore struct {
	app.PlayerStore
}

func (dailyRaceStore) ClaimDaily(context.Context, domain.PlayerKey, int64, time.Time, time.Time) (int64, bool, error) {
	return 0, false, nil
}

func TestDailyService_RaceReportsAlreadyClaimed(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	svc := app.NewDailyService(dailyRaceStore{f.players}, f.clock, f.rng, f.loc)
	_, err := svc.Claim(f.ctx, alice)
	var already *app.DailyAlreadyClaimedError
	if !errors.As(err, &already) {
		t.Fatalf("err = %v, want DailyAlreadyClaimedError", err)
	}
}

func TestDailyService_DefaultsToUTC(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	svc := app.NewDailyService(f.players, f.clock, f.rng, nil)
	if _, err := svc.Claim(f.ctx, alice); err != nil {
		t.Fatal(err)
	}
}

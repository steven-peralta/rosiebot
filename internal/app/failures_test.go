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

var errStore = errors.New("store failure")

type failStore struct {
	app.PlayerStore
	fail map[string]bool
}

func (f failStore) failing(name string) bool { return f.fail[name] }

func (f failStore) EnsurePlayer(ctx context.Context, key domain.PlayerKey) (domain.Player, error) {
	if f.failing("EnsurePlayer") {
		return domain.Player{}, errStore
	}
	return f.PlayerStore.EnsurePlayer(ctx, key)
}

func (f failStore) OwnedSlugs(ctx context.Context, key domain.PlayerKey, slugs []string) ([]string, error) {
	if f.failing("OwnedSlugs") {
		return nil, errStore
	}
	return f.PlayerStore.OwnedSlugs(ctx, key, slugs)
}

func (f failStore) ListOwned(ctx context.Context, key domain.PlayerKey, prefix string, limit int) ([]domain.OwnedWaifu, error) {
	if f.failing("ListOwned") {
		return nil, errStore
	}
	return f.PlayerStore.ListOwned(ctx, key, prefix, limit)
}

func (f failStore) GetOwned(ctx context.Context, key domain.PlayerKey, slug string) (domain.OwnedWaifu, error) {
	if f.failing("GetOwned") {
		return domain.OwnedWaifu{}, errStore
	}
	return f.PlayerStore.GetOwned(ctx, key, slug)
}

func (f failStore) SetCoins(ctx context.Context, key domain.PlayerKey, coins int64) (int64, error) {
	if f.failing("SetCoins") {
		return 0, errStore
	}
	return f.PlayerStore.SetCoins(ctx, key, coins)
}

func (f failStore) AdjustCoins(ctx context.Context, key domain.PlayerKey, delta int64) (int64, bool, error) {
	if f.failing("AdjustCoins") {
		return 0, false, errStore
	}
	return f.PlayerStore.AdjustCoins(ctx, key, delta)
}

func (f failStore) AddOwned(ctx context.Context, key domain.PlayerKey, w domain.OwnedWaifu) (bool, error) {
	if f.failing("AddOwned") {
		return false, errStore
	}
	return f.PlayerStore.AddOwned(ctx, key, w)
}

func (f failStore) SellOwned(ctx context.Context, key domain.PlayerKey, slug string, price int64) (int64, bool, error) {
	if f.failing("SellOwned") {
		return 0, false, errStore
	}
	if f.failing("SellOwnedNotOK") {
		return 0, false, nil
	}
	return f.PlayerStore.SellOwned(ctx, key, slug, price)
}

func (f failStore) ClaimDaily(ctx context.Context, key domain.PlayerKey, amount int64, windowStart, now time.Time) (int64, bool, error) {
	if f.failing("ClaimDaily") {
		return 0, false, errStore
	}
	return f.PlayerStore.ClaimDaily(ctx, key, amount, windowStart, now)
}

func (f failStore) WithinTx(ctx context.Context, fn func(app.PlayerRepo) error) error {
	if f.failing("WithinTx") {
		return errStore
	}
	return f.PlayerStore.WithinTx(ctx, func(r app.PlayerRepo) error {
		return fn(failRepo{PlayerRepo: r, fail: f.fail})
	})
}

type failRepo struct {
	app.PlayerRepo
	fail map[string]bool
}

func (f failRepo) DebitCoins(ctx context.Context, key domain.PlayerKey, amount int64) (int64, bool, error) {
	if f.fail["DebitCoins"] {
		return 0, false, errStore
	}
	return f.PlayerRepo.DebitCoins(ctx, key, amount)
}

func (f failRepo) AddOwned(ctx context.Context, key domain.PlayerKey, w domain.OwnedWaifu) (bool, error) {
	if f.fail["AddOwned"] {
		return false, errStore
	}
	return f.PlayerRepo.AddOwned(ctx, key, w)
}

func (f failRepo) LockPlayers(ctx context.Context, keys ...domain.PlayerKey) ([]domain.Player, error) {
	if f.fail["LockPlayers"] {
		return nil, errStore
	}
	return f.PlayerRepo.LockPlayers(ctx, keys...)
}

func (f failRepo) OwnedSlugs(ctx context.Context, key domain.PlayerKey, slugs []string) ([]string, error) {
	if f.fail["TxOwnedSlugs"] {
		return nil, errStore
	}
	if f.fail["TxOwnedSlugsTarget"] && key == bob {
		return nil, errStore
	}
	return f.PlayerRepo.OwnedSlugs(ctx, key, slugs)
}

func (f failRepo) TransferOwned(ctx context.Context, from, to domain.PlayerKey, slugs []string) (int64, error) {
	if f.fail["TransferOwned"] {
		return 0, errStore
	}
	if f.fail["TransferOwnedSecond"] && from == bob {
		return 0, errStore
	}
	return f.PlayerRepo.TransferOwned(ctx, from, to, slugs)
}

func (f *fixture) failing(names ...string) failStore {
	fail := map[string]bool{}
	for _, n := range names {
		fail[n] = true
	}
	return failStore{PlayerStore: f.players, fail: fail}
}

func expectStoreErr(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, errStore) {
		t.Fatalf("err = %v, want wrapped store failure", err)
	}
}

func TestStoreFailures_Roll(t *testing.T) {
	t.Run("ensure player", func(t *testing.T) {
		f := newFixture(t)
		_, err := app.NewRollService(f.failing("EnsurePlayer"), f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil).Roll(f.ctx, alice)
		expectStoreErr(t, err)
	})
	t.Run("owned slugs", func(t *testing.T) {
		f := newFixture(t)
		f.script(d100(50))
		f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 1, 0), nil).Once()
		_, err := app.NewRollService(f.failing("OwnedSlugs"), f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil).Roll(f.ctx, alice)
		expectStoreErr(t, err)
	})
	for _, name := range []string{"WithinTx", "DebitCoins", "AddOwned"} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.script(d100(50))
			f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 1, 0), nil).Once()
			f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()
			_, err := app.NewRollService(f.failing(name), f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil).Roll(f.ctx, alice)
			expectStoreErr(t, err)
			if f.coins(alice) != domain.StartingCoins {
				t.Errorf("charged despite failure: %d", f.coins(alice))
			}
		})
	}
	t.Run("popular page error", func(t *testing.T) {
		f := newFixture(t)
		f.script(d100(5), 0)
		f.source.EXPECT().PopularPage(mock.Anything, 1).Return(app.PopularPage{}, errors.New("boom")).Once()
		if _, err := f.roll().Roll(f.ctx, alice); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("wotd error", func(t *testing.T) {
		f := newFixture(t)
		f.ranking.Set(rankingOf(200))
		f.script(d100(1))
		svc := app.NewRollService(f.players, f.source, f.ranking, app.NewWotdService(failingDaily{getErr: errors.New("boom")}, f.ranking, f.clock, f.rng, f.loc, nil), f.banner(), f.clock, f.rng, 0, nil)
		if _, err := svc.Roll(f.ctx, alice); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestStoreFailures_Daily(t *testing.T) {
	f := newFixture(t)
	_, err := app.NewDailyService(f.failing("EnsurePlayer"), f.clock, f.rng, f.loc).Claim(f.ctx, alice)
	expectStoreErr(t, err)

	f = newFixture(t)
	f.script(d100(50))
	_, err = app.NewDailyService(f.failing("ClaimDaily"), f.clock, f.rng, f.loc).Claim(f.ctx, alice)
	expectStoreErr(t, err)
}

func TestStoreFailures_CoinsAndInventory(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem")

	_, err := app.NewCoinsService(f.failing("EnsurePlayer")).Balance(f.ctx, alice)
	expectStoreErr(t, err)

	inv := app.NewInventoryService(f.failing("EnsurePlayer"))
	_, err = inv.List(f.ctx, alice)
	expectStoreErr(t, err)

	inv = app.NewInventoryService(f.failing("ListOwned"))
	_, err = inv.List(f.ctx, alice)
	expectStoreErr(t, err)
	_, err = inv.Suggest(f.ctx, alice, "", 5)
	expectStoreErr(t, err)

	inv = app.NewInventoryService(f.failing("GetOwned"))
	_, _, err = inv.Owns(f.ctx, alice, "rem")
	expectStoreErr(t, err)
	_, err = inv.Sell(f.ctx, alice, "rem")
	expectStoreErr(t, err)

	inv = app.NewInventoryService(f.failing("SellOwned"))
	_, err = inv.Sell(f.ctx, alice, "rem")
	expectStoreErr(t, err)

	inv = app.NewInventoryService(f.failing("SellOwnedNotOK"))
	if _, err := inv.Sell(f.ctx, alice, "rem"); !errors.Is(err, app.ErrNotOwned) {
		t.Errorf("sell race err = %v, want ErrNotOwned", err)
	}
}

func TestStoreFailures_Trade(t *testing.T) {
	setup := func(t *testing.T) *fixture {
		f := newFixture(t)
		f.give(alice, "rem")
		f.give(bob, "ram")
		return f
	}
	o := offer([]string{"rem"}, []string{"ram"})

	t.Run("propose ensure sender", func(t *testing.T) {
		f := setup(t)
		_, err := app.NewTradeService(f.failing("EnsurePlayer")).Propose(f.ctx, o)
		expectStoreErr(t, err)
	})
	t.Run("propose owned slugs", func(t *testing.T) {
		f := setup(t)
		_, err := app.NewTradeService(f.failing("OwnedSlugs")).Propose(f.ctx, o)
		expectStoreErr(t, err)
	})
	t.Run("propose lookup", func(t *testing.T) {
		f := setup(t)
		_, err := app.NewTradeService(f.failing("GetOwned")).Propose(f.ctx, o)
		expectStoreErr(t, err)
	})
	for _, name := range []string{"WithinTx", "LockPlayers", "TxOwnedSlugs", "TxOwnedSlugsTarget", "TransferOwned", "TransferOwnedSecond"} {
		t.Run("accept "+name, func(t *testing.T) {
			f := setup(t)
			err := app.NewTradeService(f.failing(name)).Accept(f.ctx, o)
			expectStoreErr(t, err)
			if !f.owns(alice, "rem") || !f.owns(bob, "ram") {
				t.Error("failed accept must not move anything")
			}
		})
	}
}

func TestClockHelpers(t *testing.T) {
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := app.ClockFunc(func() time.Time { return fixed }).Now(); !got.Equal(fixed) {
		t.Errorf("ClockFunc.Now = %v", got)
	}
	if time.Since(app.SystemClock().Now()) > time.Minute {
		t.Error("SystemClock should track wall time")
	}
}

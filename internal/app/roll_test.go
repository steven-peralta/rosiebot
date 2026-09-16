package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestRollService_InsufficientCoinsNoAPICall(t *testing.T) {
	f := newFixture(t)
	f.give(alice)
	if _, ok, err := f.players.DebitCoins(f.ctx, alice, domain.StartingCoins); err != nil || !ok {
		t.Fatal("setup debit failed")
	}

	_, err := f.roll().Roll(f.ctx, alice)
	if !errors.Is(err, app.ErrInsufficientCoins) {
		t.Fatalf("err = %v, want ErrInsufficientCoins", err)
	}
	f.source.AssertNotCalled(t, "Random", mock.Anything)
}

func TestRollService_RegularRoll(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 500, 20), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollRegular || res.Waifu.Slug != "rem" || res.Balance != 0 || res.Attempts != 1 || res.Ranked != nil {
		t.Errorf("result = %+v", res)
	}
	if f.coins(alice) != 0 || !f.owns(alice, "rem") {
		t.Errorf("state after roll: coins=%d owns=%v", f.coins(alice), f.owns(alice, "rem"))
	}
}

func TestRollService_CriticalUsesRankedSet(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.script(d100(7), 3)
	f.source.EXPECT().Get(mock.Anything, "ranked-003").Return(detail("ranked-003"), nil).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollCritical || res.Ranked == nil || res.Ranked.Position != 4 || res.Ranked.Stars == 0 {
		t.Errorf("result = %+v ranked=%+v", res, res.Ranked)
	}
	f.source.AssertNotCalled(t, "Random", mock.Anything)
	f.source.AssertNotCalled(t, "PopularPage", mock.Anything, mock.Anything)
}

func TestRollService_CriticalFallbackBeforeSnapshot(t *testing.T) {
	f := newFixture(t)
	f.script(d100(2), 41, 1)
	f.source.EXPECT().PopularPage(mock.Anything, 42).Return(app.PopularPage{Page: 42, LastPage: 5113, Rows: []domain.WaifuSummary{
		summary("thin", 60, 40),
		summary("fat-a", 900, 50),
		summary("fat-b", 800, 10),
	}}, nil).Once()
	f.source.EXPECT().Get(mock.Anything, "fat-b").Return(detail("fat-b"), nil).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollCritical || res.Waifu.Slug != "fat-b" || res.Ranked != nil {
		t.Errorf("result = %+v", res)
	}
}

func TestRollService_CriticalFallbackDegradesWhenPageHasNoRankedRows(t *testing.T) {
	f := newFixture(t)
	f.script(d100(13), 999)
	f.source.EXPECT().PopularPage(mock.Anything, 1000).Return(app.PopularPage{Page: 1000, Rows: []domain.WaifuSummary{summary("edge", 60, 40)}}, nil).Once()
	f.source.EXPECT().Random(mock.Anything).Return(summary("plain", 5, 1), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "plain").Return(detail("plain"), nil).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Waifu.Slug != "plain" {
		t.Errorf("result = %+v", res)
	}
}

func TestRollService_WaifuOfTheDayRoll(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.script(d100(1), 0)
	f.source.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, slug string) (domain.Waifu, error) {
		return detail(slug), nil
	}).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollWaifuOfTheDay {
		t.Errorf("kind = %v", res.Kind)
	}
	today, err := f.wotd().Today(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if today.Waifu.Slug != res.Waifu.Slug {
		t.Errorf("rolled %s but waifu of the day is %s", res.Waifu.Slug, today.Waifu.Slug)
	}
}

func TestRollService_RerollOnOwned(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "owned")
	f.script(d100(50), d100(60))
	f.source.EXPECT().Random(mock.Anything).Return(summary("owned", 1, 0), nil).Once()
	f.source.EXPECT().Random(mock.Anything).Return(summary("fresh", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "fresh").Return(detail("fresh"), nil).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || res.Waifu.Slug != "fresh" {
		t.Errorf("result = %+v", res)
	}
	f.source.AssertNotCalled(t, "Get", mock.Anything, "owned")
}

func TestRollService_RerollCapUncharged(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "owned")
	for range domain.MaxRerollAttempts {
		f.script(d100(50))
	}
	f.source.EXPECT().Random(mock.Anything).Return(summary("owned", 1, 0), nil).Times(domain.MaxRerollAttempts)

	_, err := f.roll().Roll(f.ctx, alice)
	if !errors.Is(err, app.ErrRollExhausted) {
		t.Fatalf("err = %v, want ErrRollExhausted", err)
	}
	if f.coins(alice) != domain.StartingCoins {
		t.Errorf("player was charged: %d", f.coins(alice))
	}
}

func TestRollService_LostRaceRerolls(t *testing.T) {
	f := newFixture(t)
	f.give(alice)
	f.script(d100(50), d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("contested", 1, 0), nil).Once()
	f.source.EXPECT().Random(mock.Anything).Return(summary("second", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "contested").Return(detail("contested"), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "second").Return(detail("second"), nil).Once()

	raced := false
	store := hookedStore{PlayerStore: f.players, beforeTx: func() {
		if raced {
			return
		}
		raced = true
		if _, err := f.players.AddOwned(f.ctx, alice, domain.OwnedFromSummary(summary("contested", 1, 0), f.clock.now)); err != nil {
			t.Fatal(err)
		}
	}}
	svc := app.NewRollService(store, f.source, f.ranking, f.wotd(), f.clock, f.rng, 0, nil)

	res, err := svc.Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || res.Waifu.Slug != "second" || f.coins(alice) != 0 {
		t.Errorf("result = %+v coins=%d", res, f.coins(alice))
	}
}

func TestRollService_DetailFailureDoesNotCharge(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(domain.Waifu{}, errors.New("boom")).Once()

	if _, err := f.roll().Roll(f.ctx, alice); err == nil {
		t.Fatal("expected error")
	}
	if f.coins(alice) != domain.StartingCoins || f.owns(alice, "rem") {
		t.Errorf("state changed on failure: coins=%d owns=%v", f.coins(alice), f.owns(alice, "rem"))
	}
}

func TestRollService_SourceFailurePropagates(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(domain.WaifuSummary{}, errors.New("mwl down")).Once()
	if _, err := f.roll().Roll(f.ctx, alice); err == nil {
		t.Fatal("expected error")
	}
}

func TestRollService_ConcurrentDebitLosesWhenBalanceGone(t *testing.T) {
	f := newFixture(t)
	f.give(alice)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()
	store := hookedStore{PlayerStore: f.players, beforeTx: func() {
		if _, ok, err := f.players.DebitCoins(f.ctx, alice, domain.StartingCoins); err != nil || !ok {
			t.Fatal("setup debit failed")
		}
	}}
	svc := app.NewRollService(store, f.source, f.ranking, f.wotd(), f.clock, f.rng, 0, nil)

	_, err := svc.Roll(f.ctx, alice)
	if !errors.Is(err, app.ErrInsufficientCoins) {
		t.Fatalf("err = %v, want ErrInsufficientCoins", err)
	}
	if f.owns(alice, "rem") {
		t.Error("waifu granted without payment")
	}
}

func TestRolled_CoversFullD100(t *testing.T) {
	seen := map[int]bool{}
	for v := range 100 {
		seen[app.Rolled(&seqRandom{t: t, vals: []int{v}})] = true
	}
	if len(seen) != 100 || !seen[1] || !seen[100] {
		t.Errorf("Rolled should map IntN(100) onto 1..100, got %d distinct", len(seen))
	}
}

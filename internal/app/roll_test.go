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
	f.source.EXPECT().PopularPage(mock.MatchedBy(func(ctx context.Context) bool { return !app.IsBackground(ctx) }), 42).Return(app.PopularPage{Page: 42, LastPage: 5113, Rows: []domain.WaifuSummary{
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

func TestRollService_WaifuOfTheDayWithoutRankingDegradesToCritical(t *testing.T) {
	f := newFixture(t)
	f.script(d100(1), 999, 0)
	f.source.EXPECT().PopularPage(mock.Anything, 1000).Return(app.PopularPage{Rows: []domain.WaifuSummary{summary("popular", 500, 20)}, Page: 1000, LastPage: 1001}, nil).Once()
	f.source.EXPECT().Get(mock.Anything, "popular").Return(detail("popular"), nil).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollWaifuOfTheDay || res.Waifu.Slug != "popular" {
		t.Errorf("result = %+v", res)
	}
	f.source.AssertNotCalled(t, "Daily", mock.Anything)
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
	svc := app.NewRollService(store, f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil)

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
	svc := app.NewRollService(store, f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil)

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

func (f *fixture) fund(key domain.PlayerKey, amount int64) {
	f.t.Helper()
	f.give(key)
	if _, ok, err := f.players.ClaimDaily(f.ctx, key, amount, f.clock.now.Add(-time.Hour), f.clock.now); err != nil || !ok {
		f.t.Fatalf("setup funding failed: ok=%v err=%v", ok, err)
	}
}

func TestRollService_BannerRollHitsFeatured(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-050", "ranked-000", "ranked-020", "ranked-005", "ranked-100")
	f.fund(alice, domain.BannerRollCost-domain.StartingCoins)
	f.script(d100(5), 2)
	f.source.EXPECT().Get(mock.Anything, "ranked-020").Return(detail("ranked-020"), nil).Once()

	res, err := f.roll().RollBanner(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollBanner || !res.Banner || res.Waifu.Slug != "ranked-020" || res.Balance != 0 || res.Ranked == nil || res.Ranked.Position != 21 {
		t.Errorf("result = %+v ranked=%+v", res, res.Ranked)
	}
	if f.coins(alice) != 0 || !f.owns(alice, "ranked-020") {
		t.Errorf("state after banner roll: coins=%d owns=%v", f.coins(alice), f.owns(alice, "ranked-020"))
	}
}

func TestRollService_BannerCriticalAndRegularStillCost400(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	f.fund(alice, 2*domain.BannerRollCost-domain.StartingCoins)

	f.script(d100(15), 7)
	f.source.EXPECT().Get(mock.Anything, "ranked-007").Return(detail("ranked-007"), nil).Once()
	res, err := f.roll().RollBanner(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollCritical || !res.Banner || res.Balance != domain.BannerRollCost {
		t.Errorf("critical on banner = %+v", res)
	}

	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()
	res, err = f.roll().RollBanner(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollRegular || !res.Banner || res.Balance != 0 {
		t.Errorf("regular on banner = %+v", res)
	}
}

func TestRollService_BannerDegradesToCriticalWhenAllOwned(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	f.give(alice, "ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	f.fund(alice, domain.BannerRollCost-domain.StartingCoins)
	f.script(d100(3), 9)
	f.source.EXPECT().Get(mock.Anything, "ranked-009").Return(detail("ranked-009"), nil).Once()

	res, err := f.roll().RollBanner(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != domain.RollCritical || !res.Banner || res.Waifu.Slug != "ranked-009" || f.coins(alice) != 0 {
		t.Errorf("degraded result = %+v coins=%d", res, f.coins(alice))
	}
}

func TestRollService_BannerWithoutBannerUnchargedNoAPICall(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.fund(alice, domain.BannerRollCost)
	if _, err := f.roll().RollBanner(f.ctx, alice); !errors.Is(err, app.ErrNoBanner) {
		t.Fatalf("err = %v, want ErrNoBanner", err)
	}
	if f.coins(alice) != domain.BannerRollCost+domain.StartingCoins {
		t.Errorf("player was charged: %d", f.coins(alice))
	}
	svc := app.NewRollService(f.players, f.source, f.ranking, f.wotd(), nil, f.clock, f.rng, 0, nil)
	if _, err := svc.RollBanner(f.ctx, alice); !errors.Is(err, app.ErrNoBanner) {
		t.Fatalf("nil banner service err = %v, want ErrNoBanner", err)
	}
	f.source.AssertNotCalled(t, "Random", mock.Anything)
	f.source.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
}

func TestRollService_BannerInsufficientCoins(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	f.fund(alice, domain.BannerRollCost-domain.StartingCoins-1)
	if _, err := f.roll().RollBanner(f.ctx, alice); !errors.Is(err, app.ErrInsufficientCoins) {
		t.Fatalf("err = %v, want ErrInsufficientCoins", err)
	}
	if f.coins(alice) != domain.BannerRollCost-1 {
		t.Errorf("coins = %d", f.coins(alice))
	}
}

func TestRollService_BannerRerollOnOwnedAndExhausted(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-000", "ranked-005", "ranked-006", "ranked-007", "ranked-008")
	f.give(alice, "ranked-000")
	f.fund(alice, domain.BannerRollCost-domain.StartingCoins)
	f.script(d100(15), 0, d100(5), 0)
	f.source.EXPECT().Get(mock.Anything, "ranked-005").Return(detail("ranked-005"), nil).Once()

	res, err := f.roll().RollBanner(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || res.Kind != domain.RollBanner || res.Waifu.Slug != "ranked-005" {
		t.Errorf("result = %+v", res)
	}

	f.fund(bob, domain.BannerRollCost-domain.StartingCoins)
	f.give(bob, "ranked-000")
	for range domain.MaxRerollAttempts {
		f.script(d100(15), 0)
	}
	if _, err := f.roll().RollBanner(f.ctx, bob); !errors.Is(err, app.ErrRollExhausted) {
		t.Fatalf("err = %v, want ErrRollExhausted", err)
	}
	if f.coins(bob) != domain.BannerRollCost {
		t.Errorf("bob was charged: %d", f.coins(bob))
	}
}

func TestRollService_BannerLostRaceRerolls(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-000", "ranked-005", "ranked-006", "ranked-007", "ranked-008")
	f.fund(alice, domain.BannerRollCost-domain.StartingCoins)
	f.script(d100(5), 0, d100(5), 0, d100(5), 1)
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(detail("ranked-000"), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-005").Return(detail("ranked-005"), nil).Once()

	raced := false
	store := hookedStore{PlayerStore: f.players, beforeTx: func() {
		if raced {
			return
		}
		raced = true
		if _, err := f.players.AddOwned(f.ctx, alice, domain.OwnedFromSummary(summary("ranked-000", 1, 0), f.clock.now)); err != nil {
			t.Fatal(err)
		}
	}}
	svc := app.NewRollService(store, f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil)

	res, err := svc.RollBanner(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 3 || res.Waifu.Slug != "ranked-005" || f.coins(alice) != 0 {
		t.Errorf("result = %+v coins=%d", res, f.coins(alice))
	}
}

func TestRollService_MissingUpstreamWaifuRerolls(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50), d100(60))
	f.source.EXPECT().Random(mock.Anything).Return(summary("gone", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "gone").Return(domain.Waifu{}, app.ErrNotFound).Once()
	f.source.EXPECT().Random(mock.Anything).Return(summary("here", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "here").Return(detail("here"), nil).Once()

	res, err := f.roll().Roll(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || res.Waifu.Slug != "here" || f.coins(alice) != 0 || f.owns(alice, "gone") {
		t.Errorf("result = %+v coins=%d", res, f.coins(alice))
	}
}

func TestRollService_BannerDropsMissingCharacterFromPool(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	f.seedBanner("ranked-000", "ranked-005", "ranked-006", "ranked-007", "ranked-008")
	f.fund(alice, domain.BannerRollCost-domain.StartingCoins)
	f.script(d100(5), 0, d100(5), 0)
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(domain.Waifu{}, app.ErrNotFound).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-005").Return(detail("ranked-005"), nil).Once()

	res, err := f.roll().RollBanner(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if res.Attempts != 2 || res.Kind != domain.RollBanner || res.Waifu.Slug != "ranked-005" || f.coins(alice) != 0 {
		t.Errorf("a missing featured character should be dropped so the next banner hit lands elsewhere: %+v coins=%d", res, f.coins(alice))
	}
}

func TestRollService_RecordsHistory(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem", 500, 20), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Once()
	if _, err := f.roll().Roll(f.ctx, alice); err != nil {
		t.Fatal(err)
	}
	recent, err := f.players.RecentRolls(f.ctx, alice, 5)
	if err != nil || len(recent) != 1 || recent[0].Slug != "rem" || recent[0].Kind != domain.RollRegular || recent[0].Cost != domain.RollCost || !recent[0].At.Equal(f.clock.now) {
		t.Errorf("history = %+v %v", recent, err)
	}

	f.fund(bob, domain.RollCost)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("ram", 1, 0), nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ram").Return(detail("ram"), nil).Once()
	broken := app.NewRollService(failStore{PlayerStore: f.players, fail: map[string]bool{"RecordRoll": true}}, f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil)
	if _, err := broken.Roll(f.ctx, bob); !errors.Is(err, errStore) {
		t.Fatalf("record failure should fail the roll, got %v", err)
	}
	if f.owns(bob, "ram") || f.coins(bob) != domain.StartingCoins+domain.RollCost {
		t.Error("a failed history write must roll the whole pull back")
	}
}

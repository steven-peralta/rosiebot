package app_test

import (
	"errors"
	"testing"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestStatsService_ProfileAndLeaderboard(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	favs := memory.NewFavoriteStore()
	svc := app.NewStatsService(f.players, f.ranking, favs)

	f.give(alice, "ranked-000", "ranked-005", "ranked-100", "plain-a")
	f.give(bob, "ranked-001", "plain-b", "plain-c", "plain-d", "plain-e")
	f.give(domain.PlayerKey{GuildID: "guild", UserID: "carol"})
	if _, err := f.players.SetCoins(f.ctx, bob, 5000); err != nil {
		t.Fatal(err)
	}
	if _, err := favs.Add(f.ctx, alice, domain.Favorite{Kind: domain.FavoriteWaifu, Slug: "rem", Name: "Rem"}); err != nil {
		t.Fatal(err)
	}
	if err := f.players.RecordRoll(f.ctx, alice, domain.RollRecord{Slug: "plain-a", Name: "Name plain-a", Kind: domain.RollRegular, Cost: 200, At: f.clock.now}); err != nil {
		t.Fatal(err)
	}
	if err := f.players.RecordRoll(f.ctx, alice, domain.RollRecord{Slug: "ranked-000", Name: "Name ranked-000", Kind: domain.RollCritical, Cost: 200, At: f.clock.now.Add(1)}); err != nil {
		t.Fatal(err)
	}

	p, err := svc.Profile(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if p.Coins != domain.StartingCoins || p.Owned != 4 || p.Tiers[5] != 1 || p.Tiers[4] != 1 || p.Tiers[1] != 1 || p.Tiers[0] != 1 || p.Value != 1000+500+150+100 {
		t.Errorf("profile counts = %+v", p)
	}
	if p.Best == nil || p.Best.Slug != "ranked-000" || p.Rolls != 2 || p.LastRoll == nil || p.LastRoll.Slug != "ranked-000" || p.Favorites != 1 {
		t.Errorf("profile extras = %+v best=%+v last=%+v", p, p.Best, p.LastRoll)
	}
	if p.ValueRank != 1 || p.CollectionRank != 2 || p.Players != 3 {
		t.Errorf("profile ranks = value %d collection %d of %d", p.ValueRank, p.CollectionRank, p.Players)
	}

	cases := map[domain.Metric][]app.Standing{
		domain.MetricCoins:      {{"bob", 5000}, {"alice", 200}, {"carol", 200}},
		domain.MetricCollection: {{"bob", 5}, {"alice", 4}, {"carol", 0}},
		domain.MetricValue:      {{"alice", 1750}, {"bob", 1000 + 400}, {"carol", 0}},
		domain.MetricStars:      {{"alice", 2}, {"bob", 1}, {"carol", 0}},
	}
	for metric, want := range cases {
		board, err := svc.Leaderboard(f.ctx, "guild", metric)
		if err != nil {
			t.Fatalf("%s: %v", metric, err)
		}
		if len(board.Standings) != len(want) {
			t.Fatalf("%s standings = %+v", metric, board.Standings)
		}
		for i := range want {
			if board.Standings[i] != want[i] {
				t.Errorf("%s[%d] = %+v, want %+v", metric, i, board.Standings[i], want[i])
			}
		}
	}
	if rank, ok := (app.Board{}).Rank("nobody"); ok || rank != 0 {
		t.Error("missing rank")
	}
	if _, err := svc.Leaderboard(f.ctx, "guild", domain.Metric("fame")); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("bad metric = %v", err)
	}
	if board, _ := svc.Leaderboard(f.ctx, "empty-guild", domain.MetricValue); len(board.Standings) != 0 {
		t.Errorf("empty guild = %+v", board)
	}

	for _, name := range []string{"EnsurePlayer", "ListOwned"} {
		broken := app.NewStatsService(failStore{PlayerStore: f.players, fail: map[string]bool{name: true}}, f.ranking, favs)
		if _, err := broken.Profile(f.ctx, alice); !errors.Is(err, errStore) {
			t.Errorf("%s failure should surface, got %v", name, err)
		}
	}
	if _, err := app.NewStatsService(f.players, f.ranking, failingFavorites{err: errStore}).Profile(f.ctx, alice); !errors.Is(err, errStore) {
		t.Errorf("favorites failure should surface, got %v", err)
	}
}

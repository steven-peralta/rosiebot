package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestErrorsPropagateOnCancelledContext(t *testing.T) {
	alice, bob := keys(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewStore(testPool, nil)
	now := time.Now()

	checks := map[string]func() error{
		"EnsurePlayer":  func() error { _, err := s.EnsurePlayer(ctx, alice); return err },
		"GetPlayer":     func() error { _, err := s.GetPlayer(ctx, alice); return err },
		"LockPlayers":   func() error { _, err := s.LockPlayers(ctx, alice, bob); return err },
		"DebitCoins":    func() error { _, _, err := s.DebitCoins(ctx, alice, 1); return err },
		"ClaimDaily":    func() error { _, _, err := s.ClaimDaily(ctx, alice, 1, now, now); return err },
		"AddOwned":      func() error { _, err := s.AddOwned(ctx, alice, owned("x", now)); return err },
		"GetOwned":      func() error { _, err := s.GetOwned(ctx, alice, "x"); return err },
		"SellOwned":     func() error { _, _, err := s.SellOwned(ctx, alice, "x", 1); return err },
		"OwnedSlugs":    func() error { _, err := s.OwnedSlugs(ctx, alice, []string{"x"}); return err },
		"TransferOwned": func() error { _, err := s.TransferOwned(ctx, alice, bob, []string{"x"}); return err },
		"ListOwned":     func() error { _, err := s.ListOwned(ctx, alice, "", 0); return err },
		"CountOwned":    func() error { _, err := s.CountOwned(ctx, alice); return err },
		"RankingLoad":   func() error { _, err := NewRankingStore(testPool).LoadLatest(ctx); return err },
		"RankingSave": func() error {
			return NewRankingStore(testPool).Save(ctx, domain.BuildRanking(nil, domain.DefaultMinVotes, now, 1))
		},
		"DailyGet": func() error { _, err := NewDailyStore(testPool).Get(ctx, now); return err },
		"DailyPut": func() error {
			_, err := NewDailyStore(testPool).Put(ctx, now, domain.WaifuSummary{Slug: "x"})
			return err
		},
	}
	for name, fn := range checks {
		if err := fn(); err == nil {
			t.Errorf("%s: expected an error on a cancelled context", name)
		}
	}
}

func TestRankingStore_SaveFailsInsideTx(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	rs := NewRankingStore(testPool)
	r := domain.BuildRanking([]domain.WaifuSummary{{Slug: "a", Likes: 500, Trash: 1}}, domain.DefaultMinVotes, time.Now(), 1)
	if err := rs.Save(ctx, r); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, "ALTER TABLE ranking_rows ADD CONSTRAINT no_rows CHECK (position < 0) NOT VALID"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = testPool.Exec(ctx, "ALTER TABLE ranking_rows DROP CONSTRAINT no_rows") }()
	if err := rs.Save(ctx, r); err == nil {
		t.Error("save should surface the copy failure")
	}
	latest, err := rs.LoadLatest(ctx)
	if err != nil || latest.Len() != 1 {
		t.Errorf("failed save must not disturb the latest snapshot: %v %v", latest, err)
	}
}

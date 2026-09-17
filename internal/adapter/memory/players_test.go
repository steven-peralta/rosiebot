package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

var (
	alice = domain.PlayerKey{GuildID: "g", UserID: "alice"}
	bob   = domain.PlayerKey{GuildID: "g", UserID: "bob"}
)

func owned(slug string, at time.Time) domain.OwnedWaifu {
	return domain.OwnedWaifu{Slug: slug, Name: "N " + slug, AcquiredAt: at}
}

func TestPlayerStore_TxRollsBackOnError(t *testing.T) {
	ctx := context.Background()
	s := NewPlayerStore(nil)
	if _, err := s.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := s.WithinTx(ctx, func(r app.PlayerRepo) error {
		if _, ok, err := r.DebitCoins(ctx, alice, 50); err != nil || !ok {
			t.Fatal("debit inside tx failed")
		}
		if _, err := r.AddOwned(ctx, alice, owned("rem", time.Now())); err != nil {
			t.Fatal(err)
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	p, _ := s.GetPlayer(ctx, alice)
	n, _ := s.CountOwned(ctx, alice)
	if p.Coins != domain.StartingCoins || n != 0 {
		t.Errorf("rollback failed: coins=%d owned=%d", p.Coins, n)
	}
}

func TestPlayerStore_TxCommits(t *testing.T) {
	ctx := context.Background()
	s := NewPlayerStore(nil)
	err := s.WithinTx(ctx, func(r app.PlayerRepo) error {
		if _, err := r.EnsurePlayer(ctx, alice); err != nil {
			return err
		}
		if _, err := r.GetPlayer(ctx, alice); err != nil {
			return err
		}
		if _, err := r.LockPlayers(ctx, alice); err != nil {
			return err
		}
		if _, err := r.AddOwned(ctx, alice, owned("rem", time.Now())); err != nil {
			return err
		}
		if _, err := r.GetOwned(ctx, alice, "rem"); err != nil {
			return err
		}
		if _, _, err := r.ClaimDaily(ctx, alice, 400, time.Now().Add(-time.Hour), time.Now()); err != nil {
			return err
		}
		if _, err := r.ListOwned(ctx, alice, "", 0); err != nil {
			return err
		}
		if n, err := r.CountOwned(ctx, alice); err != nil || n != 1 {
			t.Errorf("count = %d, %v", n, err)
		}
		if _, ok, err := r.SellOwned(ctx, alice, "rem", 100); err != nil || !ok {
			t.Errorf("sell = %v, %v", ok, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := s.GetPlayer(ctx, alice)
	if p.Coins != domain.StartingCoins+400+100 {
		t.Errorf("coins = %d", p.Coins)
	}
}

func TestPlayerStore_UnknownPlayerBehaviour(t *testing.T) {
	ctx := context.Background()
	s := NewPlayerStore(nil)
	if _, err := s.GetPlayer(ctx, alice); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("GetPlayer = %v", err)
	}
	if _, err := s.LockPlayers(ctx, alice); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("LockPlayers = %v", err)
	}
	if _, ok, _ := s.DebitCoins(ctx, alice, 1); ok {
		t.Error("debit of unknown player should not succeed")
	}
	if _, ok, _ := s.ClaimDaily(ctx, alice, 1, time.Time{}, time.Time{}); ok {
		t.Error("claim of unknown player should not succeed")
	}
	if _, err := s.SetCoins(ctx, alice, 5); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("SetCoins = %v", err)
	}
	if _, ok, _ := s.AdjustCoins(ctx, alice, 5); ok {
		t.Error("adjust of unknown player should not succeed")
	}
	if _, err := s.AddOwned(ctx, alice, owned("x", time.Time{})); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("AddOwned = %v", err)
	}
	if _, ok, _ := s.SellOwned(ctx, alice, "x", 1); ok {
		t.Error("sell of unknown player should not succeed")
	}
	if _, err := s.GetOwned(ctx, alice, "x"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("GetOwned = %v", err)
	}
}

func TestPlayerStore_AddOwnedDuplicateAndTransfer(t *testing.T) {
	ctx := context.Background()
	s := NewPlayerStore(nil)
	for _, k := range []domain.PlayerKey{alice, bob} {
		if _, err := s.EnsurePlayer(ctx, k); err != nil {
			t.Fatal(err)
		}
	}
	if ins, _ := s.AddOwned(ctx, alice, owned("rem", time.Time{})); !ins {
		t.Fatal("first insert should succeed")
	}
	if ins, _ := s.AddOwned(ctx, alice, owned("rem", time.Time{})); ins {
		t.Fatal("duplicate insert should report false")
	}
	if _, err := s.AddOwned(ctx, bob, owned("rem", time.Time{})); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TransferOwned(ctx, alice, bob, []string{"rem"}); !errors.Is(err, app.ErrAlreadyOwned) {
		t.Errorf("transfer onto duplicate = %v", err)
	}
	moved, err := s.TransferOwned(ctx, alice, bob, []string{"missing"})
	if err != nil || moved != 0 {
		t.Errorf("transfer of missing = %d, %v", moved, err)
	}
	players, err := s.LockPlayers(ctx, bob, alice)
	if err != nil || players[0].Key != alice || players[1].Key != bob {
		t.Errorf("LockPlayers order = %+v, %v", players, err)
	}
	slugs, _ := s.OwnedSlugs(ctx, bob, []string{"rem", "nope"})
	if len(slugs) != 1 || slugs[0] != "rem" {
		t.Errorf("OwnedSlugs = %v", slugs)
	}
}

func TestPlayerStore_ListOwnedPrefixAndLimit(t *testing.T) {
	ctx := context.Background()
	s := NewPlayerStore(nil)
	if _, err := s.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, slug := range []string{"rem", "ram", "emilia"} {
		if _, err := s.AddOwned(ctx, alice, owned(slug, base.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	all, _ := s.ListOwned(ctx, alice, "", 0)
	if len(all) != 3 || all[0].Slug != "rem" || all[2].Slug != "emilia" {
		t.Errorf("all = %+v", all)
	}
	r, _ := s.ListOwned(ctx, alice, "n r", 0)
	if len(r) != 2 {
		t.Errorf("prefix = %+v", r)
	}
	one, _ := s.ListOwned(ctx, alice, "", 1)
	if len(one) != 1 {
		t.Errorf("limit = %+v", one)
	}
	if _, ok, _ := s.SellOwned(ctx, bob, "rem", 1); ok {
		t.Error("selling someone else's waifu should fail")
	}
}

func TestRankingStoreAndHolder(t *testing.T) {
	ctx := context.Background()
	rs := NewRankingStore()
	if _, err := rs.LoadLatest(ctx); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("empty LoadLatest = %v", err)
	}
	r := domain.BuildRanking([]domain.WaifuSummary{{Slug: "a", Likes: 500, Trash: 1}}, domain.DefaultMinVotes, time.Time{}, 0)
	if err := rs.Save(ctx, r); err != nil {
		t.Fatal(err)
	}
	got, err := rs.LoadLatest(ctx)
	if err != nil || got.Len() != 1 {
		t.Errorf("LoadLatest = %v, %v", got, err)
	}
	h := NewRankingHolder(nil)
	if h.Current() != nil {
		t.Error("holder should start empty")
	}
	h.Set(r)
	if h.Current().Len() != 1 {
		t.Error("holder should publish the ranking")
	}
}

func TestDailyStore_Replace(t *testing.T) {
	ctx := context.Background()
	s := NewDailyStore()
	day := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	if err := s.Replace(ctx, day, domain.WaifuSummary{Slug: "fresh"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(ctx, day); got.Slug != "fresh" {
		t.Errorf("replace on empty = %+v", got)
	}
	if err := s.Replace(ctx, day, domain.WaifuSummary{Slug: "newer"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(ctx, day); got.Slug != "newer" {
		t.Errorf("replace should overwrite, got %+v", got)
	}
}

func TestDailyStore(t *testing.T) {
	ctx := context.Background()
	d := NewDailyStore()
	day := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	if _, err := d.Get(ctx, day); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("Get on empty = %v", err)
	}
	if w, err := d.Put(ctx, day, domain.WaifuSummary{Slug: "a"}); err != nil || w.Slug != "a" {
		t.Errorf("Put = %+v, %v", w, err)
	}
	if w, err := d.Put(ctx, day, domain.WaifuSummary{Slug: "b"}); err != nil || w.Slug != "a" {
		t.Errorf("second Put = %+v, %v", w, err)
	}
	if w, err := d.Get(ctx, day); err != nil || w.Slug != "a" {
		t.Errorf("Get = %+v, %v", w, err)
	}
}

func TestBannerStore(t *testing.T) {
	ctx := context.Background()
	s := NewBannerStore()
	week := time.Date(2026, 9, 14, 15, 0, 0, 0, time.UTC)
	if _, err := s.Get(ctx, week); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("miss = %v", err)
	}
	first, err := s.Put(ctx, domain.Banner{WeekStart: week, Series: domain.Series{Slug: "first"}})
	if err != nil || first.Series.Slug != "first" {
		t.Fatal(err)
	}
	second, err := s.Put(ctx, domain.Banner{WeekStart: week, Series: domain.Series{Slug: "second"}})
	if err != nil || second.Series.Slug != "first" {
		t.Errorf("second put = %+v %v", second, err)
	}
	got, err := s.Get(ctx, week.In(time.FixedZone("x", -5*3600)))
	if err != nil || got.Series.Slug != "first" {
		t.Errorf("same instant in another zone = %+v %v", got, err)
	}
	if err := s.Replace(ctx, domain.Banner{WeekStart: week, Series: domain.Series{Slug: "third"}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(ctx, week); got.Series.Slug != "third" {
		t.Errorf("replace should overwrite, got %s", got.Series.Slug)
	}
}

func TestPlayerStore_SetAndAdjustCoins(t *testing.T) {
	ctx := context.Background()
	s := NewPlayerStore(nil)
	if _, err := s.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}
	if got, err := s.SetCoins(ctx, alice, 50); err != nil || got != 50 {
		t.Errorf("SetCoins = %d %v", got, err)
	}
	if got, ok, err := s.AdjustCoins(ctx, alice, -50); err != nil || !ok || got != 0 {
		t.Errorf("AdjustCoins to zero = %d %v %v", got, ok, err)
	}
	if _, ok, _ := s.AdjustCoins(ctx, alice, -1); ok {
		t.Error("adjust below zero should fail")
	}
	err := s.WithinTx(ctx, func(r app.PlayerRepo) error {
		if _, err := r.SetCoins(ctx, alice, 10); err != nil {
			return err
		}
		_, ok, err := r.AdjustCoins(ctx, alice, 5)
		if err != nil || !ok {
			return errors.New("adjust in tx failed")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := s.GetPlayer(ctx, alice); p.Coins != 15 {
		t.Errorf("coins after tx = %d", p.Coins)
	}
}

func TestFavoriteStore(t *testing.T) {
	ctx := context.Background()
	s := NewFavoriteStore()
	at := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	if added, err := s.Add(ctx, alice, domain.Favorite{Kind: domain.FavoriteWaifu, Slug: "rem", Name: "Rem", AddedAt: at.Add(time.Minute)}); err != nil || !added {
		t.Fatalf("add = %v %v", added, err)
	}
	if added, _ := s.Add(ctx, alice, domain.Favorite{Kind: domain.FavoriteWaifu, Slug: "rem"}); added {
		t.Error("duplicate add should report false")
	}
	if _, err := s.Add(ctx, alice, domain.Favorite{Kind: domain.FavoriteWaifu, Slug: "ram", Name: "Ram", AddedAt: at}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(ctx, alice, domain.Favorite{Kind: domain.FavoriteSeries, Slug: "re-zero", Name: "Re:Zero", AddedAt: at}); err != nil {
		t.Fatal(err)
	}
	got, err := s.List(ctx, alice, domain.FavoriteWaifu)
	if err != nil || len(got) != 2 || got[0].Slug != "ram" || got[1].Slug != "rem" {
		t.Errorf("list should be in insertion order: %+v %v", got, err)
	}
	if removed, _ := s.Remove(ctx, alice, domain.FavoriteWaifu, "rem"); !removed {
		t.Error("remove should report true")
	}
	if removed, _ := s.Remove(ctx, alice, domain.FavoriteWaifu, "rem"); removed {
		t.Error("second remove should report false")
	}
	if got, _ := s.List(ctx, alice, domain.FavoriteSeries); len(got) != 1 || got[0].Name != "Re:Zero" {
		t.Errorf("series list = %+v", got)
	}
	if got, _ := s.List(ctx, bob, domain.FavoriteWaifu); len(got) != 0 {
		t.Errorf("other player = %+v", got)
	}
}

func TestFavoriteStore_Find(t *testing.T) {
	ctx := context.Background()
	s := NewFavoriteStore()
	other := domain.PlayerKey{GuildID: "h", UserID: "alice"}
	for _, k := range []domain.PlayerKey{alice, bob, other} {
		if _, err := s.Add(ctx, k, domain.Favorite{Kind: domain.FavoriteWaifu, Slug: "rem", Name: "Rem"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Add(ctx, alice, domain.Favorite{Kind: domain.FavoriteSeries, Slug: "rem", Name: "Series named rem"}); err != nil {
		t.Fatal(err)
	}
	all, err := s.Find(ctx, domain.FavoriteWaifu, []string{"rem", "ram"}, "")
	if err != nil || len(all) != 3 || all[0].Key != alice || all[1].Key != other || all[2].Key != bob {
		t.Errorf("find across guilds = %+v %v", all, err)
	}
	inGuild, _ := s.Find(ctx, domain.FavoriteWaifu, []string{"rem"}, "g")
	if len(inGuild) != 2 {
		t.Errorf("find in guild = %+v", inGuild)
	}
	if none, _ := s.Find(ctx, domain.FavoriteWaifu, []string{"nobody"}, ""); len(none) != 0 {
		t.Errorf("no match = %+v", none)
	}
}

func TestAlertStore(t *testing.T) {
	ctx := context.Background()
	s := NewAlertStore()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	if st, _ := s.Setting(ctx, alice); !st.Enabled || st.DMClosed {
		t.Errorf("default = %+v", st)
	}
	if err := s.SetEnabled(ctx, alice, false, now); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Setting(ctx, alice); st.Enabled {
		t.Error("should be disabled")
	}
	if err := s.SetEnabled(ctx, alice, true, now); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkDMClosed(ctx, "alice", now); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Setting(ctx, alice); !st.Enabled || !st.DMClosed {
		t.Errorf("after closing = %+v", st)
	}
	if err := s.ClearDMClosed(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.Setting(ctx, alice); st.DMClosed {
		t.Error("closed mark should clear")
	}
	if first, _ := s.MarkSent(ctx, "e1", "alice", now); !first {
		t.Error("first send")
	}
	if first, _ := s.MarkSent(ctx, "e1", "alice", now); first {
		t.Error("second send of the same event")
	}
	if first, _ := s.MarkSent(ctx, "e1", "bob", now); !first || s.SentCount() != 2 {
		t.Error("another user is a separate send")
	}
}

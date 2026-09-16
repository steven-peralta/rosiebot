package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestCacheStore_WaifuRoundTripTouchAndPrune(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock := &fixedClock{now}
	s := NewCacheStore(testPool, clock)

	if _, err := s.GetWaifu(ctx, "cache-miss"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("miss = %v", err)
	}
	weight := 45.5
	age := 17
	w := domain.Waifu{
		WaifuSummary: domain.WaifuSummary{Slug: "cache-rem", UUID: "u", Name: "Rem", OriginalName: "レム", PictureURL: "p", Likes: 5, Trash: 1},
		URL:          "https://www.mywaifulist.moe/waifu/cache-rem",
		Description:  "maid",
		NSFW:         true,
		Weight:       &weight,
		Age:          &age,
		Appearances:  []domain.Series{{Slug: "re-zero", Name: "Re:Zero"}},
	}
	if err := s.PutWaifu(ctx, w, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetWaifu(ctx, "cache-rem")
	if err != nil {
		t.Fatal(err)
	}
	if got.Waifu.Name != "Rem" || got.Waifu.OriginalName != "レム" || *got.Waifu.Weight != 45.5 || *got.Waifu.Age != 17 || !got.Waifu.NSFW || got.Waifu.Appearances[0].Slug != "re-zero" || !got.FetchedAt.Equal(now.Add(-time.Hour)) {
		t.Errorf("round trip = %+v", got)
	}

	var lastRead time.Time
	if err := testPool.QueryRow(ctx, "SELECT last_read_at FROM waifu_cache WHERE slug = 'cache-rem'").Scan(&lastRead); err != nil {
		t.Fatal(err)
	}
	if !lastRead.Equal(now.Add(-time.Hour)) {
		t.Errorf("a read within the touch interval must not rewrite last_read_at: %v", lastRead)
	}
	clock.now = now.Add(2 * time.Hour)
	if _, err := s.GetWaifu(ctx, "cache-rem"); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, "SELECT last_read_at FROM waifu_cache WHERE slug = 'cache-rem'").Scan(&lastRead); err != nil {
		t.Fatal(err)
	}
	if !lastRead.Equal(clock.now) {
		t.Errorf("read after the touch interval should update last_read_at: %v", lastRead)
	}

	w.Likes = 99
	if err := s.PutWaifu(ctx, w, clock.now); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetWaifu(ctx, "cache-rem"); got.Waifu.Likes != 99 || !got.FetchedAt.Equal(clock.now) {
		t.Errorf("upsert = %+v", got)
	}

	if n, err := s.Prune(ctx, clock.now.Add(-time.Minute)); err != nil || n != 0 {
		t.Errorf("prune of fresh entries = %d %v", n, err)
	}
	if n, err := s.Prune(ctx, clock.now.Add(time.Minute)); err != nil || n < 1 {
		t.Errorf("prune of stale entries = %d %v", n, err)
	}
	if _, err := s.GetWaifu(ctx, "cache-rem"); !errors.Is(err, app.ErrNotFound) {
		t.Error("pruned entry should be gone")
	}
}

func TestCacheStore_PageRoundTrip(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	s := NewCacheStore(testPool, &fixedClock{now})

	if _, err := s.GetPage(ctx, "search|nobody|1"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("miss = %v", err)
	}
	page := app.SearchPage{Page: 1, LastPage: 3, Items: []domain.WaifuSummary{{Slug: "a", Name: "A", Likes: 1}, {Slug: "b", Name: "B", Trash: 2}}}
	if err := s.PutPage(ctx, "search|shinji|1", page, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetPage(ctx, "search|shinji|1")
	if err != nil || got.Page.LastPage != 3 || len(got.Page.Items) != 2 || got.Page.Items[1].Trash != 2 || !got.FetchedAt.Equal(now) {
		t.Errorf("round trip = %+v %v", got, err)
	}
	if err := s.PutPage(ctx, "search|shinji|1", app.SearchPage{Page: 1, LastPage: 1}, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetPage(ctx, "search|shinji|1"); got.Page.LastPage != 1 {
		t.Errorf("upsert = %+v", got)
	}
	if n, err := s.Prune(ctx, now.Add(2*time.Hour)); err != nil || n < 1 {
		t.Errorf("prune pages = %d %v", n, err)
	}
}

func TestCacheStore_SeriesRoundTrip(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	s := NewCacheStore(testPool, &fixedClock{now})
	if _, err := s.GetSeries(ctx, "works|nothing|1"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("miss = %v", err)
	}
	series := []domain.Series{{Slug: "re-zero", Name: "Re:Zero", URL: "u"}, {Slug: "other", Name: "Other"}}
	if err := s.PutSeries(ctx, "works|re zero|1", series, 1, now); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSeries(ctx, "works|re zero|1")
	if err != nil || len(got.Series) != 2 || got.Series[0].URL != "u" || !got.FetchedAt.Equal(now) {
		t.Errorf("round trip = %+v %v", got, err)
	}
	if _, err := testPool.Exec(ctx, "INSERT INTO page_cache (key, payload, fetched_at) VALUES ('works|corrupt|1', '\"nope\"'::jsonb, now()) ON CONFLICT (key) DO NOTHING"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSeries(ctx, "works|corrupt|1"); err == nil {
		t.Error("corrupt series payload should fail to decode")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.GetSeries(cancelled, "x"); err == nil {
		t.Error("GetSeries on cancelled context")
	}
	if err := s.PutSeries(cancelled, "x", series, 1, now); err == nil {
		t.Error("PutSeries on cancelled context")
	}
}

func TestCacheStore_ErrorsOnCancelledContext(t *testing.T) {
	requireDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := NewCacheStore(testPool, nil)
	if _, err := s.GetWaifu(ctx, "x"); err == nil {
		t.Error("GetWaifu")
	}
	if err := s.PutWaifu(ctx, domain.Waifu{WaifuSummary: domain.WaifuSummary{Slug: "x"}}, time.Now()); err == nil {
		t.Error("PutWaifu")
	}
	if _, err := s.GetPage(ctx, "x"); err == nil {
		t.Error("GetPage")
	}
	if err := s.PutPage(ctx, "x", app.SearchPage{}, time.Now()); err == nil {
		t.Error("PutPage")
	}
	if _, err := s.Prune(ctx, time.Now()); err == nil {
		t.Error("Prune")
	}
}

func TestCacheStore_CorruptPayload(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	if _, err := testPool.Exec(ctx, "INSERT INTO waifu_cache (slug, payload, fetched_at) VALUES ('corrupt', '\"nope\"'::jsonb, now()) ON CONFLICT (slug) DO NOTHING"); err != nil {
		t.Fatal(err)
	}
	if _, err := testPool.Exec(ctx, "INSERT INTO page_cache (key, payload, fetched_at) VALUES ('corrupt', '\"nope\"'::jsonb, now()) ON CONFLICT (key) DO NOTHING"); err != nil {
		t.Fatal(err)
	}
	s := NewCacheStore(testPool, nil)
	if _, err := s.GetWaifu(ctx, "corrupt"); err == nil {
		t.Error("corrupt waifu payload should fail to decode")
	}
	if _, err := s.GetPage(ctx, "corrupt"); err == nil {
		t.Error("corrupt page payload should fail to decode")
	}
}

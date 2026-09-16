package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type tickingClock struct{ now time.Time }

func (c *tickingClock) Now() time.Time { return c.now }

func TestWaifuCache(t *testing.T) {
	ctx := context.Background()
	clock := &tickingClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)}
	c := NewWaifuCache(clock)

	if _, err := c.GetWaifu(ctx, "x"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("miss = %v", err)
	}
	if err := c.PutWaifu(ctx, domain.Waifu{WaifuSummary: domain.WaifuSummary{Slug: "x", Name: "X"}}, clock.now); err != nil {
		t.Fatal(err)
	}
	got, err := c.GetWaifu(ctx, "x")
	if err != nil || got.Waifu.Name != "X" || !got.FetchedAt.Equal(clock.now) {
		t.Errorf("hit = %+v %v", got, err)
	}

	if _, err := c.GetPage(ctx, "k"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("page miss = %v", err)
	}
	if err := c.PutPage(ctx, "k", app.SearchPage{Page: 1, LastPage: 2}, clock.now); err != nil {
		t.Fatal(err)
	}
	if page, err := c.GetPage(ctx, "k"); err != nil || page.Page.LastPage != 2 {
		t.Errorf("page hit = %+v %v", page, err)
	}
	if _, err := c.GetSeries(ctx, "s"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("series miss = %v", err)
	}
	if err := c.PutSeries(ctx, "s", []domain.Series{{Slug: "re-zero"}}, 1, clock.now); err != nil {
		t.Fatal(err)
	}
	if got, err := c.GetSeries(ctx, "s"); err != nil || len(got.Series) != 1 {
		t.Errorf("series hit = %+v %v", got, err)
	}
	if c.Len() != 3 {
		t.Errorf("len = %d", c.Len())
	}

	clock.now = clock.now.Add(48 * time.Hour)
	if _, err := c.GetWaifu(ctx, "x"); err != nil {
		t.Fatal(err)
	}
	n, err := c.Prune(ctx, clock.now.Add(-time.Hour))
	if err != nil || n != 2 || c.Len() != 1 {
		t.Errorf("prune should drop the unread page and series and keep the recently read waifu: n=%d len=%d err=%v", n, c.Len(), err)
	}
	if NewWaifuCache(nil).clock == nil {
		t.Error("nil clock should default")
	}
}

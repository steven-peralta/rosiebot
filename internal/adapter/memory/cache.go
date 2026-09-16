package memory

import (
	"context"
	"sync"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type WaifuCache struct {
	mu     sync.Mutex
	clock  app.Clock
	waifus map[string]cachedWaifu
	pages  map[string]app.CachedPage
	series map[string]app.CachedSeries
}

type cachedWaifu struct {
	app.CachedWaifu
	lastRead time.Time
}

var _ app.WaifuCache = (*WaifuCache)(nil)

func NewWaifuCache(clock app.Clock) *WaifuCache {
	if clock == nil {
		clock = app.SystemClock()
	}
	return &WaifuCache{clock: clock, waifus: map[string]cachedWaifu{}, pages: map[string]app.CachedPage{}, series: map[string]app.CachedSeries{}}
}

func (c *WaifuCache) GetWaifu(ctx context.Context, slug string) (app.CachedWaifu, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.waifus[slug]
	if !ok {
		return app.CachedWaifu{}, app.ErrNotFound
	}
	entry.lastRead = c.clock.Now()
	c.waifus[slug] = entry
	return entry.CachedWaifu, nil
}

func (c *WaifuCache) PutWaifu(ctx context.Context, w domain.Waifu, fetchedAt time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.waifus[w.Slug] = cachedWaifu{CachedWaifu: app.CachedWaifu{Waifu: w, FetchedAt: fetchedAt}, lastRead: c.clock.Now()}
	return nil
}

func (c *WaifuCache) GetPage(ctx context.Context, key string) (app.CachedPage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	page, ok := c.pages[key]
	if !ok {
		return app.CachedPage{}, app.ErrNotFound
	}
	return page, nil
}

func (c *WaifuCache) PutPage(ctx context.Context, key string, page app.SearchPage, fetchedAt time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pages[key] = app.CachedPage{Page: page, FetchedAt: fetchedAt}
	return nil
}

func (c *WaifuCache) GetSeries(ctx context.Context, key string) (app.CachedSeries, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.series[key]
	if !ok {
		return app.CachedSeries{}, app.ErrNotFound
	}
	return entry, nil
}

func (c *WaifuCache) PutSeries(ctx context.Context, key string, series []domain.Series, lastPage int, fetchedAt time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.series[key] = app.CachedSeries{Series: series, LastPage: lastPage, FetchedAt: fetchedAt}
	return nil
}

func (c *WaifuCache) Prune(ctx context.Context, unreadSince time.Time) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var n int64
	for slug, entry := range c.waifus {
		if entry.lastRead.Before(unreadSince) {
			delete(c.waifus, slug)
			n++
		}
	}
	for key, page := range c.pages {
		if page.FetchedAt.Before(unreadSince) {
			delete(c.pages, key)
			n++
		}
	}
	for key, entry := range c.series {
		if entry.FetchedAt.Before(unreadSince) {
			delete(c.series, key)
			n++
		}
	}
	return n, nil
}

func (c *WaifuCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.waifus) + len(c.pages) + len(c.series)
}

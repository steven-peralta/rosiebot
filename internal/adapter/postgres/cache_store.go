package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/steven-peralta/rosiebot/internal/adapter/postgres/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const touchInterval = time.Hour

type CacheStore struct {
	q     *gen.Queries
	clock app.Clock
}

var _ app.WaifuCache = (*CacheStore)(nil)

func NewCacheStore(pool *pgxpool.Pool, clock app.Clock) *CacheStore {
	if clock == nil {
		clock = app.SystemClock()
	}
	return &CacheStore{q: gen.New(pool), clock: clock}
}

func (s *CacheStore) GetWaifu(ctx context.Context, slug string) (app.CachedWaifu, error) {
	row, err := s.q.GetCachedWaifu(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.CachedWaifu{}, app.ErrNotFound
	}
	if err != nil {
		return app.CachedWaifu{}, fmt.Errorf("postgres: get cached waifu: %w", err)
	}
	var w domain.Waifu
	if err := json.Unmarshal(row.Payload, &w); err != nil {
		return app.CachedWaifu{}, fmt.Errorf("postgres: decode cached waifu %s: %w", slug, err)
	}
	now := s.clock.Now()
	if err := s.q.TouchCachedWaifu(ctx, gen.TouchCachedWaifuParams{Now: now, Slug: slug, StaleBefore: now.Add(-touchInterval)}); err != nil {
		return app.CachedWaifu{}, fmt.Errorf("postgres: touch cached waifu: %w", err)
	}
	return app.CachedWaifu{Waifu: w, FetchedAt: row.FetchedAt}, nil
}

func (s *CacheStore) PutWaifu(ctx context.Context, w domain.Waifu, fetchedAt time.Time) error {
	payload, err := json.Marshal(w)
	if err != nil {
		return fmt.Errorf("postgres: encode cached waifu: %w", err)
	}
	if err := s.q.PutCachedWaifu(ctx, gen.PutCachedWaifuParams{Slug: w.Slug, Payload: payload, FetchedAt: fetchedAt}); err != nil {
		return fmt.Errorf("postgres: put cached waifu: %w", err)
	}
	return nil
}

func (s *CacheStore) GetPage(ctx context.Context, key string) (app.CachedPage, error) {
	row, err := s.q.GetCachedPage(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.CachedPage{}, app.ErrNotFound
	}
	if err != nil {
		return app.CachedPage{}, fmt.Errorf("postgres: get cached page: %w", err)
	}
	var page app.SearchPage
	if err := json.Unmarshal(row.Payload, &page); err != nil {
		return app.CachedPage{}, fmt.Errorf("postgres: decode cached page %s: %w", key, err)
	}
	return app.CachedPage{Page: page, FetchedAt: row.FetchedAt}, nil
}

func (s *CacheStore) PutPage(ctx context.Context, key string, page app.SearchPage, fetchedAt time.Time) error {
	payload, err := json.Marshal(page)
	if err != nil {
		return fmt.Errorf("postgres: encode cached page: %w", err)
	}
	if err := s.q.PutCachedPage(ctx, gen.PutCachedPageParams{Key: key, Payload: payload, FetchedAt: fetchedAt}); err != nil {
		return fmt.Errorf("postgres: put cached page: %w", err)
	}
	return nil
}

func (s *CacheStore) GetSeries(ctx context.Context, key string) (app.CachedSeries, error) {
	row, err := s.q.GetCachedPage(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.CachedSeries{}, app.ErrNotFound
	}
	if err != nil {
		return app.CachedSeries{}, fmt.Errorf("postgres: get cached series: %w", err)
	}
	var entry seriesPayload
	if err := json.Unmarshal(row.Payload, &entry); err != nil {
		return app.CachedSeries{}, fmt.Errorf("postgres: decode cached series %s: %w", key, err)
	}
	return app.CachedSeries{Series: entry.Series, LastPage: entry.LastPage, FetchedAt: row.FetchedAt}, nil
}

type seriesPayload struct {
	Series   []domain.Series `json:"series"`
	LastPage int             `json:"last_page"`
}

func (s *CacheStore) PutSeries(ctx context.Context, key string, series []domain.Series, lastPage int, fetchedAt time.Time) error {
	payload, err := json.Marshal(seriesPayload{Series: series, LastPage: lastPage})
	if err != nil {
		return fmt.Errorf("postgres: encode cached series: %w", err)
	}
	if err := s.q.PutCachedPage(ctx, gen.PutCachedPageParams{Key: key, Payload: payload, FetchedAt: fetchedAt}); err != nil {
		return fmt.Errorf("postgres: put cached series: %w", err)
	}
	return nil
}

func (s *CacheStore) Prune(ctx context.Context, unreadSince time.Time) (int64, error) {
	waifus, err := s.q.PruneCachedWaifus(ctx, unreadSince)
	if err != nil {
		return 0, fmt.Errorf("postgres: prune cached waifus: %w", err)
	}
	pages, err := s.q.PruneCachedPages(ctx, unreadSince)
	if err != nil {
		return waifus, fmt.Errorf("postgres: prune cached pages: %w", err)
	}
	return waifus + pages, nil
}

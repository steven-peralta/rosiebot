package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	DefaultWaifuTTL      = 24 * time.Hour
	DefaultSearchTTL     = time.Hour
	DefaultCacheRetain   = 30 * 24 * time.Hour
	cacheRefreshTimeout  = 45 * time.Second
	cacheMaintenanceTick = 6 * time.Hour
)

type CachedWaifu struct {
	Waifu     domain.Waifu
	FetchedAt time.Time
}

type CachedPage struct {
	Page      SearchPage
	FetchedAt time.Time
}

type CachedSeries struct {
	Series    []domain.Series
	LastPage  int
	FetchedAt time.Time
}

type WaifuCache interface {
	GetWaifu(ctx context.Context, slug string) (CachedWaifu, error)
	PutWaifu(ctx context.Context, w domain.Waifu, fetchedAt time.Time) error
	GetPage(ctx context.Context, key string) (CachedPage, error)
	PutPage(ctx context.Context, key string, page SearchPage, fetchedAt time.Time) error
	GetSeries(ctx context.Context, key string) (CachedSeries, error)
	PutSeries(ctx context.Context, key string, series []domain.Series, lastPage int, fetchedAt time.Time) error
	Prune(ctx context.Context, unreadSince time.Time) (int64, error)
}

type CacheConfig struct {
	WaifuTTL  time.Duration
	SearchTTL time.Duration
	Retain    time.Duration
}

func (c CacheConfig) withDefaults() CacheConfig {
	if c.WaifuTTL <= 0 {
		c.WaifuTTL = DefaultWaifuTTL
	}
	if c.SearchTTL <= 0 {
		c.SearchTTL = DefaultSearchTTL
	}
	if c.Retain <= 0 {
		c.Retain = DefaultCacheRetain
	}
	return c
}

type CachedSource struct {
	next     WaifuSource
	cache    WaifuCache
	clock    Clock
	cfg      CacheConfig
	log      *slog.Logger
	mu       sync.Mutex
	inflight map[string]struct{}
	wg       sync.WaitGroup
}

var _ WaifuSource = (*CachedSource)(nil)

func NewCachedSource(next WaifuSource, cache WaifuCache, clock Clock, cfg CacheConfig, log *slog.Logger) *CachedSource {
	if log == nil {
		log = slog.Default()
	}
	return &CachedSource{next: next, cache: cache, clock: clock, cfg: cfg.withDefaults(), log: log, inflight: map[string]struct{}{}}
}

func (s *CachedSource) Random(ctx context.Context) (domain.WaifuSummary, error) {
	return s.next.Random(ctx)
}

func (s *CachedSource) Daily(ctx context.Context) (domain.WaifuSummary, error) {
	return s.next.Daily(ctx)
}

func (s *CachedSource) PopularPage(ctx context.Context, page int) (PopularPage, error) {
	return s.next.PopularPage(ctx, page)
}

func (s *CachedSource) Get(ctx context.Context, slug string) (domain.Waifu, error) {
	now := s.clock.Now()
	cached, err := s.cache.GetWaifu(ctx, slug)
	switch {
	case err == nil && now.Sub(cached.FetchedAt) < s.cfg.WaifuTTL:
		return cached.Waifu, nil
	case err == nil:
		s.refresh("waifu:"+slug, func(ctx context.Context) error {
			w, err := s.next.Get(ctx, slug)
			if err != nil {
				return err
			}
			return s.cache.PutWaifu(ctx, w, s.clock.Now())
		})
		return cached.Waifu, nil
	case !errors.Is(err, ErrNotFound):
		s.log.Warn("waifu cache read failed", "slug", slug, "err", err)
	}
	w, err := s.next.Get(ctx, slug)
	if err != nil {
		return domain.Waifu{}, err
	}
	if err := s.cache.PutWaifu(ctx, w, now); err != nil {
		s.log.Warn("waifu cache write failed", "slug", slug, "err", err)
	}
	return w, nil
}

func (s *CachedSource) SearchWaifus(ctx context.Context, term string, page int) (SearchPage, error) {
	return s.cachedPage(ctx, pageKey("search", term, page), func(ctx context.Context) (SearchPage, error) {
		return s.next.SearchWaifus(ctx, term, page)
	})
}

func (s *CachedSource) ListCharacters(ctx context.Context, page int) (SearchPage, error) {
	return s.cachedPage(ctx, pageKey("list", "", page), func(ctx context.Context) (SearchPage, error) {
		return s.next.ListCharacters(ctx, page)
	})
}

func (s *CachedSource) WorkCharacters(ctx context.Context, slug string, page int) (SearchPage, error) {
	return s.cachedPage(ctx, pageKey("work", slug, page), func(ctx context.Context) (SearchPage, error) {
		return s.next.WorkCharacters(ctx, slug, page)
	})
}

func (s *CachedSource) SearchWorks(ctx context.Context, term string) ([]domain.Series, error) {
	key := pageKey("works", term, 1)
	now := s.clock.Now()
	cached, err := s.cache.GetSeries(ctx, key)
	switch {
	case err == nil && now.Sub(cached.FetchedAt) < s.cfg.SearchTTL:
		return cached.Series, nil
	case err == nil:
		s.refresh("series:"+key, func(ctx context.Context) error {
			series, err := s.next.SearchWorks(ctx, term)
			if err != nil {
				return err
			}
			return s.cache.PutSeries(ctx, key, series, 1, s.clock.Now())
		})
		return cached.Series, nil
	case !errors.Is(err, ErrNotFound):
		s.log.Warn("series cache read failed", "key", key, "err", err)
	}
	series, err := s.next.SearchWorks(ctx, term)
	if err != nil {
		return nil, err
	}
	if err := s.cache.PutSeries(ctx, key, series, 1, now); err != nil {
		s.log.Warn("series cache write failed", "key", key, "err", err)
	}
	return series, nil
}

func (s *CachedSource) ListWorks(ctx context.Context, page int) (SeriesPage, error) {
	key := pageKey("worklist", "", page)
	now := s.clock.Now()
	cached, err := s.cache.GetSeries(ctx, key)
	switch {
	case err == nil && now.Sub(cached.FetchedAt) < s.cfg.SearchTTL:
		return SeriesPage{Items: cached.Series, Page: page, LastPage: cached.LastPage}, nil
	case err == nil:
		s.refresh("series:"+key, func(ctx context.Context) error {
			sp, err := s.next.ListWorks(ctx, page)
			if err != nil {
				return err
			}
			return s.cache.PutSeries(ctx, key, sp.Items, sp.LastPage, s.clock.Now())
		})
		return SeriesPage{Items: cached.Series, Page: page, LastPage: cached.LastPage}, nil
	case !errors.Is(err, ErrNotFound):
		s.log.Warn("series cache read failed", "key", key, "err", err)
	}
	sp, err := s.next.ListWorks(ctx, page)
	if err != nil {
		return SeriesPage{}, err
	}
	if err := s.cache.PutSeries(ctx, key, sp.Items, sp.LastPage, now); err != nil {
		s.log.Warn("series cache write failed", "key", key, "err", err)
	}
	return sp, nil
}

func (s *CachedSource) Work(ctx context.Context, slug string) (domain.Series, error) {
	return s.next.Work(ctx, slug)
}

func (s *CachedSource) cachedPage(ctx context.Context, key string, fetch func(context.Context) (SearchPage, error)) (SearchPage, error) {
	now := s.clock.Now()
	cached, err := s.cache.GetPage(ctx, key)
	switch {
	case err == nil && now.Sub(cached.FetchedAt) < s.cfg.SearchTTL:
		return cached.Page, nil
	case err == nil:
		s.refresh("page:"+key, func(ctx context.Context) error {
			page, err := fetch(ctx)
			if err != nil {
				return err
			}
			return s.cache.PutPage(ctx, key, page, s.clock.Now())
		})
		return cached.Page, nil
	case !errors.Is(err, ErrNotFound):
		s.log.Warn("page cache read failed", "key", key, "err", err)
	}
	page, err := fetch(ctx)
	if err != nil {
		return SearchPage{}, err
	}
	if err := s.cache.PutPage(ctx, key, page, now); err != nil {
		s.log.Warn("page cache write failed", "key", key, "err", err)
	}
	return page, nil
}

func (s *CachedSource) refresh(key string, fn func(context.Context) error) {
	s.mu.Lock()
	if _, busy := s.inflight[key]; busy {
		s.mu.Unlock()
		return
	}
	s.inflight[key] = struct{}{}
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.inflight, key)
			s.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(WithBackground(context.Background()), cacheRefreshTimeout)
		defer cancel()
		if err := fn(ctx); err != nil {
			s.log.Warn("cache refresh failed, keeping stale entry", "key", key, "err", err)
		}
	}()
}

func (s *CachedSource) Flush() {
	s.wg.Wait()
}

func (s *CachedSource) Prune(ctx context.Context) (int64, error) {
	n, err := s.cache.Prune(ctx, s.clock.Now().Add(-s.cfg.Retain))
	if err != nil {
		return 0, fmt.Errorf("prune cache: %w", err)
	}
	return n, nil
}

func (s *CachedSource) RunMaintenance(ctx context.Context) {
	ticker := time.NewTicker(cacheMaintenanceTick)
	defer ticker.Stop()
	for {
		if n, err := s.Prune(ctx); err != nil {
			s.log.Warn("cache maintenance failed", "err", err)
		} else if n > 0 {
			s.log.Info("cache pruned", "entries", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func pageKey(kind, subject string, page int) string {
	subject = strings.Join(strings.Fields(strings.ToLower(subject)), " ")
	return kind + "|" + subject + "|" + strconv.Itoa(max(page, 1))
}

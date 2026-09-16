package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

var ErrRefreshInProgress = errors.New("ranking refresh already in progress")

type RankingConfig struct {
	RefreshInterval time.Duration
	FailureRetry    time.Duration
	MinVotes        int
	MaxPages        int
	PageRetries     int
	BackoffBase     time.Duration
	BackoffMax      time.Duration
}

func DefaultRankingConfig() RankingConfig {
	return RankingConfig{
		RefreshInterval: 24 * time.Hour,
		FailureRetry:    time.Hour,
		MinVotes:        domain.DefaultMinVotes,
		MaxPages:        1500,
		PageRetries:     5,
		BackoffBase:     time.Second,
		BackoffMax:      time.Minute,
	}
}

func (c RankingConfig) withDefaults() RankingConfig {
	d := DefaultRankingConfig()
	if c.RefreshInterval <= 0 {
		c.RefreshInterval = d.RefreshInterval
	}
	if c.FailureRetry <= 0 {
		c.FailureRetry = d.FailureRetry
	}
	if c.MinVotes <= 0 {
		c.MinVotes = d.MinVotes
	}
	if c.MaxPages <= 0 {
		c.MaxPages = d.MaxPages
	}
	if c.PageRetries <= 0 {
		c.PageRetries = d.PageRetries
	}
	if c.BackoffBase <= 0 {
		c.BackoffBase = d.BackoffBase
	}
	if c.BackoffMax <= 0 {
		c.BackoffMax = d.BackoffMax
	}
	return c
}

type RankingService struct {
	store      RankingStore
	source     WaifuSource
	clock      Clock
	cfg        RankingConfig
	log        *slog.Logger
	sleep      func(context.Context, time.Duration) error
	current    atomic.Pointer[domain.Ranking]
	refreshing atomic.Bool
}

var _ RankingProvider = (*RankingService)(nil)

func NewRankingService(store RankingStore, source WaifuSource, clock Clock, cfg RankingConfig, log *slog.Logger) *RankingService {
	if log == nil {
		log = slog.Default()
	}
	return &RankingService{store: store, source: source, clock: clock, cfg: cfg.withDefaults(), log: log, sleep: sleepContext}
}

func (s *RankingService) Current() *domain.Ranking {
	return s.current.Load()
}

func (s *RankingService) NextRefresh() time.Time {
	if r := s.Current(); r != nil {
		return r.FetchedAt.Add(s.cfg.RefreshInterval)
	}
	return s.clock.Now()
}

func (s *RankingService) Load(ctx context.Context) error {
	r, err := s.store.LoadLatest(ctx)
	if errors.Is(err, ErrNotFound) {
		s.log.Info("no ranking snapshot yet")
		return nil
	}
	if err != nil {
		return fmt.Errorf("load ranking: %w", err)
	}
	s.current.Store(r)
	s.log.Info("ranking loaded", "rows", r.Len(), "fetched_at", r.FetchedAt, "cutoff_page", r.CutoffPage)
	return nil
}

func (s *RankingService) Refresh(ctx context.Context) error {
	if !s.refreshing.CompareAndSwap(false, true) {
		return ErrRefreshInProgress
	}
	defer s.refreshing.Store(false)

	started := s.clock.Now()
	rows, cutoff, err := s.walk(ctx)
	if err != nil {
		return err
	}
	ranking := domain.BuildRanking(rows, s.cfg.MinVotes, started, cutoff)
	if err := s.store.Save(ctx, ranking); err != nil {
		return fmt.Errorf("save ranking: %w", err)
	}
	s.current.Store(ranking)
	s.log.Info("ranking refreshed", "rows", ranking.Len(), "cutoff_page", cutoff, "took", s.clock.Now().Sub(started))
	return nil
}

func (s *RankingService) walk(ctx context.Context) ([]domain.WaifuSummary, int, error) {
	var rows []domain.WaifuSummary
	lastPage := 0
	for page := 1; ; page++ {
		if page > s.cfg.MaxPages {
			s.log.Warn("ranking walk hit the page cap", "max_pages", s.cfg.MaxPages)
			return rows, page - 1, nil
		}
		pp, err := s.fetchPage(ctx, page)
		if err != nil {
			return nil, 0, err
		}
		rows = append(rows, pp.Rows...)
		if pp.LastPage > 0 {
			lastPage = pp.LastPage
		}
		if len(pp.Rows) == 0 || minTotal(pp.Rows) <= s.cfg.MinVotes || (lastPage > 0 && page >= lastPage) {
			return rows, page, nil
		}
	}
}

func (s *RankingService) fetchPage(ctx context.Context, page int) (PopularPage, error) {
	var lastErr error
	for attempt := 1; attempt <= s.cfg.PageRetries; attempt++ {
		pp, err := s.source.PopularPage(ctx, page)
		if err == nil {
			return pp, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return PopularPage{}, ctx.Err()
		}
		if attempt == s.cfg.PageRetries {
			break
		}
		wait := min(s.cfg.BackoffBase<<(attempt-1), s.cfg.BackoffMax)
		s.log.Warn("ranking page fetch failed", "page", page, "attempt", attempt, "err", err, "retry_in", wait)
		if err := s.sleep(ctx, wait); err != nil {
			return PopularPage{}, err
		}
	}
	return PopularPage{}, fmt.Errorf("ranking page %d failed after %d attempts: %w", page, s.cfg.PageRetries, lastErr)
}

func (s *RankingService) Run(ctx context.Context) error {
	if err := s.Load(ctx); err != nil {
		s.log.Error("loading ranking snapshot failed", "err", err)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if wait := s.NextRefresh().Sub(s.clock.Now()); wait > 0 {
			if err := s.sleep(ctx, wait); err != nil {
				return err
			}
		}
		if err := s.Refresh(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			s.log.Error("ranking refresh failed, keeping previous table", "err", err, "retry_in", s.cfg.FailureRetry)
			if err := s.sleep(ctx, s.cfg.FailureRetry); err != nil {
				return err
			}
		}
	}
}

func minTotal(rows []domain.WaifuSummary) int {
	m := rows[0].TotalVotes()
	for _, r := range rows[1:] {
		m = min(m, r.TotalVotes())
	}
	return m
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

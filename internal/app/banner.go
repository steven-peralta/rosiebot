package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type BannerConfig struct {
	MaxAttempts   int
	MaxPages      int
	MinRanked     int
	MinStars      int
	RetryInterval time.Duration
	RankingWait   time.Duration
}

func DefaultBannerConfig() BannerConfig {
	return BannerConfig{
		MaxAttempts:   12,
		MaxPages:      30,
		MinRanked:     domain.BannerMinRanked,
		MinStars:      domain.BannerMinStars,
		RetryInterval: 10 * time.Minute,
		RankingWait:   30 * time.Second,
	}
}

func (c BannerConfig) withDefaults() BannerConfig {
	d := DefaultBannerConfig()
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = d.MaxAttempts
	}
	if c.MaxPages <= 0 {
		c.MaxPages = d.MaxPages
	}
	if c.MinRanked <= 0 {
		c.MinRanked = d.MinRanked
	}
	if c.MinStars <= 0 {
		c.MinStars = d.MinStars
	}
	if c.RetryInterval <= 0 {
		c.RetryInterval = d.RetryInterval
	}
	if c.RankingWait <= 0 {
		c.RankingWait = d.RankingWait
	}
	return c
}

type BannerResult struct {
	Banner    domain.Banner
	RefreshIn time.Duration
}

type BannerService struct {
	store   BannerStore
	ranking RankingProvider
	source  WaifuSource
	clock   Clock
	rng     Random
	loc     *time.Location
	cfg     BannerConfig
	log     *slog.Logger
	sleep   func(context.Context, time.Duration) error
}

func NewBannerService(store BannerStore, ranking RankingProvider, source WaifuSource, clock Clock, rng Random, loc *time.Location, cfg BannerConfig, log *slog.Logger) *BannerService {
	if loc == nil {
		loc = time.UTC
	}
	if log == nil {
		log = slog.Default()
	}
	return &BannerService{store: store, ranking: ranking, source: source, clock: clock, rng: rng, loc: loc, cfg: cfg.withDefaults(), log: log, sleep: sleepContext}
}

func (s *BannerService) Current(ctx context.Context) (BannerResult, error) {
	now := s.clock.Now()
	banner, err := s.store.Get(ctx, domain.BannerWeekStart(now, s.loc))
	if errors.Is(err, ErrNotFound) {
		return BannerResult{}, ErrNoBanner
	}
	if err != nil {
		return BannerResult{}, fmt.Errorf("load banner: %w", err)
	}
	return BannerResult{Banner: banner, RefreshIn: domain.BannerRefreshIn(now, s.loc)}, nil
}

func (s *BannerService) Ensure(ctx context.Context) (domain.Banner, error) {
	week := domain.BannerWeekStart(s.clock.Now(), s.loc)
	existing, err := s.store.Get(ctx, week)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return domain.Banner{}, fmt.Errorf("load banner: %w", err)
	}
	picked, err := s.pick(ctx, week, s.previousSeries(ctx, week))
	if err != nil {
		return domain.Banner{}, err
	}
	stored, err := s.store.Put(ctx, picked)
	if err != nil {
		return domain.Banner{}, fmt.Errorf("store banner: %w", err)
	}
	s.log.Info("banner selected", "week", week, "series", stored.Series.Slug, "characters", len(stored.Characters))
	return stored, nil
}

func (s *BannerService) Reroll(ctx context.Context) (domain.Banner, error) {
	week := domain.BannerWeekStart(s.clock.Now(), s.loc)
	exclude := s.previousSeries(ctx, week)
	if current, err := s.store.Get(ctx, week); err == nil {
		exclude[current.Series.Slug] = struct{}{}
	} else if !errors.Is(err, ErrNotFound) {
		return domain.Banner{}, fmt.Errorf("load banner: %w", err)
	}
	picked, err := s.pick(ctx, week, exclude)
	if err != nil {
		return domain.Banner{}, err
	}
	if err := s.store.Replace(ctx, picked); err != nil {
		return domain.Banner{}, fmt.Errorf("replace banner: %w", err)
	}
	s.log.Info("banner rerolled", "week", week, "series", picked.Series.Slug, "characters", len(picked.Characters))
	return picked, nil
}

func (s *BannerService) previousSeries(ctx context.Context, week time.Time) map[string]struct{} {
	exclude := map[string]struct{}{}
	if prev, err := s.store.Get(ctx, week.AddDate(0, 0, -7)); err == nil && prev.Series.Slug != "" {
		exclude[prev.Series.Slug] = struct{}{}
	}
	return exclude
}

func (s *BannerService) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := s.Ensure(WithBackground(ctx)); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			retry := s.cfg.RetryInterval
			if errors.Is(err, ErrNoRanking) {
				retry = s.cfg.RankingWait
			}
			wait := min(retry, domain.BannerRefreshIn(s.clock.Now(), s.loc))
			s.log.Warn("banner refresh failed", "err", err, "retry_in", wait)
			if err := s.sleep(ctx, wait); err != nil {
				return err
			}
			continue
		}
		if err := s.sleep(ctx, domain.BannerRefreshIn(s.clock.Now(), s.loc)); err != nil {
			return err
		}
	}
}

func (s *BannerService) pick(ctx context.Context, week time.Time, exclude map[string]struct{}) (domain.Banner, error) {
	ranking := s.ranking.Current()
	if ranking.Len() == 0 {
		return domain.Banner{}, ErrNoRanking
	}

	tried := map[string]struct{}{}
	for attempt := 1; attempt <= s.cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return domain.Banner{}, err
		}
		seed, err := ranking.Sample(s.rng, func(r domain.RankedWaifu) bool { return r.Stars >= s.cfg.MinStars })
		if err != nil {
			if seed, err = ranking.Random(s.rng); err != nil {
				return domain.Banner{}, ErrNoRanking
			}
		}
		detail, err := s.source.Get(ctx, seed.Slug)
		if err != nil {
			s.log.Warn("banner seed lookup failed", "slug", seed.Slug, "attempt", attempt, "err", err)
			continue
		}
		series, ok := detail.FirstSeries()
		if !ok || series.Slug == "" {
			continue
		}
		if _, skip := exclude[series.Slug]; skip {
			continue
		}
		if _, dup := tried[series.Slug]; dup {
			continue
		}
		tried[series.Slug] = struct{}{}

		members, err := s.members(ctx, series.Slug)
		if err != nil {
			s.log.Warn("banner series lookup failed", "series", series.Slug, "attempt", attempt, "err", err)
			continue
		}
		chars := ranking.Subset(members)
		if !domain.BannerEligible(chars, s.cfg.MinRanked, s.cfg.MinStars) {
			s.log.Debug("banner candidate not eligible", "series", series.Slug, "ranked", len(chars))
			continue
		}
		if series.PictureURL == "" {
			if full, err := s.source.Work(ctx, series.Slug); err == nil {
				series = full
			}
		}
		return domain.NewBanner(week, series, chars), nil
	}
	return domain.Banner{}, fmt.Errorf("%w after %d attempts", ErrNoEligibleSeries, s.cfg.MaxAttempts)
}

func (s *BannerService) members(ctx context.Context, slug string) ([]string, error) {
	var slugs []string
	for page := 1; page <= s.cfg.MaxPages; page++ {
		res, err := s.source.WorkCharacters(ctx, slug, page)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", page, err)
		}
		for _, it := range res.Items {
			slugs = append(slugs, it.Slug)
		}
		if page >= res.LastPage || len(res.Items) == 0 {
			break
		}
	}
	return slugs, nil
}

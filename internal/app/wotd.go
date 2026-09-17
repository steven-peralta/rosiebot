package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type WotdResult struct {
	Waifu     domain.WaifuSummary
	Day       time.Time
	RefreshIn time.Duration
}

type WotdService struct {
	store   DailyStore
	ranking RankingProvider
	clock   Clock
	rng     Random
	loc     *time.Location
	log     *slog.Logger
	picked  func(day time.Time, w domain.WaifuSummary)
}

func (s *WotdService) OnPicked(fn func(day time.Time, w domain.WaifuSummary)) {
	s.picked = fn
}

func (s *WotdService) notifyPicked(day time.Time, w domain.WaifuSummary) {
	if s.picked != nil {
		s.picked(day, w)
	}
}

func NewWotdService(store DailyStore, ranking RankingProvider, clock Clock, rng Random, loc *time.Location, log *slog.Logger) *WotdService {
	if loc == nil {
		loc = time.UTC
	}
	if log == nil {
		log = slog.Default()
	}
	return &WotdService{store: store, ranking: ranking, clock: clock, rng: rng, loc: loc, log: log}
}

func (s *WotdService) Today(ctx context.Context) (WotdResult, error) {
	now := s.clock.Now()
	day := domain.WotdDay(now, s.loc)
	result := WotdResult{Day: day, RefreshIn: domain.WotdRefreshIn(now, s.loc)}

	existing, err := s.store.Get(ctx, day)
	switch {
	case err == nil:
		ranking := s.ranking.Current()
		if ranking.Len() == 0 {
			result.Waifu = existing
			return result, nil
		}
		if _, ok := ranking.Lookup(existing.Slug); ok {
			result.Waifu = existing
			return result, nil
		}
		pick, err := s.pick()
		if err != nil {
			return WotdResult{}, err
		}
		if err := s.store.Replace(ctx, day, pick); err != nil {
			return WotdResult{}, fmt.Errorf("replace waifu of the day: %w", err)
		}
		s.log.Warn("replaced unranked waifu of the day", "day", day, "was", existing.Slug, "now", pick.Slug)
		s.notifyPicked(day, pick)
		result.Waifu = pick
		return result, nil
	case !errors.Is(err, ErrNotFound):
		return WotdResult{}, fmt.Errorf("load waifu of the day: %w", err)
	}

	pick, err := s.pick()
	if err != nil {
		return WotdResult{}, err
	}
	stored, err := s.store.Put(ctx, day, pick)
	if err != nil {
		return WotdResult{}, fmt.Errorf("store waifu of the day: %w", err)
	}
	if stored.Slug == pick.Slug {
		s.notifyPicked(day, stored)
	}
	result.Waifu = stored
	return result, nil
}

func (s *WotdService) pick() (domain.WaifuSummary, error) {
	ranking := s.ranking.Current()
	if ranking.Len() == 0 {
		return domain.WaifuSummary{}, ErrNoRanking
	}
	row, err := ranking.Sample(s.rng, domain.WotdEligible)
	if err != nil {
		return domain.WaifuSummary{}, ErrNoRanking
	}
	return row.WaifuSummary, nil
}

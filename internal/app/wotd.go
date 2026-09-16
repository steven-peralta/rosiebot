package app

import (
	"context"
	"errors"
	"fmt"
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
	source  WaifuSource
	clock   Clock
	rng     Random
	loc     *time.Location
}

func NewWotdService(store DailyStore, ranking RankingProvider, source WaifuSource, clock Clock, rng Random, loc *time.Location) *WotdService {
	if loc == nil {
		loc = time.UTC
	}
	return &WotdService{store: store, ranking: ranking, source: source, clock: clock, rng: rng, loc: loc}
}

func (s *WotdService) Today(ctx context.Context) (WotdResult, error) {
	now := s.clock.Now()
	day := domain.WotdDay(now, s.loc)
	result := WotdResult{Day: day, RefreshIn: domain.WotdRefreshIn(now, s.loc)}

	existing, err := s.store.Get(ctx, day)
	if err == nil {
		result.Waifu = existing
		return result, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return WotdResult{}, fmt.Errorf("load waifu of the day: %w", err)
	}

	pick, err := s.pick(ctx)
	if err != nil {
		return WotdResult{}, err
	}
	stored, err := s.store.Put(ctx, day, pick)
	if err != nil {
		return WotdResult{}, fmt.Errorf("store waifu of the day: %w", err)
	}
	result.Waifu = stored
	return result, nil
}

func (s *WotdService) pick(ctx context.Context) (domain.WaifuSummary, error) {
	if ranking := s.ranking.Current(); ranking.Len() > 0 {
		if row, err := ranking.Sample(s.rng, domain.WotdEligible); err == nil {
			return row.WaifuSummary, nil
		}
	}
	daily, err := s.source.Daily(ctx)
	if err != nil {
		return domain.WaifuSummary{}, fmt.Errorf("fallback to MWL daily: %w", err)
	}
	return daily, nil
}

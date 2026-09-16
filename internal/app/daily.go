package app

import (
	"context"
	"fmt"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type DailyResult struct {
	Coins      int64
	Multiplier int
	Balance    int64
}

func (r DailyResult) Critical() bool { return r.Multiplier > 1 }

type DailyService struct {
	players PlayerStore
	clock   Clock
	rng     Random
	loc     *time.Location
}

func NewDailyService(players PlayerStore, clock Clock, rng Random, loc *time.Location) *DailyService {
	if loc == nil {
		loc = time.UTC
	}
	return &DailyService{players: players, clock: clock, rng: rng, loc: loc}
}

func (s *DailyService) Claim(ctx context.Context, key domain.PlayerKey) (DailyResult, error) {
	player, err := s.players.EnsurePlayer(ctx, key)
	if err != nil {
		return DailyResult{}, fmt.Errorf("ensure player: %w", err)
	}
	now := s.clock.Now()
	windowStart := domain.DailyWindowStart(now, s.loc)
	if !domain.DailyClaimAllowed(player.DailyClaimedAt, now, s.loc) {
		return DailyResult{}, &DailyAlreadyClaimedError{ClaimedAt: *player.DailyClaimedAt, RefreshIn: domain.DailyRefreshIn(now, s.loc)}
	}

	multiplier, err := domain.DailyMultiplier(Rolled(s.rng))
	if err != nil {
		return DailyResult{}, err
	}
	amount := int64(domain.DailyCoins * multiplier)
	balance, ok, err := s.players.ClaimDaily(ctx, key, amount, windowStart, now)
	if err != nil {
		return DailyResult{}, fmt.Errorf("claim daily: %w", err)
	}
	if !ok {
		return DailyResult{}, &DailyAlreadyClaimedError{ClaimedAt: now, RefreshIn: domain.DailyRefreshIn(now, s.loc)}
	}
	return DailyResult{Coins: amount, Multiplier: multiplier, Balance: balance}, nil
}

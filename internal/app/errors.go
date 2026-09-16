package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrNotOwned          = errors.New("waifu not owned")
	ErrAlreadyOwned      = errors.New("waifu already owned")
	ErrRollExhausted     = errors.New("could not find a waifu you do not already own")
	ErrTradeConflict     = errors.New("trade no longer valid")
	ErrNoRanking         = errors.New("ranking not available yet")
	ErrRateLimited       = errors.New("upstream rate limited")
	ErrInsufficientCoins = domain.ErrInsufficientCoins
)

type DailyAlreadyClaimedError struct {
	ClaimedAt time.Time
	RefreshIn time.Duration
}

func (e *DailyAlreadyClaimedError) Error() string {
	return fmt.Sprintf("daily already claimed at %s, next in %s", e.ClaimedAt.Format(time.RFC3339), domain.FormatCountdown(e.RefreshIn))
}

func Rolled(rng Random) int {
	return rng.IntN(domain.D100Max-domain.D100Min+1) + domain.D100Min
}

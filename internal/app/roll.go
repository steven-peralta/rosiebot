package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const FallbackCutoffPage = 1000

type RollResult struct {
	Kind     domain.RollKind
	Waifu    domain.Waifu
	Ranked   *domain.RankedWaifu
	Balance  int64
	Attempts int
}

type RollService struct {
	players  PlayerStore
	source   WaifuSource
	ranking  RankingProvider
	wotd     *WotdService
	clock    Clock
	rng      Random
	minVotes int
	log      *slog.Logger
}

func NewRollService(players PlayerStore, source WaifuSource, ranking RankingProvider, wotd *WotdService, clock Clock, rng Random, minVotes int, log *slog.Logger) *RollService {
	if log == nil {
		log = slog.Default()
	}
	if minVotes <= 0 {
		minVotes = domain.DefaultMinVotes
	}
	return &RollService{players: players, source: source, ranking: ranking, wotd: wotd, clock: clock, rng: rng, minVotes: minVotes, log: log}
}

func (s *RollService) Roll(ctx context.Context, key domain.PlayerKey) (RollResult, error) {
	player, err := s.players.EnsurePlayer(ctx, key)
	if err != nil {
		return RollResult{}, fmt.Errorf("ensure player: %w", err)
	}
	if player.Coins < domain.RollCost {
		return RollResult{}, ErrInsufficientCoins
	}

	for attempt := 1; attempt <= domain.MaxRerollAttempts; attempt++ {
		kind, err := domain.RollKindFor(Rolled(s.rng))
		if err != nil {
			return RollResult{}, err
		}
		summary, err := s.pick(ctx, kind)
		if err != nil {
			return RollResult{}, fmt.Errorf("pick %s waifu: %w", kind, err)
		}

		owned, err := s.players.OwnedSlugs(ctx, key, []string{summary.Slug})
		if err != nil {
			return RollResult{}, fmt.Errorf("check ownership: %w", err)
		}
		if len(owned) > 0 {
			s.log.Debug("reroll: already owned", "slug", summary.Slug, "attempt", attempt)
			continue
		}

		detail, err := s.source.Get(ctx, summary.Slug)
		if err != nil {
			return RollResult{}, fmt.Errorf("fetch %s: %w", summary.Slug, err)
		}

		var balance int64
		err = s.players.WithinTx(ctx, func(r PlayerRepo) error {
			bal, ok, err := r.DebitCoins(ctx, key, domain.RollCost)
			if err != nil {
				return err
			}
			if !ok {
				return ErrInsufficientCoins
			}
			inserted, err := r.AddOwned(ctx, key, domain.OwnedFromSummary(summary, s.clock.Now()))
			if err != nil {
				return err
			}
			if !inserted {
				return ErrAlreadyOwned
			}
			balance = bal
			return nil
		})
		if errors.Is(err, ErrAlreadyOwned) {
			s.log.Debug("reroll: lost race", "slug", summary.Slug, "attempt", attempt)
			continue
		}
		if err != nil {
			return RollResult{}, err
		}

		result := RollResult{Kind: kind, Waifu: detail, Balance: balance, Attempts: attempt}
		if ranked, ok := s.ranking.Current().Lookup(summary.Slug); ok {
			result.Ranked = &ranked
		}
		return result, nil
	}
	return RollResult{}, ErrRollExhausted
}

func (s *RollService) pick(ctx context.Context, kind domain.RollKind) (domain.WaifuSummary, error) {
	switch kind {
	case domain.RollWaifuOfTheDay:
		today, err := s.wotd.Today(ctx)
		if err != nil {
			return domain.WaifuSummary{}, err
		}
		return today.Waifu, nil
	case domain.RollCritical:
		return s.pickRanked(ctx)
	default:
		return s.source.Random(ctx)
	}
}

func (s *RollService) pickRanked(ctx context.Context) (domain.WaifuSummary, error) {
	if ranking := s.ranking.Current(); ranking.Len() > 0 {
		if row, err := ranking.Random(s.rng); err == nil {
			return row.WaifuSummary, nil
		}
	}

	cutoff := FallbackCutoffPage
	page := s.rng.IntN(cutoff) + 1
	popular, err := s.source.PopularPage(ctx, page)
	if err != nil {
		return domain.WaifuSummary{}, err
	}
	candidates := make([]domain.WaifuSummary, 0, len(popular.Rows))
	for _, row := range popular.Rows {
		if row.TotalVotes() > s.minVotes {
			candidates = append(candidates, row)
		}
	}
	if len(candidates) == 0 {
		s.log.Warn("critical roll fallback found no ranked rows, degrading to regular roll", "page", page)
		return s.source.Random(ctx)
	}
	return candidates[s.rng.IntN(len(candidates))], nil
}

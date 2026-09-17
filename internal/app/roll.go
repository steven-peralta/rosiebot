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
	Banner   bool
}

type rollPlan struct {
	cost   int64
	banner bool
	kindFn func(int) (domain.RollKind, error)
	pickFn func(context.Context, domain.RollKind) (domain.WaifuSummary, domain.RollKind, error)
	dropFn func(slug string)
}

type RollService struct {
	players  PlayerStore
	source   WaifuSource
	ranking  RankingProvider
	wotd     *WotdService
	banner   *BannerService
	clock    Clock
	rng      Random
	minVotes int
	log      *slog.Logger
	rolled   func(key domain.PlayerKey, w domain.WaifuSummary)
}

func (s *RollService) OnRolled(fn func(key domain.PlayerKey, w domain.WaifuSummary)) {
	s.rolled = fn
}

func NewRollService(players PlayerStore, source WaifuSource, ranking RankingProvider, wotd *WotdService, banner *BannerService, clock Clock, rng Random, minVotes int, log *slog.Logger) *RollService {
	if log == nil {
		log = slog.Default()
	}
	if minVotes <= 0 {
		minVotes = domain.DefaultMinVotes
	}
	return &RollService{players: players, source: source, ranking: ranking, wotd: wotd, banner: banner, clock: clock, rng: rng, minVotes: minVotes, log: log}
}

func (s *RollService) Roll(ctx context.Context, key domain.PlayerKey) (RollResult, error) {
	return s.roll(ctx, key, rollPlan{
		cost:   domain.RollCost,
		kindFn: domain.RollKindFor,
		pickFn: func(ctx context.Context, kind domain.RollKind) (domain.WaifuSummary, domain.RollKind, error) {
			w, err := s.pick(ctx, kind)
			return w, kind, err
		},
	})
}

func (s *RollService) RollBanner(ctx context.Context, key domain.PlayerKey) (RollResult, error) {
	if s.banner == nil {
		return RollResult{}, ErrNoBanner
	}
	current, err := s.banner.Current(ctx)
	if err != nil {
		return RollResult{}, err
	}
	owned, err := s.players.OwnedSlugs(ctx, key, current.Banner.Slugs())
	if err != nil {
		return RollResult{}, fmt.Errorf("check banner ownership: %w", err)
	}
	pool := current.Banner.Unowned(owned)
	return s.roll(ctx, key, rollPlan{
		cost:   domain.BannerRollCost,
		banner: true,
		kindFn: domain.BannerRollKindFor,
		dropFn: func(slug string) {
			kept := pool[:0]
			for _, c := range pool {
				if c.Slug != slug {
					kept = append(kept, c)
				}
			}
			pool = kept
		},
		pickFn: func(ctx context.Context, kind domain.RollKind) (domain.WaifuSummary, domain.RollKind, error) {
			if kind != domain.RollBanner {
				w, err := s.pick(ctx, kind)
				return w, kind, err
			}
			if len(pool) == 0 {
				s.log.Debug("banner pool exhausted, degrading to critical", "series", current.Banner.Series.Slug, "user", key.UserID)
				w, err := s.pickRanked(ctx)
				return w, domain.RollCritical, err
			}
			return pool[s.rng.IntN(len(pool))].WaifuSummary, domain.RollBanner, nil
		},
	})
}

func (s *RollService) roll(ctx context.Context, key domain.PlayerKey, plan rollPlan) (RollResult, error) {
	player, err := s.players.EnsurePlayer(ctx, key)
	if err != nil {
		return RollResult{}, fmt.Errorf("ensure player: %w", err)
	}
	if player.Coins < plan.cost {
		return RollResult{}, ErrInsufficientCoins
	}

	for attempt := 1; attempt <= domain.MaxRerollAttempts; attempt++ {
		kind, err := plan.kindFn(Rolled(s.rng))
		if err != nil {
			return RollResult{}, err
		}
		summary, kind, err := plan.pickFn(ctx, kind)
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
		if errors.Is(err, ErrNotFound) {
			s.log.Warn("reroll: picked waifu no longer exists upstream", "slug", summary.Slug, "kind", kind, "attempt", attempt)
			if plan.dropFn != nil {
				plan.dropFn(summary.Slug)
			}
			continue
		}
		if err != nil {
			return RollResult{}, fmt.Errorf("fetch %s: %w", summary.Slug, err)
		}

		var balance int64
		err = s.players.WithinTx(ctx, func(r PlayerRepo) error {
			bal, ok, err := r.DebitCoins(ctx, key, plan.cost)
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

		if s.rolled != nil {
			s.rolled(key, summary)
		}
		result := RollResult{Kind: kind, Waifu: detail, Balance: balance, Attempts: attempt, Banner: plan.banner}
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
		if errors.Is(err, ErrNoRanking) {
			s.log.Warn("no waifu of the day yet, degrading to a critical roll")
			return s.pickRanked(ctx)
		}
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

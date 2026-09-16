package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type TradeProposal struct {
	Offer   domain.TradeOffer
	Give    []domain.OwnedWaifu
	Receive []domain.OwnedWaifu
}

type TradeService struct {
	players PlayerStore
}

func NewTradeService(players PlayerStore) *TradeService {
	return &TradeService{players: players}
}

func (s *TradeService) Propose(ctx context.Context, offer domain.TradeOffer) (TradeProposal, error) {
	offer, err := domain.NormaliseTrade(offer)
	if err != nil {
		return TradeProposal{}, err
	}
	if _, err := s.players.EnsurePlayer(ctx, offer.Sender); err != nil {
		return TradeProposal{}, fmt.Errorf("ensure sender: %w", err)
	}
	if _, err := s.players.EnsurePlayer(ctx, offer.Target); err != nil {
		return TradeProposal{}, fmt.Errorf("ensure target: %w", err)
	}
	if err := s.validate(ctx, s.players, offer); err != nil {
		return TradeProposal{}, err
	}
	give, err := s.lookup(ctx, offer.Sender, offer.Give)
	if err != nil {
		return TradeProposal{}, err
	}
	receive, err := s.lookup(ctx, offer.Target, offer.Receive)
	if err != nil {
		return TradeProposal{}, err
	}
	return TradeProposal{Offer: offer, Give: give, Receive: receive}, nil
}

func (s *TradeService) Accept(ctx context.Context, offer domain.TradeOffer) error {
	offer, err := domain.NormaliseTrade(offer)
	if err != nil {
		return err
	}
	return s.players.WithinTx(ctx, func(r PlayerRepo) error {
		if _, err := r.LockPlayers(ctx, offer.Sender, offer.Target); err != nil {
			return fmt.Errorf("lock players: %w", err)
		}
		if err := s.validate(ctx, r, offer); err != nil {
			var violation *domain.TradeViolation
			if errors.As(err, &violation) {
				return fmt.Errorf("%w: %w", ErrTradeConflict, err)
			}
			return err
		}
		if len(offer.Give) > 0 {
			moved, err := r.TransferOwned(ctx, offer.Sender, offer.Target, offer.Give)
			if err != nil {
				return fmt.Errorf("transfer to target: %w", err)
			}
			if moved != int64(len(offer.Give)) {
				return fmt.Errorf("%w: moved %d of %d offered", ErrTradeConflict, moved, len(offer.Give))
			}
		}
		if len(offer.Receive) > 0 {
			moved, err := r.TransferOwned(ctx, offer.Target, offer.Sender, offer.Receive)
			if err != nil {
				return fmt.Errorf("transfer to sender: %w", err)
			}
			if moved != int64(len(offer.Receive)) {
				return fmt.Errorf("%w: moved %d of %d requested", ErrTradeConflict, moved, len(offer.Receive))
			}
		}
		return nil
	})
}

func (s *TradeService) validate(ctx context.Context, r PlayerRepo, offer domain.TradeOffer) error {
	all := append(append([]string{}, offer.Give...), offer.Receive...)
	senderOwns, err := r.OwnedSlugs(ctx, offer.Sender, all)
	if err != nil {
		return fmt.Errorf("sender inventory: %w", err)
	}
	targetOwns, err := r.OwnedSlugs(ctx, offer.Target, all)
	if err != nil {
		return fmt.Errorf("target inventory: %w", err)
	}
	return domain.ValidateTrade(offer, senderOwns, targetOwns)
}

func (s *TradeService) lookup(ctx context.Context, key domain.PlayerKey, slugs []string) ([]domain.OwnedWaifu, error) {
	out := make([]domain.OwnedWaifu, 0, len(slugs))
	for _, slug := range slugs {
		w, err := s.players.GetOwned(ctx, key, slug)
		if err != nil {
			return nil, fmt.Errorf("lookup %s: %w", slug, err)
		}
		out = append(out, w)
	}
	return out, nil
}

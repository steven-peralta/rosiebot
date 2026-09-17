package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type SellResult struct {
	Waifu   domain.OwnedWaifu
	Price   int64
	Stars   int
	Balance int64
}

type InventoryService struct {
	players PlayerStore
	ranking RankingProvider
}

func NewInventoryService(players PlayerStore, ranking RankingProvider) *InventoryService {
	return &InventoryService{players: players, ranking: ranking}
}

func (s *InventoryService) Price(slug string) (int64, int) {
	stars := 0
	if s.ranking != nil {
		if r, ok := s.ranking.Current().Lookup(slug); ok {
			stars = r.Stars
		}
	}
	return domain.SellPriceFor(stars), stars
}

func (s *InventoryService) List(ctx context.Context, key domain.PlayerKey) ([]domain.OwnedWaifu, error) {
	if _, err := s.players.EnsurePlayer(ctx, key); err != nil {
		return nil, fmt.Errorf("ensure player: %w", err)
	}
	owned, err := s.players.ListOwned(ctx, key, "", 0)
	if err != nil {
		return nil, fmt.Errorf("list owned: %w", err)
	}
	return owned, nil
}

func (s *InventoryService) Suggest(ctx context.Context, key domain.PlayerKey, prefix string, limit int) ([]domain.OwnedWaifu, error) {
	owned, err := s.players.ListOwned(ctx, key, prefix, limit)
	if err != nil {
		return nil, fmt.Errorf("list owned: %w", err)
	}
	return owned, nil
}

func (s *InventoryService) Owns(ctx context.Context, key domain.PlayerKey, slug string) (domain.OwnedWaifu, bool, error) {
	owned, err := s.players.GetOwned(ctx, key, slug)
	if errors.Is(err, ErrNotFound) {
		return domain.OwnedWaifu{}, false, nil
	}
	if err != nil {
		return domain.OwnedWaifu{}, false, fmt.Errorf("get owned: %w", err)
	}
	return owned, true, nil
}

func (s *InventoryService) Sell(ctx context.Context, key domain.PlayerKey, slug string) (SellResult, error) {
	owned, err := s.players.GetOwned(ctx, key, slug)
	if errors.Is(err, ErrNotFound) {
		return SellResult{}, ErrNotOwned
	}
	if err != nil {
		return SellResult{}, fmt.Errorf("get owned: %w", err)
	}
	price, stars := s.Price(slug)
	balance, ok, err := s.players.SellOwned(ctx, key, slug, price)
	if err != nil {
		return SellResult{}, fmt.Errorf("sell: %w", err)
	}
	if !ok {
		return SellResult{}, ErrNotOwned
	}
	return SellResult{Waifu: owned, Price: price, Stars: stars, Balance: balance}, nil
}

type BulkQuote struct {
	Count int
	Total int64
	Tiers [domain.MaxStars + 1]int
}

type BulkSellResult struct {
	Count   int
	Total   int64
	Balance int64
}

func (s *InventoryService) eligible(owned []domain.OwnedWaifu, maxStars int) []domain.OwnedWaifu {
	out := make([]domain.OwnedWaifu, 0, len(owned))
	for _, w := range owned {
		if _, stars := s.Price(w.Slug); stars <= maxStars {
			out = append(out, w)
		}
	}
	return out
}

func (s *InventoryService) QuoteBelow(ctx context.Context, key domain.PlayerKey, maxStars int) (BulkQuote, error) {
	owned, err := s.List(ctx, key)
	if err != nil {
		return BulkQuote{}, err
	}
	var q BulkQuote
	for _, w := range s.eligible(owned, maxStars) {
		price, stars := s.Price(w.Slug)
		q.Count++
		q.Total += price
		q.Tiers[stars]++
	}
	return q, nil
}

func (s *InventoryService) SellBelow(ctx context.Context, key domain.PlayerKey, maxStars int) (BulkSellResult, error) {
	player, err := s.players.EnsurePlayer(ctx, key)
	if err != nil {
		return BulkSellResult{}, fmt.Errorf("ensure player: %w", err)
	}
	res := BulkSellResult{Balance: player.Coins}
	err = s.players.WithinTx(ctx, func(r PlayerRepo) error {
		owned, err := r.ListOwned(ctx, key, "", 0)
		if err != nil {
			return fmt.Errorf("list owned: %w", err)
		}
		for _, w := range s.eligible(owned, maxStars) {
			price, _ := s.Price(w.Slug)
			balance, ok, err := r.SellOwned(ctx, key, w.Slug, price)
			if err != nil {
				return fmt.Errorf("sell %s: %w", w.Slug, err)
			}
			if !ok {
				continue
			}
			res.Count++
			res.Total += price
			res.Balance = balance
		}
		return nil
	})
	if err != nil {
		return BulkSellResult{}, err
	}
	return res, nil
}

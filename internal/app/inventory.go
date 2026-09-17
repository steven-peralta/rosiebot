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

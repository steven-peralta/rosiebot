package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type AdminService struct {
	players PlayerStore
	source  WaifuSource
	clock   Clock
	log     *slog.Logger
}

func NewAdminService(players PlayerStore, source WaifuSource, clock Clock, log *slog.Logger) *AdminService {
	if log == nil {
		log = slog.Default()
	}
	return &AdminService{players: players, source: source, clock: clock, log: log}
}

func (s *AdminService) SetCoins(ctx context.Context, actor, key domain.PlayerKey, coins int64) (int64, error) {
	if coins < 0 {
		return 0, ErrNegativeAmount
	}
	if _, err := s.players.EnsurePlayer(ctx, key); err != nil {
		return 0, fmt.Errorf("ensure player: %w", err)
	}
	balance, err := s.players.SetCoins(ctx, key, coins)
	if err != nil {
		return 0, fmt.Errorf("set coins: %w", err)
	}
	s.log.Info("admin set coins", "actor", actor.UserID, "guild", key.GuildID, "user", key.UserID, "coins", balance)
	return balance, nil
}

func (s *AdminService) AdjustCoins(ctx context.Context, actor, key domain.PlayerKey, delta int64) (int64, error) {
	if _, err := s.players.EnsurePlayer(ctx, key); err != nil {
		return 0, fmt.Errorf("ensure player: %w", err)
	}
	balance, ok, err := s.players.AdjustCoins(ctx, key, delta)
	if err != nil {
		return 0, fmt.Errorf("adjust coins: %w", err)
	}
	if !ok {
		return 0, ErrInsufficientCoins
	}
	s.log.Info("admin adjusted coins", "actor", actor.UserID, "guild", key.GuildID, "user", key.UserID, "delta", delta, "coins", balance)
	return balance, nil
}

func (s *AdminService) GrantWaifu(ctx context.Context, actor, key domain.PlayerKey, slug string) (domain.Waifu, error) {
	if _, err := s.players.EnsurePlayer(ctx, key); err != nil {
		return domain.Waifu{}, fmt.Errorf("ensure player: %w", err)
	}
	w, err := s.source.Get(ctx, slug)
	if err != nil {
		return domain.Waifu{}, err
	}
	inserted, err := s.players.AddOwned(ctx, key, domain.OwnedFromSummary(w.WaifuSummary, s.clock.Now()))
	if err != nil {
		return domain.Waifu{}, fmt.Errorf("add owned: %w", err)
	}
	if !inserted {
		return domain.Waifu{}, ErrAlreadyOwned
	}
	s.log.Info("admin granted waifu", "actor", actor.UserID, "guild", key.GuildID, "user", key.UserID, "slug", w.Slug)
	return w, nil
}

func (s *AdminService) RevokeWaifu(ctx context.Context, actor, key domain.PlayerKey, slug string) (domain.OwnedWaifu, error) {
	owned, err := s.players.GetOwned(ctx, key, slug)
	if errors.Is(err, ErrNotFound) {
		return domain.OwnedWaifu{}, ErrNotOwned
	}
	if err != nil {
		return domain.OwnedWaifu{}, fmt.Errorf("get owned: %w", err)
	}
	_, ok, err := s.players.SellOwned(ctx, key, slug, 0)
	if err != nil {
		return domain.OwnedWaifu{}, fmt.Errorf("remove owned: %w", err)
	}
	if !ok {
		return domain.OwnedWaifu{}, ErrNotOwned
	}
	s.log.Info("admin revoked waifu", "actor", actor.UserID, "guild", key.GuildID, "user", key.UserID, "slug", slug)
	return owned, nil
}

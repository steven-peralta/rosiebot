package app

import (
	"context"
	"fmt"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type CoinsService struct {
	players PlayerStore
}

func NewCoinsService(players PlayerStore) *CoinsService {
	return &CoinsService{players: players}
}

func (s *CoinsService) Balance(ctx context.Context, key domain.PlayerKey) (int64, error) {
	player, err := s.players.EnsurePlayer(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("ensure player: %w", err)
	}
	return player.Coins, nil
}

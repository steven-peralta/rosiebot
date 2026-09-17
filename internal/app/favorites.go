package app

import (
	"context"
	"fmt"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type FavoriteService struct {
	store FavoriteStore
	clock Clock
}

func NewFavoriteService(store FavoriteStore, clock Clock) *FavoriteService {
	return &FavoriteService{store: store, clock: clock}
}

func (s *FavoriteService) Toggle(ctx context.Context, key domain.PlayerKey, fav domain.Favorite) (bool, error) {
	if !fav.Kind.Valid() || fav.Slug == "" {
		return false, fmt.Errorf("favorite %q %q: %w", fav.Kind, fav.Slug, ErrNotFound)
	}
	removed, err := s.store.Remove(ctx, key, fav.Kind, fav.Slug)
	if err != nil {
		return false, fmt.Errorf("remove favorite: %w", err)
	}
	if removed {
		return false, nil
	}
	fav.AddedAt = s.clock.Now()
	if _, err := s.store.Add(ctx, key, fav); err != nil {
		return false, fmt.Errorf("add favorite: %w", err)
	}
	return true, nil
}

func (s *FavoriteService) List(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind) ([]domain.Favorite, error) {
	favs, err := s.store.List(ctx, key, kind)
	if err != nil {
		return nil, fmt.Errorf("list favorites: %w", err)
	}
	return favs, nil
}

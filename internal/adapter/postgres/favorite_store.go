package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/steven-peralta/rosiebot/internal/adapter/postgres/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type FavoriteStore struct {
	q *gen.Queries
}

var _ app.FavoriteStore = (*FavoriteStore)(nil)

func NewFavoriteStore(pool *pgxpool.Pool) *FavoriteStore {
	return &FavoriteStore{q: gen.New(pool)}
}

func (s *FavoriteStore) Add(ctx context.Context, key domain.PlayerKey, fav domain.Favorite) (bool, error) {
	n, err := s.q.AddFavorite(ctx, gen.AddFavoriteParams{
		GuildID: key.GuildID, UserID: key.UserID, Kind: string(fav.Kind), Slug: fav.Slug,
		Name: fav.Name, Url: fav.URL, PictureUrl: fav.PictureURL, AddedAt: fav.AddedAt,
	})
	if err != nil {
		return false, fmt.Errorf("postgres: add favorite: %w", err)
	}
	return n > 0, nil
}

func (s *FavoriteStore) Remove(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind, slug string) (bool, error) {
	n, err := s.q.RemoveFavorite(ctx, gen.RemoveFavoriteParams{GuildID: key.GuildID, UserID: key.UserID, Kind: string(kind), Slug: slug})
	if err != nil {
		return false, fmt.Errorf("postgres: remove favorite: %w", err)
	}
	return n > 0, nil
}

func (s *FavoriteStore) List(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind) ([]domain.Favorite, error) {
	rows, err := s.q.ListFavorites(ctx, gen.ListFavoritesParams{GuildID: key.GuildID, UserID: key.UserID, Kind: string(kind)})
	if err != nil {
		return nil, fmt.Errorf("postgres: list favorites: %w", err)
	}
	out := make([]domain.Favorite, len(rows))
	for i, r := range rows {
		out[i] = domain.Favorite{Kind: domain.FavoriteKind(r.Kind), Slug: r.Slug, Name: r.Name, URL: r.Url, PictureURL: r.PictureUrl, AddedAt: r.AddedAt}
	}
	return out, nil
}

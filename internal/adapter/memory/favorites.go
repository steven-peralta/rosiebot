package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type favKey struct {
	key  domain.PlayerKey
	kind domain.FavoriteKind
	slug string
}

type FavoriteStore struct {
	mu   sync.Mutex
	favs map[favKey]domain.Favorite
}

var _ app.FavoriteStore = (*FavoriteStore)(nil)

func NewFavoriteStore() *FavoriteStore {
	return &FavoriteStore{favs: map[favKey]domain.Favorite{}}
}

func (s *FavoriteStore) Add(ctx context.Context, key domain.PlayerKey, fav domain.Favorite) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := favKey{key, fav.Kind, fav.Slug}
	if _, ok := s.favs[k]; ok {
		return false, nil
	}
	s.favs[k] = fav
	return true, nil
}

func (s *FavoriteStore) Remove(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind, slug string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := favKey{key, kind, slug}
	if _, ok := s.favs[k]; !ok {
		return false, nil
	}
	delete(s.favs, k)
	return true, nil
}

func (s *FavoriteStore) List(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind) ([]domain.Favorite, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []domain.Favorite{}
	for k, f := range s.favs {
		if k.key == key && k.kind == kind {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].AddedAt.Equal(out[j].AddedAt) {
			return out[i].AddedAt.Before(out[j].AddedAt)
		}
		return out[i].Slug < out[j].Slug
	})
	return out, nil
}

func (s *FavoriteStore) Find(ctx context.Context, kind domain.FavoriteKind, slugs []string, guildID string) ([]app.FavoriteMatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	want := make(map[string]struct{}, len(slugs))
	for _, slug := range slugs {
		want[slug] = struct{}{}
	}
	out := []app.FavoriteMatch{}
	for k, f := range s.favs {
		if k.kind != kind {
			continue
		}
		if guildID != "" && k.key.GuildID != guildID {
			continue
		}
		if _, ok := want[k.slug]; ok {
			out = append(out, app.FavoriteMatch{Key: k.key, Favorite: f})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key.UserID != out[j].Key.UserID {
			return out[i].Key.UserID < out[j].Key.UserID
		}
		if out[i].Key.GuildID != out[j].Key.GuildID {
			return out[i].Key.GuildID < out[j].Key.GuildID
		}
		return out[i].Favorite.Slug < out[j].Favorite.Slug
	})
	return out, nil
}

package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type failingFavorites struct{ err error }

func (f failingFavorites) Add(context.Context, domain.PlayerKey, domain.Favorite) (bool, error) {
	return false, f.err
}

func (f failingFavorites) Remove(context.Context, domain.PlayerKey, domain.FavoriteKind, string) (bool, error) {
	return false, f.err
}

func (f failingFavorites) List(context.Context, domain.PlayerKey, domain.FavoriteKind) ([]domain.Favorite, error) {
	return nil, f.err
}

func TestFavoriteService_ToggleAndList(t *testing.T) {
	f := newFixture(t)
	store := memory.NewFavoriteStore()
	svc := app.NewFavoriteService(store, f.clock)
	rem := domain.Favorite{Kind: domain.FavoriteWaifu, Slug: "rem", Name: "Rem"}
	rezero := domain.Favorite{Kind: domain.FavoriteSeries, Slug: "re-zero", Name: "Re:Zero"}

	if added, err := svc.Toggle(f.ctx, alice, rem); err != nil || !added {
		t.Fatalf("first toggle = %v %v", added, err)
	}
	f.clock.Advance(1)
	if added, err := svc.Toggle(f.ctx, alice, rezero); err != nil || !added {
		t.Fatalf("series toggle = %v %v", added, err)
	}
	waifus, err := svc.List(f.ctx, alice, domain.FavoriteWaifu)
	if err != nil || len(waifus) != 1 || waifus[0].Slug != "rem" || !waifus[0].AddedAt.Equal(f.clock.now.Add(-1)) {
		t.Errorf("waifu favorites = %+v %v", waifus, err)
	}
	series, err := svc.List(f.ctx, alice, domain.FavoriteSeries)
	if err != nil || len(series) != 1 || series[0].Slug != "re-zero" {
		t.Errorf("series favorites = %+v %v", series, err)
	}
	if others, _ := svc.List(f.ctx, bob, domain.FavoriteWaifu); len(others) != 0 {
		t.Errorf("favorites leaked across players: %+v", others)
	}
	if added, err := svc.Toggle(f.ctx, alice, rem); err != nil || added {
		t.Errorf("second toggle should remove: %v %v", added, err)
	}
	if waifus, _ := svc.List(f.ctx, alice, domain.FavoriteWaifu); len(waifus) != 0 {
		t.Errorf("favorite should be gone: %+v", waifus)
	}
	if _, err := svc.Toggle(f.ctx, alice, domain.Favorite{Kind: "studio", Slug: "x"}); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("bad kind = %v", err)
	}
	if _, err := svc.Toggle(f.ctx, alice, domain.Favorite{Kind: domain.FavoriteWaifu}); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("empty slug = %v", err)
	}
	broken := app.NewFavoriteService(failingFavorites{err: errors.New("boom")}, f.clock)
	if _, err := broken.Toggle(f.ctx, alice, rem); err == nil {
		t.Error("store error should surface from toggle")
	}
	if _, err := broken.List(f.ctx, alice, domain.FavoriteWaifu); err == nil {
		t.Error("store error should surface from list")
	}
}

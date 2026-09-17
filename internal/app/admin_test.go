package app_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func (f *fixture) admin() *app.AdminService {
	return app.NewAdminService(f.players, f.source, f.clock, nil)
}

func TestAdminService_Coins(t *testing.T) {
	f := newFixture(t)
	svc := f.admin()

	if got, err := svc.SetCoins(f.ctx, alice, bob, 1000); err != nil || got != 1000 || f.coins(bob) != 1000 {
		t.Errorf("SetCoins on a new player = %d %v (coins=%d)", got, err, f.coins(bob))
	}
	if _, err := svc.SetCoins(f.ctx, alice, bob, -1); !errors.Is(err, app.ErrNegativeAmount) {
		t.Errorf("negative set = %v", err)
	}
	if got, err := svc.AdjustCoins(f.ctx, alice, bob, 250); err != nil || got != 1250 {
		t.Errorf("increment = %d %v", got, err)
	}
	if got, err := svc.AdjustCoins(f.ctx, alice, bob, -1250); err != nil || got != 0 {
		t.Errorf("decrement to zero = %d %v", got, err)
	}
	if _, err := svc.AdjustCoins(f.ctx, alice, bob, -1); !errors.Is(err, app.ErrInsufficientCoins) {
		t.Errorf("decrement below zero = %v", err)
	}
	if got, err := svc.AdjustCoins(f.ctx, alice, domain.PlayerKey{GuildID: "guild", UserID: "new"}, 5); err != nil || got != domain.StartingCoins+5 {
		t.Errorf("increment creates the player first = %d %v", got, err)
	}
}

func TestAdminService_GrantAndRevoke(t *testing.T) {
	f := newFixture(t)
	svc := f.admin()
	f.source.EXPECT().Get(mock.Anything, "rem").Return(detail("rem"), nil).Twice()

	w, err := svc.GrantWaifu(f.ctx, alice, bob, "rem")
	if err != nil || w.Slug != "rem" || !f.owns(bob, "rem") {
		t.Fatalf("grant = %+v %v owns=%v", w, err, f.owns(bob, "rem"))
	}
	if f.coins(bob) != domain.StartingCoins {
		t.Errorf("grant must not touch coins: %d", f.coins(bob))
	}
	if _, err := svc.GrantWaifu(f.ctx, alice, bob, "rem"); !errors.Is(err, app.ErrAlreadyOwned) {
		t.Errorf("second grant = %v", err)
	}
	f.source.EXPECT().Get(mock.Anything, "ghost").Return(domain.Waifu{}, app.ErrNotFound).Once()
	if _, err := svc.GrantWaifu(f.ctx, alice, bob, "ghost"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("unknown slug = %v", err)
	}

	got, err := svc.RevokeWaifu(f.ctx, alice, bob, "rem")
	if err != nil || got.Slug != "rem" || f.owns(bob, "rem") {
		t.Errorf("revoke = %+v %v owns=%v", got, err, f.owns(bob, "rem"))
	}
	if f.coins(bob) != domain.StartingCoins {
		t.Errorf("revoke must not pay out: %d", f.coins(bob))
	}
	if _, err := svc.RevokeWaifu(f.ctx, alice, bob, "rem"); !errors.Is(err, app.ErrNotOwned) {
		t.Errorf("second revoke = %v", err)
	}
	if _, err := svc.RevokeWaifu(f.ctx, alice, domain.PlayerKey{GuildID: "guild", UserID: "nobody"}, "rem"); !errors.Is(err, app.ErrNotOwned) {
		t.Errorf("revoke from unknown player = %v", err)
	}
}

func TestAdminService_StoreErrorsPropagate(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"EnsurePlayer", "SetCoins", "AdjustCoins", "AddOwned", "GetOwned", "SellOwned"} {
		f.give(bob, "rem")
		store := failStore{PlayerStore: f.players, fail: map[string]bool{name: true}}
		svc := app.NewAdminService(store, f.source, f.clock, nil)
		f.source.ExpectedCalls = nil
		f.source.EXPECT().Get(mock.Anything, mock.Anything).Return(detail("ram"), nil).Maybe()
		var errs []error
		_, err := svc.SetCoins(f.ctx, alice, bob, 1)
		errs = append(errs, err)
		_, err = svc.AdjustCoins(f.ctx, alice, bob, 1)
		errs = append(errs, err)
		_, err = svc.GrantWaifu(f.ctx, alice, bob, "ram")
		errs = append(errs, err)
		_, err = svc.RevokeWaifu(f.ctx, alice, bob, "rem")
		errs = append(errs, err)
		failed := false
		for _, e := range errs {
			if errors.Is(e, errStore) {
				failed = true
			}
		}
		if !failed {
			t.Errorf("%s failure should surface, got %v", name, errs)
		}
	}
}

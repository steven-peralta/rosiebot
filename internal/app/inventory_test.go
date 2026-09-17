package app_test

import (
	"errors"
	"testing"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestCoinsService_BalanceCreatesPlayer(t *testing.T) {
	f := newFixture(t)
	bal, err := app.NewCoinsService(f.players).Balance(f.ctx, bob)
	if err != nil {
		t.Fatal(err)
	}
	if bal != domain.StartingCoins {
		t.Errorf("balance = %d", bal)
	}
}

func TestInventoryService_ListOrderAndEmpty(t *testing.T) {
	f := newFixture(t)
	svc := app.NewInventoryService(f.players, f.ranking)

	empty, err := svc.List(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Errorf("new player should own nothing, got %d", len(empty))
	}

	f.give(alice, "c", "a", "b")
	owned, err := svc.List(f.ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 3 || owned[0].Slug != "c" || owned[1].Slug != "a" || owned[2].Slug != "b" {
		t.Errorf("owned order = %+v, want acquisition order", owned)
	}
}

func TestInventoryService_Sell(t *testing.T) {
	f := newFixture(t)
	svc := app.NewInventoryService(f.players, f.ranking)
	f.give(alice, "rem")

	res, err := svc.Sell(f.ctx, alice, "rem")
	if err != nil {
		t.Fatal(err)
	}
	if res.Waifu.Slug != "rem" || res.Price != domain.SellPrice || res.Balance != domain.StartingCoins+domain.SellPrice {
		t.Errorf("sell result = %+v", res)
	}
	if f.owns(alice, "rem") {
		t.Error("waifu still owned after sale")
	}

	if _, err := svc.Sell(f.ctx, alice, "rem"); !errors.Is(err, app.ErrNotOwned) {
		t.Errorf("second sale err = %v, want ErrNotOwned", err)
	}

	f.ranking.Set(rankingOf(200))
	f.give(alice, "ranked-000", "ranked-100")
	before := f.coins(alice)
	top, err := svc.Sell(f.ctx, alice, "ranked-000")
	if err != nil || top.Stars != 5 || top.Price != 1000 || top.Balance != before+1000 {
		t.Errorf("selling a 5-star = %+v %v", top, err)
	}
	low, err := svc.Sell(f.ctx, alice, "ranked-100")
	if err != nil || low.Stars != 1 || low.Price != 150 || low.Balance != before+1150 {
		t.Errorf("selling a 1-star = %+v %v", low, err)
	}
	if price, stars := svc.Price("ghost"); price != domain.SellPrice || stars != 0 {
		t.Errorf("unranked quote = %d %d", price, stars)
	}
	if price, _ := app.NewInventoryService(f.players, nil).Price("ranked-000"); price != domain.SellPrice {
		t.Errorf("without a ranking provider the base price applies, got %d", price)
	}
	if _, err := svc.Sell(f.ctx, bob, "rem"); !errors.Is(err, app.ErrNotOwned) {
		t.Errorf("sale by non-owner err = %v, want ErrNotOwned", err)
	}
}

func TestInventoryService_OwnsAndSuggest(t *testing.T) {
	f := newFixture(t)
	svc := app.NewInventoryService(f.players, f.ranking)
	f.give(alice, "rem", "ram", "emilia")

	if _, ok, err := svc.Owns(f.ctx, alice, "ram"); err != nil || !ok {
		t.Errorf("Owns(ram) = %v, %v", ok, err)
	}
	if _, ok, err := svc.Owns(f.ctx, alice, "subaru"); err != nil || ok {
		t.Errorf("Owns(subaru) = %v, %v", ok, err)
	}

	got, err := svc.Suggest(f.ctx, alice, "name r", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("Suggest prefix matched %d, want 2 (rem, ram)", len(got))
	}
	got, err = svc.Suggest(f.ctx, alice, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("Suggest limit returned %d", len(got))
	}
}

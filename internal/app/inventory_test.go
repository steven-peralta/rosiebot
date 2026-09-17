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

func TestInventoryService_QuoteAndSellBelow(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))
	svc := app.NewInventoryService(f.players, f.ranking)
	f.give(alice, "ranked-000", "ranked-005", "ranked-020", "ranked-040", "ranked-100", "plain-a", "plain-b")

	q, err := svc.QuoteBelow(f.ctx, alice, 2)
	if err != nil {
		t.Fatal(err)
	}
	if q.Count != 4 || q.Total != 100+100+150+200 || q.Tiers[0] != 2 || q.Tiers[1] != 1 || q.Tiers[2] != 1 || q.Tiers[3] != 0 {
		t.Errorf("quote at 2 stars = %+v", q)
	}
	if q, _ := svc.QuoteBelow(f.ctx, alice, 0); q.Count != 2 || q.Total != 200 {
		t.Errorf("quote unranked only = %+v", q)
	}
	if q, _ := svc.QuoteBelow(f.ctx, alice, domain.MaxStars); q.Count != 7 || q.Total != 100+100+150+200+300+500+1000 {
		t.Errorf("quote everything = %+v", q)
	}

	res, err := svc.SellBelow(f.ctx, alice, 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 4 || res.Total != 550 || res.Balance != domain.StartingCoins+550 || f.coins(alice) != domain.StartingCoins+550 {
		t.Errorf("sell below 2 = %+v coins=%d", res, f.coins(alice))
	}
	for _, slug := range []string{"plain-a", "plain-b", "ranked-100", "ranked-040"} {
		if f.owns(alice, slug) {
			t.Errorf("%s should have been sold", slug)
		}
	}
	for _, slug := range []string{"ranked-000", "ranked-005", "ranked-020"} {
		if !f.owns(alice, slug) {
			t.Errorf("%s should have been kept", slug)
		}
	}
	again, err := svc.SellBelow(f.ctx, alice, 2)
	if err != nil || again.Count != 0 || again.Total != 0 || again.Balance != domain.StartingCoins+550 {
		t.Errorf("nothing left to sell = %+v %v", again, err)
	}
	if res, err := svc.SellBelow(f.ctx, bob, domain.MaxStars); err != nil || res.Count != 0 || res.Balance != domain.StartingCoins {
		t.Errorf("sell below for a new player = %+v %v", res, err)
	}

	f.give(bob, "plain-c")
	for _, name := range []string{"EnsurePlayer", "WithinTx", "ListOwned", "SellOwned"} {
		broken := app.NewInventoryService(failStore{PlayerStore: f.players, fail: map[string]bool{name: true}}, f.ranking)
		if _, err := broken.SellBelow(f.ctx, bob, domain.MaxStars); !errors.Is(err, errStore) {
			t.Errorf("%s failure should surface, got %v", name, err)
		}
		if _, err := broken.QuoteBelow(f.ctx, bob, domain.MaxStars); name != "SellOwned" && name != "WithinTx" && !errors.Is(err, errStore) {
			t.Errorf("%s failure should surface from the quote, got %v", name, err)
		}
	}
	if !f.owns(bob, "plain-c") {
		t.Error("a failed bulk sale must roll back")
	}
}

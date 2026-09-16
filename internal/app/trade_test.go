package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func offer(give, receive []string) domain.TradeOffer {
	return domain.TradeOffer{Sender: alice, Target: bob, Give: give, Receive: receive}
}

func TestTradeService_ProposeValidatesAndNames(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem")
	f.give(bob, "ram")
	svc := app.NewTradeService(f.players)

	p, err := svc.Propose(f.ctx, offer([]string{"rem", "rem"}, []string{"ram"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Give) != 1 || p.Give[0].Name != "Name rem" || len(p.Receive) != 1 || p.Receive[0].Slug != "ram" {
		t.Errorf("proposal = %+v", p)
	}
	if len(p.Offer.Give) != 1 {
		t.Errorf("offer should be normalised: %+v", p.Offer)
	}
}

func TestTradeService_ProposeRejections(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem", "shared")
	f.give(bob, "ram", "shared")
	svc := app.NewTradeService(f.players)

	cases := []struct {
		name  string
		offer domain.TradeOffer
		want  error
		slug  string
		side  domain.TradeSide
	}{
		{name: "self", offer: domain.TradeOffer{Sender: alice, Target: alice, Give: []string{"rem"}}, want: domain.ErrTradeWithSelf},
		{name: "empty", offer: offer(nil, nil), want: domain.ErrTradeEmpty},
		{name: "sender lacks", offer: offer([]string{"ghost"}, nil), want: domain.ErrTradeNotOwned, slug: "ghost", side: domain.TradeSideSender},
		{name: "target has", offer: offer([]string{"shared"}, nil), want: domain.ErrTradeAlreadyOwn, slug: "shared", side: domain.TradeSideTarget},
		{name: "target lacks", offer: offer(nil, []string{"ghost"}), want: domain.ErrTradeNotOwned, slug: "ghost", side: domain.TradeSideTarget},
		{name: "sender has", offer: offer(nil, []string{"shared"}), want: domain.ErrTradeAlreadyOwn, slug: "shared", side: domain.TradeSideSender},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Propose(f.ctx, c.offer)
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if c.slug != "" {
				var v *domain.TradeViolation
				if !errors.As(err, &v) || v.Slug != c.slug || v.Side != c.side {
					t.Errorf("violation = %+v", v)
				}
			}
		})
	}
}

func TestTradeService_ProposeCreatesTargetPlayer(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem")
	if _, err := app.NewTradeService(f.players).Propose(f.ctx, offer([]string{"rem"}, nil)); err != nil {
		t.Fatalf("gift to a player with no row: %v", err)
	}
	if _, err := f.players.GetPlayer(f.ctx, bob); err != nil {
		t.Error("target player row should exist after proposal")
	}
}

func TestTradeService_AcceptSwapsBothSides(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem", "keep")
	f.give(bob, "ram")
	svc := app.NewTradeService(f.players)

	if err := svc.Accept(f.ctx, offer([]string{"rem"}, []string{"ram"})); err != nil {
		t.Fatal(err)
	}
	if !f.owns(bob, "rem") || !f.owns(alice, "ram") || f.owns(alice, "rem") || f.owns(bob, "ram") || !f.owns(alice, "keep") {
		t.Error("inventories not swapped correctly")
	}
}

func TestTradeService_AcceptGift(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem")
	f.give(bob)
	if err := app.NewTradeService(f.players).Accept(f.ctx, offer([]string{"rem"}, nil)); err != nil {
		t.Fatal(err)
	}
	if !f.owns(bob, "rem") || f.owns(alice, "rem") {
		t.Error("gift not transferred")
	}
}

func TestTradeService_AcceptRevalidates(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem")
	f.give(bob, "ram")
	svc := app.NewTradeService(f.players)
	o := offer([]string{"rem"}, []string{"ram"})
	if _, err := svc.Propose(f.ctx, o); err != nil {
		t.Fatal(err)
	}

	if _, ok, err := f.players.SellOwned(f.ctx, bob, "ram", domain.SellPrice); err != nil || !ok {
		t.Fatal("setup sale failed")
	}

	err := svc.Accept(f.ctx, o)
	if !errors.Is(err, app.ErrTradeConflict) {
		t.Fatalf("err = %v, want ErrTradeConflict", err)
	}
	var v *domain.TradeViolation
	if !errors.As(err, &v) || v.Slug != "ram" {
		t.Errorf("conflict should carry the violation: %v", err)
	}
	if !f.owns(alice, "rem") {
		t.Error("failed trade must not move anything")
	}
}

func TestTradeService_AcceptRejectsMalformedOffer(t *testing.T) {
	f := newFixture(t)
	if err := app.NewTradeService(f.players).Accept(f.ctx, offer(nil, nil)); !errors.Is(err, domain.ErrTradeEmpty) {
		t.Errorf("err = %v", err)
	}
}

func TestTradeService_AcceptFailsWhenPlayersMissing(t *testing.T) {
	f := newFixture(t)
	if err := app.NewTradeService(f.players).Accept(f.ctx, offer([]string{"rem"}, nil)); err == nil {
		t.Error("accept without player rows should fail")
	}
}

type shortTransferStore struct {
	app.PlayerStore
}

func (s shortTransferStore) WithinTx(ctx context.Context, fn func(app.PlayerRepo) error) error {
	return s.PlayerStore.WithinTx(ctx, func(r app.PlayerRepo) error {
		return fn(shortTransferRepo{r})
	})
}

type shortTransferRepo struct {
	app.PlayerRepo
}

func (shortTransferRepo) TransferOwned(context.Context, domain.PlayerKey, domain.PlayerKey, []string) (int64, error) {
	return 0, nil
}

func TestTradeService_AcceptDetectsPartialTransfer(t *testing.T) {
	f := newFixture(t)
	f.give(alice, "rem")
	f.give(bob, "ram")
	svc := app.NewTradeService(shortTransferStore{f.players})
	if err := svc.Accept(f.ctx, offer([]string{"rem"}, nil)); !errors.Is(err, app.ErrTradeConflict) {
		t.Errorf("give short transfer err = %v", err)
	}
	if err := svc.Accept(f.ctx, offer(nil, []string{"ram"})); !errors.Is(err, app.ErrTradeConflict) {
		t.Errorf("receive short transfer err = %v", err)
	}
}

package domain

import (
	"errors"
	"slices"
	"testing"
)

var (
	alice = PlayerKey{GuildID: "g", UserID: "alice"}
	bob   = PlayerKey{GuildID: "g", UserID: "bob"}
)

func TestNormaliseTrade_Dedupe(t *testing.T) {
	offer, err := NormaliseTrade(TradeOffer{Sender: alice, Target: bob, Give: []string{"b", "a", "b", ""}, Receive: []string{"c", "c"}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(offer.Give, []string{"a", "b"}) || !slices.Equal(offer.Receive, []string{"c"}) {
		t.Errorf("normalised = %+v", offer)
	}
}

func TestNormaliseTrade_NoSelf(t *testing.T) {
	if _, err := NormaliseTrade(TradeOffer{Sender: alice, Target: alice, Give: []string{"a"}}); !errors.Is(err, ErrTradeWithSelf) {
		t.Errorf("err = %v, want ErrTradeWithSelf", err)
	}
}

func TestNormaliseTrade_NotBothEmpty(t *testing.T) {
	if _, err := NormaliseTrade(TradeOffer{Sender: alice, Target: bob}); !errors.Is(err, ErrTradeEmpty) {
		t.Errorf("err = %v, want ErrTradeEmpty", err)
	}
	if _, err := NormaliseTrade(TradeOffer{Sender: alice, Target: bob, Give: []string{""}}); !errors.Is(err, ErrTradeEmpty) {
		t.Errorf("blank slugs only: err = %v, want ErrTradeEmpty", err)
	}
}

func TestNormaliseTrade_NoOverlap(t *testing.T) {
	if _, err := NormaliseTrade(TradeOffer{Sender: alice, Target: bob, Give: []string{"a"}, Receive: []string{"a"}}); !errors.Is(err, ErrTradeOverlap) {
		t.Errorf("err = %v, want ErrTradeOverlap", err)
	}
}

func TestNormaliseTrade_GiftAllowed(t *testing.T) {
	if _, err := NormaliseTrade(TradeOffer{Sender: alice, Target: bob, Give: []string{"a"}}); err != nil {
		t.Errorf("gift from sender: %v", err)
	}
	if _, err := NormaliseTrade(TradeOffer{Sender: alice, Target: bob, Receive: []string{"a"}}); err != nil {
		t.Errorf("gift to sender: %v", err)
	}
}

func TestValidateTrade_Matrix(t *testing.T) {
	cases := []struct {
		name       string
		give, recv []string
		senderOwns []string
		targetOwns []string
		wantSlug   string
		wantSide   TradeSide
		wantErr    error
	}{
		{name: "valid two-way", give: []string{"a"}, recv: []string{"b"}, senderOwns: []string{"a"}, targetOwns: []string{"b"}},
		{name: "valid gift", give: []string{"a"}, senderOwns: []string{"a"}, targetOwns: []string{}},
		{name: "sender doesn't own", give: []string{"a"}, senderOwns: []string{}, targetOwns: []string{}, wantSlug: "a", wantSide: TradeSideSender, wantErr: ErrTradeNotOwned},
		{name: "target already owns", give: []string{"a"}, senderOwns: []string{"a"}, targetOwns: []string{"a"}, wantSlug: "a", wantSide: TradeSideTarget, wantErr: ErrTradeAlreadyOwn},
		{name: "target doesn't own", recv: []string{"b"}, senderOwns: []string{}, targetOwns: []string{}, wantSlug: "b", wantSide: TradeSideTarget, wantErr: ErrTradeNotOwned},
		{name: "sender already owns", recv: []string{"b"}, senderOwns: []string{"b"}, targetOwns: []string{"b"}, wantSlug: "b", wantSide: TradeSideSender, wantErr: ErrTradeAlreadyOwn},
		{name: "first failure reported", give: []string{"a", "z"}, senderOwns: []string{"a"}, targetOwns: []string{}, wantSlug: "z", wantSide: TradeSideSender, wantErr: ErrTradeNotOwned},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateTrade(TradeOffer{Sender: alice, Target: bob, Give: c.give, Receive: c.recv}, c.senderOwns, c.targetOwns)
			if c.wantErr == nil {
				if err != nil {
					t.Fatalf("unexpected error %v", err)
				}
				return
			}
			var v *TradeViolation
			if !errors.As(err, &v) {
				t.Fatalf("err = %v, want TradeViolation", err)
			}
			if v.Slug != c.wantSlug || v.Side != c.wantSide || !errors.Is(err, c.wantErr) {
				t.Errorf("violation = %+v, want slug=%s side=%d err=%v", v, c.wantSlug, c.wantSide, c.wantErr)
			}
			if v.Error() == "" {
				t.Error("violation should render a message")
			}
		})
	}
}

func TestTradeViolation_Message(t *testing.T) {
	v := &TradeViolation{Slug: "rem", Side: TradeSideTarget, Err: ErrTradeNotOwned}
	if got := v.Error(); got != "target: waifu not owned (rem)" {
		t.Errorf("Error() = %q", got)
	}
	s := &TradeViolation{Slug: "rem", Side: TradeSideSender, Err: ErrTradeAlreadyOwn}
	if got := s.Error(); got != "sender: waifu already owned (rem)" {
		t.Errorf("Error() = %q", got)
	}
}

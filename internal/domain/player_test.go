package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewPlayer_StartsWithV1Balance(t *testing.T) {
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	p := NewPlayer(alice, now)
	if p.Coins != 200 || p.Key != alice || !p.CreatedAt.Equal(now) || p.DailyClaimedAt != nil {
		t.Errorf("NewPlayer = %+v", p)
	}
}

func TestPlayer_Debit(t *testing.T) {
	p := NewPlayer(alice, time.Time{})
	if err := p.Debit(RollCost); err != nil || p.Coins != 0 {
		t.Fatalf("debit exact balance: err=%v coins=%d", err, p.Coins)
	}
	if err := p.Debit(1); !errors.Is(err, ErrInsufficientCoins) || p.Coins != 0 {
		t.Errorf("overdraft: err=%v coins=%d", err, p.Coins)
	}
	if err := p.Debit(0); !errors.Is(err, ErrNegativeAmount) {
		t.Errorf("zero debit: err=%v", err)
	}
	if err := p.Debit(-3); !errors.Is(err, ErrNegativeAmount) {
		t.Errorf("negative debit: err=%v", err)
	}
}

func TestPlayer_Credit(t *testing.T) {
	p := NewPlayer(alice, time.Time{})
	if err := p.Credit(SellPrice); err != nil || p.Coins != 300 {
		t.Fatalf("credit: err=%v coins=%d", err, p.Coins)
	}
	if err := p.Credit(0); !errors.Is(err, ErrNegativeAmount) {
		t.Errorf("zero credit: err=%v", err)
	}
}

func TestWaifu_FirstSeries(t *testing.T) {
	var w Waifu
	if _, ok := w.FirstSeries(); ok {
		t.Error("no appearances should report none")
	}
	w.Appearances = []Series{{Slug: "a"}, {Slug: "b"}}
	if s, ok := w.FirstSeries(); !ok || s.Slug != "a" {
		t.Errorf("FirstSeries = %+v, %v", s, ok)
	}
}

func TestOwnedFromSummary(t *testing.T) {
	now := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
	s := WaifuSummary{Slug: "rem", UUID: "u", Name: "Rem", PictureURL: "p", Likes: 3, Trash: 1}
	o := OwnedFromSummary(s, now)
	if o.Slug != "rem" || o.UUID != "u" || o.Name != "Rem" || o.PictureURL != "p" || o.Likes != 3 || o.Trash != 1 || !o.AcquiredAt.Equal(now) {
		t.Errorf("OwnedFromSummary = %+v", o)
	}
	if s.TotalVotes() != 4 {
		t.Errorf("TotalVotes = %d", s.TotalVotes())
	}
}

package domain

import (
	"testing"
	"time"
)

func TestWotdDay_MidnightBoundary(t *testing.T) {
	loc := chicago(t)
	if got := WotdDay(at(loc, 2026, 9, 14, 23, 59), loc); !got.Equal(at(loc, 2026, 9, 14, 0, 0)) {
		t.Errorf("day before midnight = %v", got)
	}
	if got := WotdDay(at(loc, 2026, 9, 15, 0, 0), loc); !got.Equal(at(loc, 2026, 9, 15, 0, 0)) {
		t.Errorf("day at midnight = %v", got)
	}
	utc := at(loc, 2026, 9, 14, 23, 30).UTC()
	if got := WotdDay(utc, loc); !got.Equal(at(loc, 2026, 9, 14, 0, 0)) {
		t.Errorf("day from UTC input = %v", got)
	}
}

func TestWotdRefreshIn(t *testing.T) {
	loc := chicago(t)
	now := at(loc, 2026, 9, 14, 18, 0)
	if got := WotdRefreshIn(now, loc); got != 6*time.Hour {
		t.Errorf("WotdRefreshIn = %v, want 6h", got)
	}
}

func TestWotdEligible_StarsBetween1And4(t *testing.T) {
	for stars := 0; stars <= 6; stars++ {
		got := WotdEligible(RankedWaifu{Stars: stars})
		want := stars >= 1 && stars <= 4
		if got != want {
			t.Errorf("WotdEligible(stars=%d) = %v, want %v", stars, got, want)
		}
	}
}

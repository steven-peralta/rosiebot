package domain

import (
	"testing"
	"time"
)

func chicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func at(loc *time.Location, y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, loc)
}

func TestDailyWindowStart_AroundTen(t *testing.T) {
	loc := chicago(t)
	cases := []struct {
		now  time.Time
		want time.Time
	}{
		{at(loc, 2026, 9, 14, 9, 59), at(loc, 2026, 9, 13, 10, 0)},
		{at(loc, 2026, 9, 14, 10, 0), at(loc, 2026, 9, 14, 10, 0)},
		{at(loc, 2026, 9, 14, 23, 30), at(loc, 2026, 9, 14, 10, 0)},
		{at(loc, 2026, 9, 15, 0, 0), at(loc, 2026, 9, 14, 10, 0)},
	}
	for _, c := range cases {
		if got := DailyWindowStart(c.now, loc); !got.Equal(c.want) {
			t.Errorf("DailyWindowStart(%v) = %v, want %v", c.now, got, c.want)
		}
	}
}

func TestDailyWindowStart_UsesBotZoneForUTCInput(t *testing.T) {
	loc := chicago(t)
	nowUTC := at(loc, 2026, 9, 14, 9, 30).UTC()
	if got := DailyWindowStart(nowUTC, loc); !got.Equal(at(loc, 2026, 9, 13, 10, 0)) {
		t.Errorf("window start from UTC input = %v", got)
	}
}

func TestDailyWindowStart_DST(t *testing.T) {
	loc := chicago(t)
	// DST starts 2026-03-08 02:00 in Chicago.
	before := at(loc, 2026, 3, 8, 9, 0)
	if got := DailyWindowStart(before, loc); !got.Equal(at(loc, 2026, 3, 7, 10, 0)) {
		t.Errorf("window before DST switch = %v", got)
	}
	after := at(loc, 2026, 3, 8, 11, 0)
	if got := DailyWindowStart(after, loc); !got.Equal(at(loc, 2026, 3, 8, 10, 0)) {
		t.Errorf("window after DST switch = %v", got)
	}
	if d := NextDailyReset(before, loc).Sub(DailyWindowStart(before, loc)); d != 23*time.Hour {
		t.Errorf("window spanning spring forward should be 23h, got %v", d)
	}
}

func TestDailyClaimAllowed(t *testing.T) {
	loc := chicago(t)
	now := at(loc, 2026, 9, 14, 12, 0)
	if !DailyClaimAllowed(nil, now, loc) {
		t.Error("never claimed should be allowed")
	}
	earlier := at(loc, 2026, 9, 14, 9, 59)
	if !DailyClaimAllowed(&earlier, now, loc) {
		t.Error("claim before today's 10:00 should allow another")
	}
	sameWindow := at(loc, 2026, 9, 14, 10, 0)
	if DailyClaimAllowed(&sameWindow, now, loc) {
		t.Error("claim at window start should block")
	}
	later := at(loc, 2026, 9, 14, 11, 0)
	if DailyClaimAllowed(&later, now, loc) {
		t.Error("claim inside the window should block")
	}
}

func TestDailyRefreshIn(t *testing.T) {
	loc := chicago(t)
	now := at(loc, 2026, 9, 14, 12, 0)
	if got := DailyRefreshIn(now, loc); got != 22*time.Hour {
		t.Errorf("DailyRefreshIn = %v, want 22h", got)
	}
}

func TestFormatCountdown(t *testing.T) {
	cases := map[time.Duration]string{
		0:                                    "00:00:00",
		-5 * time.Second:                     "00:00:00",
		9*time.Second + 900*time.Millisecond: "00:00:09",
		61 * time.Second:                     "00:01:01",
		23*time.Hour + 59*time.Minute + 59*time.Second: "23:59:59",
		25 * time.Hour: "25:00:00",
	}
	for d, want := range cases {
		if got := FormatCountdown(d); got != want {
			t.Errorf("FormatCountdown(%v) = %q, want %q", d, got, want)
		}
	}
}

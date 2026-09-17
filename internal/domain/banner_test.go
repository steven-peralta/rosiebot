package domain

import (
	"testing"
	"time"
)

func TestBannerConstants(t *testing.T) {
	if BannerRollCost != 400 || BannerMinRanked != 5 || BannerMinStars != 4 {
		t.Errorf("banner constants drifted: cost=%d minRanked=%d minStars=%d", BannerRollCost, BannerMinRanked, BannerMinStars)
	}
	if BannerResetWeekday != time.Monday || BannerResetHour != 10 {
		t.Errorf("banner boundary = %v %d, want Monday 10", BannerResetWeekday, BannerResetHour)
	}
}

func TestBannerWeekStart_AroundMondayTen(t *testing.T) {
	loc := chicago(t)
	monday := at(loc, 2026, 9, 14, 10, 0)
	cases := map[string]struct {
		now  time.Time
		want time.Time
	}{
		"monday 09:59":     {at(loc, 2026, 9, 14, 9, 59), at(loc, 2026, 9, 7, 10, 0)},
		"monday 10:00":     {monday, monday},
		"wednesday noon":   {at(loc, 2026, 9, 16, 12, 0), monday},
		"sunday 23:00":     {at(loc, 2026, 9, 20, 23, 0), monday},
		"next monday 9:59": {at(loc, 2026, 9, 21, 9, 59), monday},
		"utc input":        {at(loc, 2026, 9, 14, 10, 30).UTC(), monday},
	}
	for name, c := range cases {
		if got := BannerWeekStart(c.now, loc); !got.Equal(c.want) {
			t.Errorf("%s: week start = %v, want %v", name, got, c.want)
		}
	}
}

func TestBannerWeekStart_DST(t *testing.T) {
	loc := chicago(t)
	spring := at(loc, 2026, 3, 4, 12, 0)
	if start := BannerWeekStart(spring, loc); !start.Equal(at(loc, 2026, 3, 2, 10, 0)) {
		t.Errorf("spring week start = %v", start)
	}
	if d := NextBannerReset(spring, loc).Sub(BannerWeekStart(spring, loc)); d != 167*time.Hour {
		t.Errorf("week spanning spring forward should be 167h, got %v", d)
	}
	if next := NextBannerReset(spring, loc); next.Hour() != 10 || next.Weekday() != time.Monday {
		t.Errorf("next reset after spring forward = %v, want Monday 10:00", next)
	}
	fall := at(loc, 2026, 10, 28, 12, 0)
	if d := NextBannerReset(fall, loc).Sub(BannerWeekStart(fall, loc)); d != 169*time.Hour {
		t.Errorf("week spanning fall back should be 169h, got %v", d)
	}
}

func TestBannerRefreshIn(t *testing.T) {
	loc := chicago(t)
	now := at(loc, 2026, 9, 16, 12, 0)
	if got := BannerRefreshIn(now, loc); got != 4*24*time.Hour+22*time.Hour {
		t.Errorf("BannerRefreshIn = %v, want 118h", got)
	}
}

func TestBannerEligible(t *testing.T) {
	stars := func(vals ...int) []RankedWaifu {
		out := make([]RankedWaifu, len(vals))
		for i, v := range vals {
			out[i] = RankedWaifu{Stars: v}
		}
		return out
	}
	cases := map[string]struct {
		chars []RankedWaifu
		want  bool
	}{
		"empty":            {nil, false},
		"too few":          {stars(5, 5, 5, 5), false},
		"enough but dull":  {stars(3, 3, 2, 1, 1), false},
		"enough with four": {stars(4, 1, 1, 1, 1), true},
		"all five":         {stars(5, 5, 5, 5, 5, 5), true},
	}
	for name, c := range cases {
		if got := BannerEligible(c.chars, BannerMinRanked, BannerMinStars); got != c.want {
			t.Errorf("%s: eligible = %v, want %v", name, got, c.want)
		}
	}
}

func TestNewBanner_SortsAndDedupes(t *testing.T) {
	chars := []RankedWaifu{
		{WaifuSummary: WaifuSummary{Slug: "c"}, Position: 30},
		{WaifuSummary: WaifuSummary{Slug: "a"}, Position: 1},
		{WaifuSummary: WaifuSummary{Slug: "c"}, Position: 30},
		{WaifuSummary: WaifuSummary{Slug: "b"}, Position: 7},
	}
	week := time.Date(2026, 9, 14, 15, 0, 0, 0, time.UTC)
	b := NewBanner(week, Series{Slug: "re-zero", Name: "Re:Zero"}, chars)
	if !b.WeekStart.Equal(week) || b.Series.Name != "Re:Zero" {
		t.Errorf("banner header = %+v", b)
	}
	if got := b.Slugs(); len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("slugs = %v", got)
	}
}

func TestBanner_Unowned(t *testing.T) {
	b := NewBanner(time.Time{}, Series{}, []RankedWaifu{
		{WaifuSummary: WaifuSummary{Slug: "a"}, Position: 1},
		{WaifuSummary: WaifuSummary{Slug: "b"}, Position: 2},
		{WaifuSummary: WaifuSummary{Slug: "c"}, Position: 3},
	})
	if got := b.Unowned([]string{"b"}); len(got) != 2 || got[0].Slug != "a" || got[1].Slug != "c" {
		t.Errorf("unowned = %v", got)
	}
	if got := b.Unowned([]string{"a", "b", "c"}); len(got) != 0 {
		t.Errorf("fully owned pool should be empty, got %v", got)
	}
	if got := b.Unowned(nil); len(got) != 3 {
		t.Errorf("nothing owned should keep all, got %d", len(got))
	}
}

func TestRanking_Subset(t *testing.T) {
	var none *Ranking
	if got := none.Subset([]string{"x"}); got != nil {
		t.Errorf("nil ranking subset = %v", got)
	}
	r := BuildRanking([]WaifuSummary{
		{Slug: "low", Likes: 200, Trash: 100},
		{Slug: "top", Likes: 5000, Trash: 10},
		{Slug: "mid", Likes: 1000, Trash: 50},
	}, DefaultMinVotes, time.Time{}, 1)
	got := r.Subset([]string{"mid", "ghost", "top", "mid"})
	if len(got) != 2 || got[0].Slug != "top" || got[1].Slug != "mid" {
		t.Errorf("subset = %v", got)
	}
	if got[0].Position != 1 || got[1].Position != 2 {
		t.Errorf("positions = %d %d", got[0].Position, got[1].Position)
	}
}

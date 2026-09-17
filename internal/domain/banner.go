package domain

import (
	"sort"
	"time"
)

const (
	BannerRollCost     = 400
	BannerMinRanked    = 5
	BannerMinStars     = 4
	BannerCardLimit    = 15
	BannerResetWeekday = time.Monday
	BannerResetHour    = DailyResetHour
)

type Banner struct {
	WeekStart  time.Time
	Series     Series
	Characters []RankedWaifu
}

func BannerWeekStart(now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), BannerResetHour, 0, 0, 0, loc)
	back := (int(local.Weekday()) - int(BannerResetWeekday) + 7) % 7
	start = start.AddDate(0, 0, -back)
	if start.After(local) {
		start = start.AddDate(0, 0, -7)
	}
	return start
}

func NextBannerReset(now time.Time, loc *time.Location) time.Time {
	return BannerWeekStart(now, loc).AddDate(0, 0, 7)
}

func BannerRefreshIn(now time.Time, loc *time.Location) time.Duration {
	return NextBannerReset(now, loc).Sub(now)
}

func BannerEligible(chars []RankedWaifu, minRanked, minStars int) bool {
	if len(chars) < minRanked {
		return false
	}
	for _, c := range chars {
		if c.Stars >= minStars {
			return true
		}
	}
	return false
}

func NewBanner(weekStart time.Time, series Series, chars []RankedWaifu) Banner {
	seen := make(map[string]struct{}, len(chars))
	out := make([]RankedWaifu, 0, len(chars))
	for _, c := range chars {
		if _, dup := seen[c.Slug]; dup {
			continue
		}
		seen[c.Slug] = struct{}{}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return Banner{WeekStart: weekStart, Series: series, Characters: out}
}

func (b Banner) Slugs() []string {
	slugs := make([]string, len(b.Characters))
	for i, c := range b.Characters {
		slugs[i] = c.Slug
	}
	return slugs
}

func (b Banner) Unowned(owned []string) []RankedWaifu {
	skip := make(map[string]struct{}, len(owned))
	for _, s := range owned {
		skip[s] = struct{}{}
	}
	out := make([]RankedWaifu, 0, len(b.Characters))
	for _, c := range b.Characters {
		if _, ok := skip[c.Slug]; !ok {
			out = append(out, c)
		}
	}
	return out
}

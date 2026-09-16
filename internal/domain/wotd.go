package domain

import "time"

const (
	WotdMinStars = 1
	WotdMaxStars = 4
)

func WotdDay(now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func WotdRefreshIn(now time.Time, loc *time.Location) time.Duration {
	return WotdDay(now, loc).AddDate(0, 0, 1).Sub(now)
}

func WotdEligible(r RankedWaifu) bool {
	return r.Stars >= WotdMinStars && r.Stars <= WotdMaxStars
}

package domain

import (
	"fmt"
	"time"
)

const DailyResetHour = 10

func DailyWindowStart(now time.Time, loc *time.Location) time.Time {
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), DailyResetHour, 0, 0, 0, loc)
	if start.After(local) {
		start = start.AddDate(0, 0, -1)
	}
	return start
}

func NextDailyReset(now time.Time, loc *time.Location) time.Time {
	return DailyWindowStart(now, loc).AddDate(0, 0, 1)
}

func DailyClaimAllowed(claimedAt *time.Time, now time.Time, loc *time.Location) bool {
	if claimedAt == nil {
		return true
	}
	return claimedAt.Before(DailyWindowStart(now, loc))
}

func DailyRefreshIn(now time.Time, loc *time.Location) time.Duration {
	return NextDailyReset(now, loc).Sub(now)
}

func FormatCountdown(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Truncate(time.Second).Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

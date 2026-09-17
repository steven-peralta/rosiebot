package app

import (
	"context"
	"time"
)

func (s *RankingService) SetSleepForTest(fn func(context.Context, time.Duration) error) {
	s.sleep = fn
}

func (s *BannerService) SetSleepForTest(fn func(context.Context, time.Duration) error) {
	s.sleep = fn
}

package mwl

import (
	"context"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultRequestsPerMinute = 60
	DefaultHeadroom          = 15
	backgroundPollInterval   = time.Second
)

type Limiter struct {
	rl        *rate.Limiter
	headroom  int
	remaining atomic.Int64
	sleep     func(context.Context, time.Duration) error
}

func NewLimiter(perMinute, headroom int) *Limiter {
	if perMinute <= 0 {
		perMinute = DefaultRequestsPerMinute
	}
	if headroom < 0 {
		headroom = 0
	}
	l := &Limiter{
		rl:       rate.NewLimiter(rate.Every(time.Minute/time.Duration(perMinute)), perMinute),
		headroom: headroom,
		sleep:    sleepCtx,
	}
	l.remaining.Store(int64(perMinute))
	return l
}

func (l *Limiter) Wait(ctx context.Context) error {
	return l.rl.Wait(ctx)
}

func (l *Limiter) WaitBackground(ctx context.Context) error {
	for {
		if l.rl.Tokens() > float64(l.headroom) && l.remaining.Load() > int64(l.headroom) {
			return l.rl.Wait(ctx)
		}
		if err := l.sleep(ctx, backgroundPollInterval); err != nil {
			return err
		}
	}
}

func (l *Limiter) Observe(remaining int) {
	l.remaining.Store(int64(remaining))
}

func (l *Limiter) Remaining() int {
	return int(l.remaining.Load())
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

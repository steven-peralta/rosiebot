package mwl

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	DefaultRequestsPerMinute = 60
	DefaultHeadroom          = 20
	DefaultBurst             = 5
	backgroundPollInterval   = time.Second
	serverWindow             = time.Minute
)

type Limiter struct {
	rl         *rate.Limiter
	perMinute  int
	headroom   int
	sleep      func(context.Context, time.Duration) error
	now        func() time.Time
	mu         sync.Mutex
	remaining  int
	observedAt time.Time
}

func NewLimiter(perMinute, headroom int) *Limiter {
	if perMinute <= 0 {
		perMinute = DefaultRequestsPerMinute
	}
	if headroom < 0 {
		headroom = 0
	}
	l := &Limiter{
		rl:        rate.NewLimiter(rate.Every(time.Minute/time.Duration(perMinute)), min(DefaultBurst, perMinute)),
		perMinute: perMinute,
		headroom:  headroom,
		sleep:     sleepCtx,
		now:       time.Now,
	}
	l.remaining = perMinute
	l.observedAt = l.now()
	return l
}

func (l *Limiter) Wait(ctx context.Context) error {
	return l.rl.Wait(ctx)
}

func (l *Limiter) WaitBackground(ctx context.Context) error {
	for {
		if l.Remaining() > l.headroom && l.rl.Tokens() >= 1 {
			return l.rl.Wait(ctx)
		}
		if err := l.sleep(ctx, backgroundPollInterval); err != nil {
			return err
		}
	}
}

func (l *Limiter) Observe(remaining int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.remaining = max(remaining, 0)
	l.observedAt = l.now()
}

func (l *Limiter) Penalize() {
	l.Observe(0)
}

func (l *Limiter) Remaining() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.now().Sub(l.observedAt) >= serverWindow {
		return l.perMinute
	}
	return l.remaining
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

package memory

import (
	"context"
	"sync"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type BannerStore struct {
	mu    sync.Mutex
	weeks map[int64]domain.Banner
}

var _ app.BannerStore = (*BannerStore)(nil)

func NewBannerStore() *BannerStore {
	return &BannerStore{weeks: map[int64]domain.Banner{}}
}

func weekKey(weekStart time.Time) int64 { return weekStart.Unix() }

func (s *BannerStore) Get(ctx context.Context, weekStart time.Time) (domain.Banner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.weeks[weekKey(weekStart)]
	if !ok {
		return domain.Banner{}, app.ErrNotFound
	}
	return b, nil
}

func (s *BannerStore) Put(ctx context.Context, b domain.Banner) (domain.Banner, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.weeks[weekKey(b.WeekStart)]; ok {
		return existing, nil
	}
	s.weeks[weekKey(b.WeekStart)] = b
	return b, nil
}

func (s *BannerStore) Replace(ctx context.Context, b domain.Banner) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.weeks[weekKey(b.WeekStart)] = b
	return nil
}

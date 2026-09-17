package memory

import (
	"context"
	"sync"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type DailyStore struct {
	mu   sync.Mutex
	days map[string]domain.WaifuSummary
}

var _ app.DailyStore = (*DailyStore)(nil)

func NewDailyStore() *DailyStore {
	return &DailyStore{days: map[string]domain.WaifuSummary{}}
}

func dayKey(day time.Time) string { return day.Format("2006-01-02") }

func (s *DailyStore) Get(ctx context.Context, day time.Time) (domain.WaifuSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	w, ok := s.days[dayKey(day)]
	if !ok {
		return domain.WaifuSummary{}, app.ErrNotFound
	}
	return w, nil
}

func (s *DailyStore) Put(ctx context.Context, day time.Time, w domain.WaifuSummary) (domain.WaifuSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.days[dayKey(day)]; ok {
		return existing, nil
	}
	s.days[dayKey(day)] = w
	return w, nil
}

func (s *DailyStore) Replace(ctx context.Context, day time.Time, w domain.WaifuSummary) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.days[dayKey(day)] = w
	return nil
}

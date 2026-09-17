package memory

import (
	"context"
	"sync"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type AlertStore struct {
	mu       sync.Mutex
	disabled map[domain.PlayerKey]bool
	closed   map[string]time.Time
	sent     map[string]time.Time
}

var _ app.AlertStore = (*AlertStore)(nil)

func NewAlertStore() *AlertStore {
	return &AlertStore{disabled: map[domain.PlayerKey]bool{}, closed: map[string]time.Time{}, sent: map[string]time.Time{}}
}

func (s *AlertStore) Setting(ctx context.Context, key domain.PlayerKey) (app.AlertSetting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, closed := s.closed[key.UserID]
	return app.AlertSetting{Enabled: !s.disabled[key], DMClosed: closed}, nil
}

func (s *AlertStore) SetEnabled(ctx context.Context, key domain.PlayerKey, enabled bool, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if enabled {
		delete(s.disabled, key)
	} else {
		s.disabled[key] = true
	}
	return nil
}

func (s *AlertStore) MarkDMClosed(ctx context.Context, userID string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed[userID] = now
	return nil
}

func (s *AlertStore) ClearDMClosed(ctx context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.closed, userID)
	return nil
}

func (s *AlertStore) MarkSent(ctx context.Context, eventID, userID string, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := eventID + "|" + userID
	if _, ok := s.sent[k]; ok {
		return false, nil
	}
	s.sent[k] = now
	return true, nil
}

func (s *AlertStore) SentCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sent)
}

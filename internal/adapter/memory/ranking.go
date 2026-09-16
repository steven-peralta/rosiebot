package memory

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type RankingStore struct {
	mu     sync.Mutex
	latest *domain.Ranking
}

var _ app.RankingStore = (*RankingStore)(nil)

func NewRankingStore() *RankingStore { return &RankingStore{} }

func (s *RankingStore) LoadLatest(ctx context.Context) (*domain.Ranking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.latest == nil {
		return nil, app.ErrNotFound
	}
	return s.latest, nil
}

func (s *RankingStore) Save(ctx context.Context, r *domain.Ranking) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = r
	return nil
}

type RankingHolder struct {
	current atomic.Pointer[domain.Ranking]
}

var _ app.RankingProvider = (*RankingHolder)(nil)

func NewRankingHolder(r *domain.Ranking) *RankingHolder {
	h := &RankingHolder{}
	h.Set(r)
	return h
}

func (h *RankingHolder) Set(r *domain.Ranking) { h.current.Store(r) }

func (h *RankingHolder) Current() *domain.Ranking { return h.current.Load() }

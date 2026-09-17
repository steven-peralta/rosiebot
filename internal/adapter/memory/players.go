package memory

import (
	"context"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type PlayerStore struct {
	mu        sync.Mutex
	clock     app.Clock
	players   map[domain.PlayerKey]domain.Player
	inventory map[domain.PlayerKey]map[string]domain.OwnedWaifu
}

var _ app.PlayerStore = (*PlayerStore)(nil)

func NewPlayerStore(clock app.Clock) *PlayerStore {
	if clock == nil {
		clock = app.SystemClock()
	}
	return &PlayerStore{
		clock:     clock,
		players:   map[domain.PlayerKey]domain.Player{},
		inventory: map[domain.PlayerKey]map[string]domain.OwnedWaifu{},
	}
}

type snapshot struct {
	players   map[domain.PlayerKey]domain.Player
	inventory map[domain.PlayerKey]map[string]domain.OwnedWaifu
}

func (s *PlayerStore) snapshot() snapshot {
	inv := make(map[domain.PlayerKey]map[string]domain.OwnedWaifu, len(s.inventory))
	for k, v := range s.inventory {
		inv[k] = maps.Clone(v)
	}
	return snapshot{players: maps.Clone(s.players), inventory: inv}
}

func (s *PlayerStore) restore(snap snapshot) {
	s.players = snap.players
	s.inventory = snap.inventory
}

func (s *PlayerStore) WithinTx(ctx context.Context, fn func(app.PlayerRepo) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := s.snapshot()
	if err := fn(txRepo{s}); err != nil {
		s.restore(snap)
		return err
	}
	return nil
}

func (s *PlayerStore) EnsurePlayer(ctx context.Context, key domain.PlayerKey) (domain.Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ensurePlayerLocked(key)
}

func (s *PlayerStore) GetPlayer(ctx context.Context, key domain.PlayerKey) (domain.Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getPlayerLocked(key)
}

func (s *PlayerStore) LockPlayers(ctx context.Context, keys ...domain.PlayerKey) ([]domain.Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lockPlayersLocked(keys...)
}

func (s *PlayerStore) DebitCoins(ctx context.Context, key domain.PlayerKey, amount int64) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.debitCoinsLocked(key, amount)
}

func (s *PlayerStore) SetCoins(ctx context.Context, key domain.PlayerKey, coins int64) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setCoinsLocked(key, coins)
}

func (s *PlayerStore) AdjustCoins(ctx context.Context, key domain.PlayerKey, delta int64) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.adjustCoinsLocked(key, delta)
}

func (s *PlayerStore) ClaimDaily(ctx context.Context, key domain.PlayerKey, amount int64, windowStart, now time.Time) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.claimDailyLocked(key, amount, windowStart, now)
}

func (s *PlayerStore) AddOwned(ctx context.Context, key domain.PlayerKey, w domain.OwnedWaifu) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addOwnedLocked(key, w)
}

func (s *PlayerStore) GetOwned(ctx context.Context, key domain.PlayerKey, slug string) (domain.OwnedWaifu, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getOwnedLocked(key, slug)
}

func (s *PlayerStore) SellOwned(ctx context.Context, key domain.PlayerKey, slug string, price int64) (int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sellOwnedLocked(key, slug, price)
}

func (s *PlayerStore) OwnedSlugs(ctx context.Context, key domain.PlayerKey, slugs []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ownedSlugsLocked(key, slugs)
}

func (s *PlayerStore) TransferOwned(ctx context.Context, from, to domain.PlayerKey, slugs []string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transferOwnedLocked(from, to, slugs)
}

func (s *PlayerStore) ListOwned(ctx context.Context, key domain.PlayerKey, prefix string, limit int) ([]domain.OwnedWaifu, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listOwnedLocked(key, prefix, limit)
}

func (s *PlayerStore) CountOwned(ctx context.Context, key domain.PlayerKey) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.inventory[key]), nil
}

func (s *PlayerStore) ensurePlayerLocked(key domain.PlayerKey) (domain.Player, error) {
	if p, ok := s.players[key]; ok {
		return p, nil
	}
	p := domain.NewPlayer(key, s.clock.Now())
	s.players[key] = p
	return p, nil
}

func (s *PlayerStore) getPlayerLocked(key domain.PlayerKey) (domain.Player, error) {
	p, ok := s.players[key]
	if !ok {
		return domain.Player{}, app.ErrNotFound
	}
	return p, nil
}

func (s *PlayerStore) lockPlayersLocked(keys ...domain.PlayerKey) ([]domain.Player, error) {
	sorted := slices.Clone(keys)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].UserID < sorted[j].UserID })
	out := make([]domain.Player, 0, len(sorted))
	for _, key := range sorted {
		p, err := s.getPlayerLocked(key)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *PlayerStore) debitCoinsLocked(key domain.PlayerKey, amount int64) (int64, bool, error) {
	p, ok := s.players[key]
	if !ok || p.Coins < amount {
		return 0, false, nil
	}
	p.Coins -= amount
	p.UpdatedAt = s.clock.Now()
	s.players[key] = p
	return p.Coins, true, nil
}

func (s *PlayerStore) setCoinsLocked(key domain.PlayerKey, coins int64) (int64, error) {
	p, ok := s.players[key]
	if !ok {
		return 0, app.ErrNotFound
	}
	p.Coins = coins
	p.UpdatedAt = s.clock.Now()
	s.players[key] = p
	return p.Coins, nil
}

func (s *PlayerStore) adjustCoinsLocked(key domain.PlayerKey, delta int64) (int64, bool, error) {
	p, ok := s.players[key]
	if !ok || p.Coins+delta < 0 {
		return 0, false, nil
	}
	p.Coins += delta
	p.UpdatedAt = s.clock.Now()
	s.players[key] = p
	return p.Coins, true, nil
}

func (s *PlayerStore) claimDailyLocked(key domain.PlayerKey, amount int64, windowStart, now time.Time) (int64, bool, error) {
	p, ok := s.players[key]
	if !ok {
		return 0, false, nil
	}
	if p.DailyClaimedAt != nil && !p.DailyClaimedAt.Before(windowStart) {
		return 0, false, nil
	}
	p.Coins += amount
	claimed := now
	p.DailyClaimedAt = &claimed
	p.UpdatedAt = now
	s.players[key] = p
	return p.Coins, true, nil
}

func (s *PlayerStore) addOwnedLocked(key domain.PlayerKey, w domain.OwnedWaifu) (bool, error) {
	if _, ok := s.players[key]; !ok {
		return false, app.ErrNotFound
	}
	inv := s.inventory[key]
	if inv == nil {
		inv = map[string]domain.OwnedWaifu{}
		s.inventory[key] = inv
	}
	if _, exists := inv[w.Slug]; exists {
		return false, nil
	}
	inv[w.Slug] = w
	return true, nil
}

func (s *PlayerStore) getOwnedLocked(key domain.PlayerKey, slug string) (domain.OwnedWaifu, error) {
	w, ok := s.inventory[key][slug]
	if !ok {
		return domain.OwnedWaifu{}, app.ErrNotFound
	}
	return w, nil
}

func (s *PlayerStore) sellOwnedLocked(key domain.PlayerKey, slug string, price int64) (int64, bool, error) {
	inv := s.inventory[key]
	if _, ok := inv[slug]; !ok {
		return 0, false, nil
	}
	p, ok := s.players[key]
	if !ok {
		return 0, false, nil
	}
	delete(inv, slug)
	p.Coins += price
	p.UpdatedAt = s.clock.Now()
	s.players[key] = p
	return p.Coins, true, nil
}

func (s *PlayerStore) ownedSlugsLocked(key domain.PlayerKey, slugs []string) ([]string, error) {
	inv := s.inventory[key]
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		if _, ok := inv[slug]; ok {
			out = append(out, slug)
		}
	}
	return out, nil
}

func (s *PlayerStore) transferOwnedLocked(from, to domain.PlayerKey, slugs []string) (int64, error) {
	src := s.inventory[from]
	dst := s.inventory[to]
	if dst == nil {
		dst = map[string]domain.OwnedWaifu{}
		s.inventory[to] = dst
	}
	var moved int64
	for _, slug := range slugs {
		w, ok := src[slug]
		if !ok {
			continue
		}
		if _, dup := dst[slug]; dup {
			return moved, app.ErrAlreadyOwned
		}
		delete(src, slug)
		dst[slug] = w
		moved++
	}
	return moved, nil
}

func (s *PlayerStore) listOwnedLocked(key domain.PlayerKey, prefix string, limit int) ([]domain.OwnedWaifu, error) {
	prefix = strings.ToLower(prefix)
	out := make([]domain.OwnedWaifu, 0, len(s.inventory[key]))
	for _, w := range s.inventory[key] {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(w.Name), prefix) {
			continue
		}
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].AcquiredAt.Equal(out[j].AcquiredAt) {
			return out[i].AcquiredAt.Before(out[j].AcquiredAt)
		}
		return out[i].Slug < out[j].Slug
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type txRepo struct {
	s *PlayerStore
}

func (t txRepo) EnsurePlayer(_ context.Context, key domain.PlayerKey) (domain.Player, error) {
	return t.s.ensurePlayerLocked(key)
}

func (t txRepo) GetPlayer(_ context.Context, key domain.PlayerKey) (domain.Player, error) {
	return t.s.getPlayerLocked(key)
}

func (t txRepo) LockPlayers(_ context.Context, keys ...domain.PlayerKey) ([]domain.Player, error) {
	return t.s.lockPlayersLocked(keys...)
}

func (t txRepo) DebitCoins(_ context.Context, key domain.PlayerKey, amount int64) (int64, bool, error) {
	return t.s.debitCoinsLocked(key, amount)
}

func (t txRepo) SetCoins(_ context.Context, key domain.PlayerKey, coins int64) (int64, error) {
	return t.s.setCoinsLocked(key, coins)
}

func (t txRepo) AdjustCoins(_ context.Context, key domain.PlayerKey, delta int64) (int64, bool, error) {
	return t.s.adjustCoinsLocked(key, delta)
}

func (t txRepo) ClaimDaily(_ context.Context, key domain.PlayerKey, amount int64, windowStart, now time.Time) (int64, bool, error) {
	return t.s.claimDailyLocked(key, amount, windowStart, now)
}

func (t txRepo) AddOwned(_ context.Context, key domain.PlayerKey, w domain.OwnedWaifu) (bool, error) {
	return t.s.addOwnedLocked(key, w)
}

func (t txRepo) GetOwned(_ context.Context, key domain.PlayerKey, slug string) (domain.OwnedWaifu, error) {
	return t.s.getOwnedLocked(key, slug)
}

func (t txRepo) SellOwned(_ context.Context, key domain.PlayerKey, slug string, price int64) (int64, bool, error) {
	return t.s.sellOwnedLocked(key, slug, price)
}

func (t txRepo) OwnedSlugs(_ context.Context, key domain.PlayerKey, slugs []string) ([]string, error) {
	return t.s.ownedSlugsLocked(key, slugs)
}

func (t txRepo) TransferOwned(_ context.Context, from, to domain.PlayerKey, slugs []string) (int64, error) {
	return t.s.transferOwnedLocked(from, to, slugs)
}

func (t txRepo) ListOwned(_ context.Context, key domain.PlayerKey, prefix string, limit int) ([]domain.OwnedWaifu, error) {
	return t.s.listOwnedLocked(key, prefix, limit)
}

func (t txRepo) CountOwned(_ context.Context, key domain.PlayerKey) (int, error) {
	return len(t.s.inventory[key]), nil
}

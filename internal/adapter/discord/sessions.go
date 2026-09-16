package discord

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type SessionKind int

const (
	SessionPager SessionKind = iota
	SessionTrade
)

type tradeState int32

const (
	tradePending tradeState = iota
	tradeAccepting
	tradeDone
)

type pageRef struct {
	summary domain.WaifuSummary
	detail  *domain.Waifu
}

type Session struct {
	Kind      SessionKind
	OwnerID   string
	ChannelID string
	MessageID string
	Content   string
	Pages     []pageRef
	Page      int
	Sellable  bool
	Offer     domain.TradeOffer
	Give      []domain.OwnedWaifu
	Receive   []domain.OwnedWaifu
	SenderID  string
	TargetID  string
	ExpiresAt time.Time
	ttl       time.Duration
	state     atomic.Int32
}

func (s *Session) touch(now time.Time) {
	if s.Kind == SessionPager {
		s.ExpiresAt = now.Add(s.ttl)
	}
}

func (s *Session) beginAccept() bool {
	return s.state.CompareAndSwap(int32(tradePending), int32(tradeAccepting))
}

func (s *Session) finish() {
	s.state.Store(int32(tradeDone))
}

func (s *Session) reopen() {
	s.state.Store(int32(tradePending))
}

type SessionStore struct {
	mu       sync.Mutex
	clock    app.Clock
	sessions map[string]*Session
}

func NewSessionStore(clock app.Clock) *SessionStore {
	return &SessionStore{clock: clock, sessions: map[string]*Session{}}
}

func (st *SessionStore) Put(messageID string, s *Session, ttl time.Duration) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s.MessageID = messageID
	s.ttl = ttl
	s.ExpiresAt = st.clock.Now().Add(ttl)
	st.sessions[messageID] = s
}

func (st *SessionStore) Get(messageID string) (*Session, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s, ok := st.sessions[messageID]
	if !ok {
		return nil, false
	}
	if !st.clock.Now().Before(s.ExpiresAt) {
		delete(st.sessions, messageID)
		return nil, false
	}
	s.touch(st.clock.Now())
	return s, true
}

func (st *SessionStore) Delete(messageID string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.sessions, messageID)
}

func (st *SessionStore) Len() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.sessions)
}

func (st *SessionStore) Expired() []*Session {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := st.clock.Now()
	var out []*Session
	for id, s := range st.sessions {
		if !now.Before(s.ExpiresAt) {
			out = append(out, s)
			delete(st.sessions, id)
		}
	}
	return out
}

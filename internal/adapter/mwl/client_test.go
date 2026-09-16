package mwl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/app"
)

type recorded struct {
	path   string
	query  string
	header http.Header
}

type server struct {
	*httptest.Server
	mu       sync.Mutex
	requests []recorded
	limited  int
}

func (s *server) record(r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, recorded{path: r.URL.Path, query: r.URL.RawQuery, header: r.Header.Clone()})
}

func (s *server) count(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, r := range s.requests {
		if r.path == path {
			n++
		}
	}
	return n
}

func (s *server) last() recorded {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[len(s.requests)-1]
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func serveFile(t *testing.T, name string) http.HandlerFunc {
	body := fixture(t, name)
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-ratelimit-remaining", "42")
		_, _ = w.Write(body)
	}
}

func serveJSON(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	}
}

func newServer(t *testing.T) *server {
	t.Helper()
	s := &server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/character/rem", serveFile(t, "character_rem.json"))
	mux.HandleFunc("/api/v1/meta/random", serveFile(t, "meta_random.json"))
	mux.HandleFunc("/api/v1/meta/daily", serveFile(t, "meta_daily.json"))
	mux.HandleFunc("/api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		var env struct {
			Data []json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(fixture(t, "search_waifus.json"), &env); err != nil {
			t.Fatal(err)
		}
		series, _ := json.Marshal(map[string]any{"id": 9, "uuid": "s", "slug": "trigun-series", "name": "Trigun", "url": "https://www.mywaifulist.moe/series/trigun", "studio": nil})
		env.Data = append(env.Data, series)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(env)
	})
	mux.HandleFunc("/api/v1/character", func(w http.ResponseWriter, r *http.Request) {
		page := pageNum(r.URL.Query().Get("page"))
		serveJSON(map[string]any{"data": []map[string]any{{"id": page, "slug": fmt.Sprintf("list-%d", page), "name": "L", "likes": 1, "trash": 0}}, "meta": map[string]any{"current_page": page, "last_page": 5113, "per_page": 10, "total": 51128}})(w, r)
	})
	mux.HandleFunc("/api/v1/search/works", serveJSON(map[string]any{"data": []map[string]any{
		{"uuid": "w-1", "slug": "re-zero", "name": "Re:Zero", "url": "https://www.mywaifulist.moe/series/re-zero", "display_picture": nil, "description": "Subaru suffers"},
		{"uuid": nil, "slug": "other", "name": "Other", "url": "https://www.mywaifulist.moe/series/other"},
	}}))
	mux.HandleFunc("/api/v1/work/re-zero/characters", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		rows := []map[string]any{{"uuid": "c1", "slug": "emilia", "name": "Emilia", "likes": 300, "trash": 10}}
		if page == "2" {
			rows = []map[string]any{{"uuid": nil, "slug": "rem", "name": "Rem", "likes": 900, "trash": 30, "display_picture": nil}}
		}
		serveJSON(map[string]any{"data": rows, "meta": map[string]any{"current_page": pageNum(page), "last_page": 2}})(w, r)
	})
	mux.HandleFunc("/api/v1/ranking/popular", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1000" {
			serveFile(t, "ranking_popular_1000.json")(w, r)
			return
		}
		serveJSON(map[string]any{"0": map[string]any{"slug": "top", "name": "Top", "likes": 9000, "trash": 1}, "meta": map[string]any{"current_page": 1, "last_page": 5113}})(w, r)
	})
	mux.HandleFunc("/api/v1/limited", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.limited++
		n := s.limited
		s.mu.Unlock()
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		serveJSON(map[string]any{"data": map[string]any{"ok": true}})(w, r)
	})
	mux.HandleFunc("/api/v1/always-limited", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	})
	mux.HandleFunc("/api/v1/unauthorized", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/api/v1/broken", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("/api/v1/garbage", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r)
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func pageNum(s string) int {
	if s == "" {
		return 1
	}
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

func newClient(t *testing.T, s *server) *Client {
	t.Helper()
	previous := defaultRetryWait
	defaultRetryWait = 10 * time.Millisecond
	t.Cleanup(func() { defaultRetryWait = previous })
	c, err := New(Config{BaseURL: s.URL + "/api/v1/", APIKey: "test-key", RequestsPerMinute: 6000, Headroom: 0})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNew_Validation(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("missing api key should fail")
	}
	if _, err := New(Config{APIKey: "k", BaseURL: "://bad"}); err == nil {
		t.Error("bad base url should fail")
	}
	c, err := New(Config{APIKey: "k"})
	if err != nil || c.base.String() != DefaultBaseURL || c.Limiter() == nil {
		t.Errorf("defaults: %v %v", c, err)
	}
}

func TestClient_SendsApikeyAndUwu(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	ctx := context.Background()

	if _, err := c.Random(ctx); err != nil {
		t.Fatal(err)
	}
	genReq := s.last()
	if genReq.header.Get("apikey") != "test-key" || genReq.header.Get("uwu") != "owo" {
		t.Errorf("generated path headers = %v", genReq.header)
	}

	if _, err := c.Get(ctx, "rem"); err != nil {
		t.Fatal(err)
	}
	rawReq := s.last()
	if rawReq.header.Get("apikey") != "test-key" || rawReq.header.Get("uwu") != "owo" || rawReq.header.Get("Accept") != "application/json" {
		t.Errorf("raw path headers = %v", rawReq.header)
	}
	if c.Limiter().Remaining() != 42 {
		t.Errorf("remaining header not observed: %d", c.Limiter().Remaining())
	}
}

func TestClient_RetriesOn429WithRetryAfter(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	var out map[string]any
	if err := c.getJSON(context.Background(), "limited", nil, &out); err != nil {
		t.Fatal(err)
	}
	if s.count("/api/v1/limited") != 2 {
		t.Errorf("expected exactly one retry, saw %d requests", s.count("/api/v1/limited"))
	}
}

func TestClient_StatusMapping(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	ctx := context.Background()
	var out map[string]any

	if err := c.getJSON(ctx, "character/missing", nil, &out); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("404 -> %v", err)
	}
	if err := c.getJSON(ctx, "unauthorized", nil, &out); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("401 -> %v", err)
	}
	if err := c.getJSON(ctx, "always-limited", nil, &out); !errors.Is(err, ErrRateLimited) {
		t.Errorf("429 -> %v", err)
	}
	var status *StatusError
	if err := c.getJSON(ctx, "broken", nil, &out); !errors.As(err, &status) || status.Status != 500 || !strings.Contains(status.Error(), "500") {
		t.Errorf("500 -> %v", err)
	}
	if err := c.getJSON(ctx, "garbage", nil, &out); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("garbage -> %v", err)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Get(ctx, "rem"); err == nil {
		t.Error("cancelled context should fail")
	}
}

func TestRetryAfter(t *testing.T) {
	resp := func(v string) *http.Response {
		h := http.Header{}
		if v != "" {
			h.Set("Retry-After", v)
		}
		return &http.Response{Header: h}
	}
	if got := retryAfter(resp("")); got != defaultRetryWait {
		t.Errorf("missing = %v", got)
	}
	if got := retryAfter(resp("3")); got != defaultRetryWait {
		t.Errorf("seconds below the floor = %v", got)
	}
	if got := retryAfter(resp("10")); got != 10*time.Second {
		t.Errorf("seconds = %v", got)
	}
	if got := retryAfter(resp("3600")); got != maxRetryAfter {
		t.Errorf("cap = %v", got)
	}
	if got := retryAfter(resp("nonsense")); got != defaultRetryWait {
		t.Errorf("garbage = %v", got)
	}
	future := time.Now().Add(5 * time.Second).UTC().Format(http.TimeFormat)
	if got := retryAfter(resp(future)); got <= 0 || got > 6*time.Second {
		t.Errorf("http date = %v", got)
	}
	past := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	if got := retryAfter(resp(past)); got != defaultRetryWait {
		t.Errorf("past date = %v", got)
	}
}

func TestLimiter_BackgroundYieldsToForeground(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	l := NewLimiter(60, 20)
	l.now = func() time.Time { return now }
	l.Observe(60)
	ctx := context.Background()

	if l.Remaining() != 60 {
		t.Fatalf("fresh remaining = %d", l.Remaining())
	}
	for range DefaultBurst {
		if err := l.Wait(ctx); err != nil {
			t.Fatal(err)
		}
	}
	l.Observe(10)
	polls := 0
	l.sleep = func(context.Context, time.Duration) error {
		polls++
		if polls >= 3 {
			return context.Canceled
		}
		return nil
	}
	if err := l.WaitBackground(ctx); !errors.Is(err, context.Canceled) || polls != 3 {
		t.Fatalf("background should poll while the server window is nearly spent: err=%v polls=%d", err, polls)
	}

	now = now.Add(30 * time.Second)
	if got := l.Remaining(); got != 10 {
		t.Errorf("remaining must not refill inside the server window: %d", got)
	}
	l.Penalize()
	if got := l.Remaining(); got != 0 {
		t.Errorf("penalized remaining = %d", got)
	}
	now = now.Add(serverWindow)
	if got := l.Remaining(); got != 60 {
		t.Errorf("remaining should reset once the window has passed: %d", got)
	}
	reset := NewLimiter(60, 20)
	reset.now = func() time.Time { return now }
	reset.Penalize()
	now = now.Add(serverWindow)
	reset.sleep = func(context.Context, time.Duration) error {
		t.Fatal("should not sleep after the window reset")
		return nil
	}
	if err := reset.WaitBackground(ctx); err != nil {
		t.Fatal(err)
	}

	fresh := NewLimiter(60, 20)
	fresh.sleep = func(context.Context, time.Duration) error { t.Fatal("should not sleep"); return nil }
	if err := fresh.WaitBackground(ctx); err != nil {
		t.Fatal(err)
	}

	if NewLimiter(0, -1).headroom != 0 || NewLimiter(0, -1).Remaining() != DefaultRequestsPerMinute {
		t.Error("defaults not applied")
	}
}

func TestSleepCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepCtx(ctx, time.Minute); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
	if err := sleepCtx(context.Background(), time.Millisecond); err != nil {
		t.Errorf("err = %v", err)
	}
}

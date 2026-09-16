package mwl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/steven-peralta/rosiebot/internal/adapter/mwl/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
)

const (
	DefaultBaseURL  = "https://mywaifulist.moe/api/v1"
	handshakeHeader = "uwu"
	handshakeValue  = "owo"
	apiKeyHeader    = "apikey"
	remainingHeader = "x-ratelimit-remaining"
	maxRetryAfter   = 60 * time.Second
	maxBodyBytes    = 4 << 20
)

var (
	ErrUnauthorized  = errors.New("mwl: api key rejected")
	ErrRateLimited   = errors.New("mwl: rate limited")
	defaultRetryWait = 5 * time.Second
)

type StatusError struct {
	Status int
	Path   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("mwl: %s returned %d", e.Path, e.Status)
}

type Config struct {
	BaseURL           string
	APIKey            string
	HTTPClient        *http.Client
	RequestsPerMinute int
	Headroom          int
	Logger            *slog.Logger
}

type Client struct {
	base    *url.URL
	http    *http.Client
	gen     *gen.Client
	limiter *Limiter
	log     *slog.Logger
}

func New(cfg Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("mwl: api key is required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	base, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("mwl: parse base url: %w", err)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	limiter := NewLimiter(cfg.RequestsPerMinute, cfg.Headroom)

	transport := &transport{
		next:    httpClient.Transport,
		apiKey:  cfg.APIKey,
		limiter: limiter,
		log:     cfg.Logger,
	}
	if transport.next == nil {
		transport.next = http.DefaultTransport
	}
	wrapped := *httpClient
	wrapped.Transport = transport

	genClient, err := gen.NewClient(base.String(), security{apiKey: cfg.APIKey}, gen.WithClient(&wrapped))
	if err != nil {
		return nil, fmt.Errorf("mwl: build generated client: %w", err)
	}
	return &Client{base: base, http: &wrapped, gen: genClient, limiter: limiter, log: cfg.Logger}, nil
}

func (c *Client) Limiter() *Limiter { return c.limiter }

type security struct {
	apiKey string
}

func (s security) ApiKey(context.Context, gen.OperationName) (gen.ApiKey, error) {
	return gen.ApiKey{APIKey: s.apiKey}, nil
}

func (s security) Uwu(context.Context, gen.OperationName) (gen.Uwu, error) {
	return gen.Uwu{APIKey: handshakeValue}, nil
}

type transport struct {
	next    http.RoundTripper
	apiKey  string
	limiter *Limiter
	log     *slog.Logger
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if err := t.acquire(ctx); err != nil {
		return nil, err
	}
	resp, err := t.send(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		return resp, nil
	}
	t.limiter.Penalize()
	wait := retryAfter(resp)
	drain(resp)
	t.log.Warn("mwl rate limited, retrying once", "path", req.URL.Path, "wait", wait)
	if err := sleepCtx(ctx, wait); err != nil {
		return nil, err
	}
	if err := t.acquire(ctx); err != nil {
		return nil, err
	}
	return t.send(req)
}

func (t *transport) acquire(ctx context.Context) error {
	if app.IsBackground(ctx) {
		return t.limiter.WaitBackground(ctx)
	}
	return t.limiter.Wait(ctx)
}

func (t *transport) send(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set(apiKeyHeader, t.apiKey)
	clone.Header.Set(handshakeHeader, handshakeValue)
	if clone.Header.Get("Accept") == "" {
		clone.Header.Set("Accept", "application/json")
	}
	resp, err := t.next.RoundTrip(clone)
	if err != nil {
		return nil, err
	}
	if v := resp.Header.Get(remainingHeader); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			t.limiter.Observe(n)
		}
	}
	return resp, nil
}

func retryAfter(resp *http.Response) time.Duration {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return defaultRetryWait
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return min(max(time.Duration(secs)*time.Second, defaultRetryWait), maxRetryAfter)
	}
	if at, err := http.ParseTime(v); err == nil {
		return min(max(time.Until(at), defaultRetryWait), maxRetryAfter)
	}
	return defaultRetryWait
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyBytes))
	_ = resp.Body.Close()
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	u := *c.base
	u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("mwl: build request %s: %w", path, err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("mwl: GET %s: %w", path, err)
	}
	defer drain(resp)

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return app.ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrUnauthorized
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return &StatusError{Status: resp.StatusCode, Path: path}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return fmt.Errorf("mwl: read %s: %w", path, err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("mwl: decode %s: %w", path, err)
	}
	return nil
}

package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/steven-peralta/rosiebot/internal/adapter/postgres/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type AlertStore struct {
	q *gen.Queries
}

var _ app.AlertStore = (*AlertStore)(nil)

func NewAlertStore(pool *pgxpool.Pool) *AlertStore {
	return &AlertStore{q: gen.New(pool)}
}

func (s *AlertStore) Setting(ctx context.Context, key domain.PlayerKey) (app.AlertSetting, error) {
	st := app.AlertSetting{Enabled: true}
	enabled, err := s.q.GetAlertEnabled(ctx, gen.GetAlertEnabledParams{GuildID: key.GuildID, UserID: key.UserID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return app.AlertSetting{}, fmt.Errorf("postgres: alert setting: %w", err)
	default:
		st.Enabled = enabled
	}
	closed, err := s.q.IsDMClosed(ctx, key.UserID)
	if err != nil {
		return app.AlertSetting{}, fmt.Errorf("postgres: dm closed: %w", err)
	}
	st.DMClosed = closed
	return st, nil
}

func (s *AlertStore) SetEnabled(ctx context.Context, key domain.PlayerKey, enabled bool, now time.Time) error {
	if err := s.q.SetAlertEnabled(ctx, gen.SetAlertEnabledParams{GuildID: key.GuildID, UserID: key.UserID, Enabled: enabled, UpdatedAt: now}); err != nil {
		return fmt.Errorf("postgres: set alerts: %w", err)
	}
	return nil
}

func (s *AlertStore) MarkDMClosed(ctx context.Context, userID string, now time.Time) error {
	if err := s.q.MarkDMClosed(ctx, gen.MarkDMClosedParams{UserID: userID, ClosedAt: now}); err != nil {
		return fmt.Errorf("postgres: mark dm closed: %w", err)
	}
	return nil
}

func (s *AlertStore) ClearDMClosed(ctx context.Context, userID string) error {
	if err := s.q.ClearDMClosed(ctx, userID); err != nil {
		return fmt.Errorf("postgres: clear dm closed: %w", err)
	}
	return nil
}

func (s *AlertStore) MarkSent(ctx context.Context, eventID, userID string, now time.Time) (bool, error) {
	n, err := s.q.MarkAlertSent(ctx, gen.MarkAlertSentParams{EventID: eventID, UserID: userID, SentAt: now})
	if err != nil {
		return false, fmt.Errorf("postgres: mark alert sent: %w", err)
	}
	return n > 0, nil
}

func (s *AlertStore) Prune(ctx context.Context, before time.Time) (int64, error) {
	n, err := s.q.PruneAlertsSent(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("postgres: prune alerts: %w", err)
	}
	return n, nil
}

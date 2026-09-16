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

type DailyStore struct {
	q *gen.Queries
}

var _ app.DailyStore = (*DailyStore)(nil)

func NewDailyStore(pool *pgxpool.Pool) *DailyStore {
	return &DailyStore{q: gen.New(pool)}
}

func (s *DailyStore) Get(ctx context.Context, day time.Time) (domain.WaifuSummary, error) {
	row, err := s.q.GetDailyWaifu(ctx, dateOnly(day))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.WaifuSummary{}, app.ErrNotFound
	}
	if err != nil {
		return domain.WaifuSummary{}, fmt.Errorf("postgres: get daily waifu: %w", err)
	}
	return toSummary(row), nil
}

func (s *DailyStore) Put(ctx context.Context, day time.Time, w domain.WaifuSummary) (domain.WaifuSummary, error) {
	_, err := s.q.PutDailyWaifu(ctx, gen.PutDailyWaifuParams{
		Day:          dateOnly(day),
		Slug:         w.Slug,
		Uuid:         w.UUID,
		Name:         w.Name,
		OriginalName: w.OriginalName,
		RomajiName:   w.RomajiName,
		PictureUrl:   w.PictureURL,
		Likes:        int32(w.Likes),
		Trash:        int32(w.Trash),
	})
	if err != nil {
		return domain.WaifuSummary{}, fmt.Errorf("postgres: put daily waifu: %w", err)
	}
	return s.Get(ctx, day)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func toSummary(d gen.DailyWaifu) domain.WaifuSummary {
	return domain.WaifuSummary{
		Slug:         d.Slug,
		UUID:         d.Uuid,
		Name:         d.Name,
		OriginalName: d.OriginalName,
		RomajiName:   d.RomajiName,
		PictureURL:   d.PictureUrl,
		Likes:        int(d.Likes),
		Trash:        int(d.Trash),
	}
}

package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/steven-peralta/rosiebot/internal/adapter/postgres/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const snapshotsToKeep = 2

type RankingStore struct {
	pool *pgxpool.Pool
	q    *gen.Queries
}

var _ app.RankingStore = (*RankingStore)(nil)

func NewRankingStore(pool *pgxpool.Pool) *RankingStore {
	return &RankingStore{pool: pool, q: gen.New(pool)}
}

func (s *RankingStore) LoadLatest(ctx context.Context) (*domain.Ranking, error) {
	snap, err := s.q.LatestCompleteSnapshot(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, app.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: latest snapshot: %w", err)
	}
	rows, err := s.q.RankingRows(ctx, snap.ID)
	if err != nil {
		return nil, fmt.Errorf("postgres: ranking rows: %w", err)
	}
	sorted := make([]domain.RankedWaifu, len(rows))
	for i, row := range rows {
		sorted[i] = domain.RankedWaifu{
			WaifuSummary: domain.WaifuSummary{
				Slug:         row.Slug,
				UUID:         row.Uuid,
				Name:         row.Name,
				OriginalName: row.OriginalName,
				RomajiName:   row.RomajiName,
				PictureURL:   row.PictureUrl,
				Likes:        int(row.Likes),
				Trash:        int(row.Trash),
			},
			Score: row.Score,
		}
	}
	return domain.RankingFromSorted(sorted, snap.FetchedAt, int(snap.CutoffPage)), nil
}

func (s *RankingStore) Save(ctx context.Context, r *domain.Ranking) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := s.q.WithTx(tx)

	rows := r.Rows()
	id, err := q.InsertRankingSnapshot(ctx, gen.InsertRankingSnapshotParams{
		FetchedAt:  r.FetchedAt,
		CutoffPage: int32(r.CutoffPage),
		RowCount:   int32(len(rows)),
	})
	if err != nil {
		return fmt.Errorf("postgres: insert snapshot: %w", err)
	}
	params := make([]gen.InsertRankingRowsParams, len(rows))
	for i, row := range rows {
		params[i] = gen.InsertRankingRowsParams{
			SnapshotID:   id,
			Position:     int32(row.Position),
			Slug:         row.Slug,
			Uuid:         row.UUID,
			Name:         row.Name,
			OriginalName: row.OriginalName,
			RomajiName:   row.RomajiName,
			PictureUrl:   row.PictureURL,
			Likes:        int32(row.Likes),
			Trash:        int32(row.Trash),
			Score:        row.Score,
			Stars:        int16(row.Stars),
		}
	}
	if len(params) > 0 {
		if _, err := q.InsertRankingRows(ctx, params); err != nil {
			return fmt.Errorf("postgres: copy ranking rows: %w", err)
		}
	}
	if err := q.CompleteRankingSnapshot(ctx, id); err != nil {
		return fmt.Errorf("postgres: complete snapshot: %w", err)
	}
	if err := q.PruneRankingSnapshots(ctx, snapshotsToKeep); err != nil {
		return fmt.Errorf("postgres: prune snapshots: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

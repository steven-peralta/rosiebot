package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/steven-peralta/rosiebot/internal/adapter/postgres/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const uniqueViolation = "23505"

type Store struct {
	repo
	pool *pgxpool.Pool
}

var _ app.PlayerStore = (*Store)(nil)

func NewStore(pool *pgxpool.Pool, clock app.Clock) *Store {
	if clock == nil {
		clock = app.SystemClock()
	}
	return &Store{repo: repo{q: gen.New(pool), clock: clock}, pool: pool}
}

func (s *Store) WithinTx(ctx context.Context, fn func(app.PlayerRepo) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(repo{q: s.q.WithTx(tx), clock: s.clock}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

type repo struct {
	q     *gen.Queries
	clock app.Clock
}

var _ app.PlayerRepo = repo{}

func (r repo) EnsurePlayer(ctx context.Context, key domain.PlayerKey) (domain.Player, error) {
	row, err := r.q.EnsurePlayer(ctx, gen.EnsurePlayerParams{
		GuildID: key.GuildID,
		UserID:  key.UserID,
		Coins:   domain.StartingCoins,
		Now:     r.clock.Now(),
	})
	if err != nil {
		return domain.Player{}, fmt.Errorf("postgres: ensure player: %w", err)
	}
	return toPlayer(row), nil
}

func (r repo) GetPlayer(ctx context.Context, key domain.PlayerKey) (domain.Player, error) {
	row, err := r.q.GetPlayer(ctx, gen.GetPlayerParams{GuildID: key.GuildID, UserID: key.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Player{}, app.ErrNotFound
	}
	if err != nil {
		return domain.Player{}, fmt.Errorf("postgres: get player: %w", err)
	}
	return toPlayer(row), nil
}

func (r repo) LockPlayers(ctx context.Context, keys ...domain.PlayerKey) ([]domain.Player, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	guild := keys[0].GuildID
	userIDs := make([]string, 0, len(keys))
	for _, k := range keys {
		if k.GuildID != guild {
			return nil, fmt.Errorf("postgres: lock players: keys span guilds %q and %q", guild, k.GuildID)
		}
		userIDs = append(userIDs, k.UserID)
	}
	rows, err := r.q.LockPlayers(ctx, gen.LockPlayersParams{GuildID: guild, UserIds: userIDs})
	if err != nil {
		return nil, fmt.Errorf("postgres: lock players: %w", err)
	}
	if len(rows) != len(keys) {
		return nil, app.ErrNotFound
	}
	out := make([]domain.Player, len(rows))
	for i, row := range rows {
		out[i] = toPlayer(row)
	}
	return out, nil
}

func (r repo) DebitCoins(ctx context.Context, key domain.PlayerKey, amount int64) (int64, bool, error) {
	balance, err := r.q.DebitCoins(ctx, gen.DebitCoinsParams{
		Amount:  amount,
		Now:     r.clock.Now(),
		GuildID: key.GuildID,
		UserID:  key.UserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("postgres: debit: %w", err)
	}
	return balance, true, nil
}

func (r repo) ClaimDaily(ctx context.Context, key domain.PlayerKey, amount int64, windowStart, now time.Time) (int64, bool, error) {
	balance, err := r.q.ClaimDaily(ctx, gen.ClaimDailyParams{
		Amount:      amount,
		Now:         &now,
		GuildID:     key.GuildID,
		UserID:      key.UserID,
		WindowStart: &windowStart,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("postgres: claim daily: %w", err)
	}
	return balance, true, nil
}

func (r repo) AddOwned(ctx context.Context, key domain.PlayerKey, w domain.OwnedWaifu) (bool, error) {
	n, err := r.q.AddOwned(ctx, gen.AddOwnedParams{
		GuildID:    key.GuildID,
		UserID:     key.UserID,
		Slug:       w.Slug,
		Uuid:       w.UUID,
		Name:       w.Name,
		PictureUrl: w.PictureURL,
		Likes:      int32(w.Likes),
		Trash:      int32(w.Trash),
		AcquiredAt: w.AcquiredAt,
	})
	if err != nil {
		return false, fmt.Errorf("postgres: add owned: %w", err)
	}
	return n == 1, nil
}

func (r repo) GetOwned(ctx context.Context, key domain.PlayerKey, slug string) (domain.OwnedWaifu, error) {
	row, err := r.q.GetOwned(ctx, gen.GetOwnedParams{GuildID: key.GuildID, UserID: key.UserID, Slug: slug})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OwnedWaifu{}, app.ErrNotFound
	}
	if err != nil {
		return domain.OwnedWaifu{}, fmt.Errorf("postgres: get owned: %w", err)
	}
	return toOwned(row), nil
}

func (r repo) SetCoins(ctx context.Context, key domain.PlayerKey, coins int64) (int64, error) {
	balance, err := r.q.SetCoins(ctx, gen.SetCoinsParams{
		Coins:   coins,
		Now:     r.clock.Now(),
		GuildID: key.GuildID,
		UserID:  key.UserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, app.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("postgres: set coins: %w", err)
	}
	return balance, nil
}

func (r repo) AdjustCoins(ctx context.Context, key domain.PlayerKey, delta int64) (int64, bool, error) {
	balance, err := r.q.AdjustCoins(ctx, gen.AdjustCoinsParams{
		Delta:   delta,
		Now:     r.clock.Now(),
		GuildID: key.GuildID,
		UserID:  key.UserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("postgres: adjust coins: %w", err)
	}
	return balance, true, nil
}

func (r repo) SellOwned(ctx context.Context, key domain.PlayerKey, slug string, price int64) (int64, bool, error) {
	balance, err := r.q.SellOwned(ctx, gen.SellOwnedParams{
		Price:   price,
		Now:     r.clock.Now(),
		GuildID: key.GuildID,
		UserID:  key.UserID,
		Slug:    slug,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("postgres: sell owned: %w", err)
	}
	return balance, true, nil
}

func (r repo) OwnedSlugs(ctx context.Context, key domain.PlayerKey, slugs []string) ([]string, error) {
	if len(slugs) == 0 {
		return []string{}, nil
	}
	out, err := r.q.OwnedSlugs(ctx, gen.OwnedSlugsParams{GuildID: key.GuildID, UserID: key.UserID, Slugs: slugs})
	if err != nil {
		return nil, fmt.Errorf("postgres: owned slugs: %w", err)
	}
	return out, nil
}

func (r repo) TransferOwned(ctx context.Context, from, to domain.PlayerKey, slugs []string) (int64, error) {
	if len(slugs) == 0 {
		return 0, nil
	}
	if from.GuildID != to.GuildID {
		return 0, fmt.Errorf("postgres: transfer across guilds is not supported")
	}
	n, err := r.q.TransferOwned(ctx, gen.TransferOwnedParams{
		ToUserID:   to.UserID,
		GuildID:    from.GuildID,
		FromUserID: from.UserID,
		Slugs:      slugs,
	})
	if isUniqueViolation(err) {
		return 0, app.ErrAlreadyOwned
	}
	if err != nil {
		return 0, fmt.Errorf("postgres: transfer owned: %w", err)
	}
	return n, nil
}

func (r repo) ListOwned(ctx context.Context, key domain.PlayerKey, prefix string, limit int) ([]domain.OwnedWaifu, error) {
	rows, err := r.q.ListOwned(ctx, gen.ListOwnedParams{
		GuildID:  key.GuildID,
		UserID:   key.UserID,
		Prefix:   escapeLike(prefix),
		RowLimit: int32(max(limit, 0)),
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: list owned: %w", err)
	}
	out := make([]domain.OwnedWaifu, len(rows))
	for i, row := range rows {
		out[i] = toOwned(row)
	}
	return out, nil
}

func (r repo) CountOwned(ctx context.Context, key domain.PlayerKey) (int, error) {
	n, err := r.q.CountOwned(ctx, gen.CountOwnedParams{GuildID: key.GuildID, UserID: key.UserID})
	if err != nil {
		return 0, fmt.Errorf("postgres: count owned: %w", err)
	}
	return int(n), nil
}

func toPlayer(p gen.Player) domain.Player {
	return domain.Player{
		Key:            domain.PlayerKey{GuildID: p.GuildID, UserID: p.UserID},
		Coins:          p.Coins,
		DailyClaimedAt: p.DailyClaimedAt,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func toOwned(i gen.Inventory) domain.OwnedWaifu {
	return domain.OwnedWaifu{
		Slug:       i.Slug,
		UUID:       i.Uuid,
		Name:       i.Name,
		PictureURL: i.PictureUrl,
		Likes:      int(i.Likes),
		Trash:      int(i.Trash),
		AcquiredAt: i.AcquiredAt,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string {
	return likeEscaper.Replace(s)
}

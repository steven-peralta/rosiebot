package app

import (
	"context"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type SearchPage struct {
	Items    []domain.WaifuSummary
	Page     int
	LastPage int
}

type PopularPage struct {
	Rows     []domain.WaifuSummary
	Page     int
	LastPage int
}

type WaifuSource interface {
	Random(ctx context.Context) (domain.WaifuSummary, error)
	Daily(ctx context.Context) (domain.WaifuSummary, error)
	Get(ctx context.Context, slug string) (domain.Waifu, error)
	ListCharacters(ctx context.Context, page int) (SearchPage, error)
	SearchWaifus(ctx context.Context, term string, page int) (SearchPage, error)
	SearchWorks(ctx context.Context, term string) ([]domain.Series, error)
	Work(ctx context.Context, slug string) (domain.Series, error)
	WorkCharacters(ctx context.Context, slug string, page int) (SearchPage, error)
	PopularPage(ctx context.Context, page int) (PopularPage, error)
}

type PlayerRepo interface {
	EnsurePlayer(ctx context.Context, key domain.PlayerKey) (domain.Player, error)
	GetPlayer(ctx context.Context, key domain.PlayerKey) (domain.Player, error)
	LockPlayers(ctx context.Context, keys ...domain.PlayerKey) ([]domain.Player, error)
	DebitCoins(ctx context.Context, key domain.PlayerKey, amount int64) (balance int64, ok bool, err error)
	SetCoins(ctx context.Context, key domain.PlayerKey, coins int64) (balance int64, err error)
	AdjustCoins(ctx context.Context, key domain.PlayerKey, delta int64) (balance int64, ok bool, err error)
	ClaimDaily(ctx context.Context, key domain.PlayerKey, amount int64, windowStart, now time.Time) (balance int64, ok bool, err error)
	AddOwned(ctx context.Context, key domain.PlayerKey, w domain.OwnedWaifu) (inserted bool, err error)
	GetOwned(ctx context.Context, key domain.PlayerKey, slug string) (domain.OwnedWaifu, error)
	SellOwned(ctx context.Context, key domain.PlayerKey, slug string, price int64) (balance int64, ok bool, err error)
	OwnedSlugs(ctx context.Context, key domain.PlayerKey, slugs []string) ([]string, error)
	TransferOwned(ctx context.Context, from, to domain.PlayerKey, slugs []string) (moved int64, err error)
	ListOwned(ctx context.Context, key domain.PlayerKey, prefix string, limit int) ([]domain.OwnedWaifu, error)
	CountOwned(ctx context.Context, key domain.PlayerKey) (int, error)
	RecordRoll(ctx context.Context, key domain.PlayerKey, r domain.RollRecord) error
	RecentRolls(ctx context.Context, key domain.PlayerKey, limit int) ([]domain.RollRecord, error)
	CountRolls(ctx context.Context, key domain.PlayerKey) (int, error)
	GuildPlayers(ctx context.Context, guildID string) ([]domain.Player, error)
	GuildInventory(ctx context.Context, guildID string) ([]OwnedRow, error)
}

type OwnedRow struct {
	UserID string
	Slug   string
}

type PlayerStore interface {
	PlayerRepo
	WithinTx(ctx context.Context, fn func(PlayerRepo) error) error
}

type RankingStore interface {
	LoadLatest(ctx context.Context) (*domain.Ranking, error)
	Save(ctx context.Context, r *domain.Ranking) error
}

type RankingProvider interface {
	Current() *domain.Ranking
}

type DailyStore interface {
	Get(ctx context.Context, day time.Time) (domain.WaifuSummary, error)
	Put(ctx context.Context, day time.Time, w domain.WaifuSummary) (domain.WaifuSummary, error)
	Replace(ctx context.Context, day time.Time, w domain.WaifuSummary) error
}

type BannerStore interface {
	Get(ctx context.Context, weekStart time.Time) (domain.Banner, error)
	Put(ctx context.Context, b domain.Banner) (domain.Banner, error)
	Replace(ctx context.Context, b domain.Banner) error
}

type FavoriteStore interface {
	Add(ctx context.Context, key domain.PlayerKey, fav domain.Favorite) (added bool, err error)
	Remove(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind, slug string) (removed bool, err error)
	List(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind) ([]domain.Favorite, error)
	Find(ctx context.Context, kind domain.FavoriteKind, slugs []string, guildID string) ([]FavoriteMatch, error)
}

type FavoriteMatch struct {
	Key      domain.PlayerKey
	Favorite domain.Favorite
}

type AlertSetting struct {
	Enabled  bool
	DMClosed bool
}

type AlertStore interface {
	Setting(ctx context.Context, key domain.PlayerKey) (AlertSetting, error)
	SetEnabled(ctx context.Context, key domain.PlayerKey, enabled bool, now time.Time) error
	MarkDMClosed(ctx context.Context, userID string, now time.Time) error
	ClearDMClosed(ctx context.Context, userID string) error
	MarkSent(ctx context.Context, eventID, userID string, now time.Time) (first bool, err error)
}

type Notifier interface {
	DirectMessage(ctx context.Context, userID, content string) error
}

type Clock interface {
	Now() time.Time
}

type Random interface {
	IntN(n int) int
}

type priorityKey struct{}

func WithBackground(ctx context.Context) context.Context {
	return context.WithValue(ctx, priorityKey{}, true)
}

func IsBackground(ctx context.Context) bool {
	v, _ := ctx.Value(priorityKey{}).(bool)
	return v
}

type ClockFunc func() time.Time

func (f ClockFunc) Now() time.Time { return f() }

func SystemClock() Clock { return ClockFunc(time.Now) }

package app_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/app/mocks"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

var (
	alice = domain.PlayerKey{GuildID: "guild", UserID: "alice"}
	bob   = domain.PlayerKey{GuildID: "guild", UserID: "bob"}
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type seqRandom struct {
	t    *testing.T
	vals []int
}

func (r *seqRandom) IntN(n int) int {
	if len(r.vals) == 0 {
		return 0
	}
	v := r.vals[0]
	r.vals = r.vals[1:]
	if v >= n {
		r.t.Fatalf("scripted random %d out of range for IntN(%d)", v, n)
	}
	return v
}

func d100(v int) int { return v - 1 }

func chicago(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func summary(slug string, likes, trash int) domain.WaifuSummary {
	return domain.WaifuSummary{Slug: slug, UUID: "uuid-" + slug, Name: "Name " + slug, PictureURL: "https://img/" + slug, Likes: likes, Trash: trash}
}

func detail(slug string) domain.Waifu {
	return domain.Waifu{WaifuSummary: summary(slug, 10, 1), URL: "https://www.mywaifulist.moe/waifu/" + slug, Description: "desc " + slug}
}

func rankingOf(n int) *domain.Ranking {
	rows := make([]domain.WaifuSummary, n)
	for i := range rows {
		rows[i] = summary(fmt.Sprintf("ranked-%03d", i), 10000-i*10, 5)
	}
	return domain.BuildRanking(rows, domain.DefaultMinVotes, time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC), 1000)
}

type fixture struct {
	t       *testing.T
	ctx     context.Context
	clock   *fakeClock
	rng     *seqRandom
	loc     *time.Location
	players *memory.PlayerStore
	daily   *memory.DailyStore
	banners *memory.BannerStore
	ranking *memory.RankingHolder
	source  *mocks.WaifuSource
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	loc := chicago(t)
	clock := &fakeClock{now: time.Date(2026, 9, 16, 12, 0, 0, 0, loc)}
	return &fixture{
		t:       t,
		ctx:     context.Background(),
		clock:   clock,
		rng:     &seqRandom{t: t},
		loc:     loc,
		players: memory.NewPlayerStore(clock),
		daily:   memory.NewDailyStore(),
		banners: memory.NewBannerStore(),
		ranking: memory.NewRankingHolder(nil),
		source:  mocks.NewWaifuSource(t),
	}
}

func (f *fixture) script(vals ...int) { f.rng.vals = append(f.rng.vals, vals...) }

func (f *fixture) wotd() *app.WotdService {
	return app.NewWotdService(f.daily, f.ranking, f.clock, f.rng, f.loc, nil)
}

func (f *fixture) banner() *app.BannerService {
	return app.NewBannerService(f.banners, f.ranking, f.source, f.clock, f.rng, f.loc, app.BannerConfig{}, nil)
}

func (f *fixture) seedBanner(slugs ...string) domain.Banner {
	f.t.Helper()
	ranking := rankingOf(200)
	chars := ranking.Subset(slugs)
	week := domain.BannerWeekStart(f.clock.now, f.loc)
	stored, err := f.banners.Put(f.ctx, domain.NewBanner(week, domain.Series{Slug: "re-zero", Name: "Re:Zero", URL: "https://www.mywaifulist.moe/series/re-zero", PictureURL: "https://img/re-zero"}, chars))
	if err != nil {
		f.t.Fatal(err)
	}
	return stored
}

func (f *fixture) roll() *app.RollService {
	return app.NewRollService(f.players, f.source, f.ranking, f.wotd(), f.banner(), f.clock, f.rng, 0, nil)
}

func (f *fixture) give(key domain.PlayerKey, slugs ...string) {
	f.t.Helper()
	if _, err := f.players.EnsurePlayer(f.ctx, key); err != nil {
		f.t.Fatal(err)
	}
	for i, slug := range slugs {
		w := domain.OwnedFromSummary(summary(slug, 1, 0), f.clock.now.Add(time.Duration(i)*time.Second))
		if _, err := f.players.AddOwned(f.ctx, key, w); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fixture) coins(key domain.PlayerKey) int64 {
	f.t.Helper()
	p, err := f.players.GetPlayer(f.ctx, key)
	if err != nil {
		f.t.Fatal(err)
	}
	return p.Coins
}

func (f *fixture) owns(key domain.PlayerKey, slug string) bool {
	f.t.Helper()
	got, err := f.players.OwnedSlugs(f.ctx, key, []string{slug})
	if err != nil {
		f.t.Fatal(err)
	}
	return len(got) == 1
}

type hookedStore struct {
	app.PlayerStore
	beforeTx func()
}

func (h hookedStore) WithinTx(ctx context.Context, fn func(app.PlayerRepo) error) error {
	if h.beforeTx != nil {
		h.beforeTx()
	}
	return h.PlayerStore.WithinTx(ctx, fn)
}

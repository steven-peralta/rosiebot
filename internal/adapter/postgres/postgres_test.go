package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/steven-peralta/rosiebot/internal/adapter/postgres/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/app/mocks"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

var (
	testDSN  string
	testPool *pgxpool.Pool
	dbErr    error
)

const defaultTestImage = "postgres:16-alpine"

func TestMain(m *testing.M) {
	ctx := context.Background()
	if os.Getenv("TESTCONTAINERS_RYUK_DISABLED") == "" {
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	}
	image := os.Getenv("POSTGRES_TEST_IMAGE")
	if image == "" {
		image = defaultTestImage
	}
	container, err := tcpostgres.Run(ctx, image,
		tcpostgres.WithDatabase("rosiebot"),
		tcpostgres.WithUsername("rosie"),
		tcpostgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90*time.Second)),
	)
	if err != nil {
		dbErr = fmt.Errorf("start postgres container: %w", err)
		if os.Getenv("REQUIRE_DB") != "" {
			fmt.Fprintln(os.Stderr, dbErr)
			os.Exit(1)
		}
		os.Exit(m.Run())
	}
	testDSN, err = container.ConnectionString(ctx, "sslmode=disable")
	if err == nil {
		err = Migrate(testDSN)
	}
	if err == nil {
		testPool, err = Connect(ctx, testDSN)
	}
	if err != nil {
		dbErr = err
		_ = container.Terminate(ctx)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	testPool.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

func requireDB(t *testing.T) {
	t.Helper()
	if dbErr != nil {
		if os.Getenv("REQUIRE_DB") != "" {
			t.Fatal(dbErr)
		}
		t.Skipf("skipping: %v (set REQUIRE_DB=1 to fail instead)", dbErr)
	}
}

var guildSeq atomic.Int64

func keys(t *testing.T) (domain.PlayerKey, domain.PlayerKey) {
	t.Helper()
	requireDB(t)
	guild := fmt.Sprintf("guild-%d", guildSeq.Add(1))
	return domain.PlayerKey{GuildID: guild, UserID: "alice"}, domain.PlayerKey{GuildID: guild, UserID: "bob"}
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func owned(slug string, at time.Time) domain.OwnedWaifu {
	return domain.OwnedWaifu{Slug: slug, UUID: "u-" + slug, Name: "Name " + slug, PictureURL: "p", Likes: 3, Trash: 1, AcquiredAt: at}
}

func TestMigrations_UpDownUp(t *testing.T) {
	requireDB(t)
	m, err := NewMigrator(testDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Down(); err != nil {
		t.Fatalf("down: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("up: %v", err)
	}
	if err := Migrate(testDSN); err != nil {
		t.Fatalf("idempotent up: %v", err)
	}
	if got := migrateURL("postgresql://u@h/db"); got != "pgx5://u@h/db" {
		t.Errorf("migrateURL = %q", got)
	}
	if got := migrateURL("pgx5://u@h/db"); got != "pgx5://u@h/db" {
		t.Errorf("migrateURL passthrough = %q", got)
	}
	if _, err := Connect(context.Background(), "postgres://nobody:x@127.0.0.1:1/none?connect_timeout=1"); err == nil {
		t.Error("connect to closed port should fail")
	}
}

func TestRepo_EnsureAndGetPlayer(t *testing.T) {
	alice, _ := keys(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	s := NewStore(testPool, fixedClock{now})

	if _, err := s.GetPlayer(ctx, alice); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("unknown player err = %v", err)
	}
	p, err := s.EnsurePlayer(ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	if p.Coins != domain.StartingCoins || p.Key != alice || !p.CreatedAt.Equal(now) || p.DailyClaimedAt != nil {
		t.Errorf("player = %+v", p)
	}
	if _, ok, err := s.DebitCoins(ctx, alice, 50); err != nil || !ok {
		t.Fatal("debit failed")
	}
	again, err := s.EnsurePlayer(ctx, alice)
	if err != nil || again.Coins != 150 {
		t.Errorf("EnsurePlayer must not reset an existing player: %+v %v", again, err)
	}
	got, err := s.GetPlayer(ctx, alice)
	if err != nil || got.Coins != 150 {
		t.Errorf("GetPlayer = %+v %v", got, err)
	}
	if bal, err := s.SetCoins(ctx, alice, 900); err != nil || bal != 900 {
		t.Errorf("SetCoins = %d %v", bal, err)
	}
	if bal, ok, err := s.AdjustCoins(ctx, alice, -900); err != nil || !ok || bal != 0 {
		t.Errorf("AdjustCoins to zero = %d %v %v", bal, ok, err)
	}
	if _, ok, err := s.AdjustCoins(ctx, alice, -1); err != nil || ok {
		t.Errorf("AdjustCoins below zero should not apply: ok=%v err=%v", ok, err)
	}
	if bal, ok, err := s.AdjustCoins(ctx, alice, 30); err != nil || !ok || bal != 30 {
		t.Errorf("AdjustCoins up = %d %v %v", bal, ok, err)
	}
	unknown := domain.PlayerKey{GuildID: alice.GuildID, UserID: "nobody"}
	if _, err := s.SetCoins(ctx, unknown, 1); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("SetCoins on unknown = %v", err)
	}
	if _, ok, err := s.AdjustCoins(ctx, unknown, 1); err != nil || ok {
		t.Errorf("AdjustCoins on unknown = ok %v err %v", ok, err)
	}
}

func TestRepo_DebitCoins_ConcurrentDoubleSpend(t *testing.T) {
	alice, _ := keys(t)
	ctx := context.Background()
	s := NewStore(testPool, nil)
	if _, err := s.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}

	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok, err := s.DebitCoins(ctx, alice, domain.RollCost)
			if err != nil {
				t.Error(err)
			}
			if ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Errorf("exactly one debit of the full balance should succeed, got %d", wins.Load())
	}
	p, _ := s.GetPlayer(ctx, alice)
	if p.Coins != 0 {
		t.Errorf("balance = %d", p.Coins)
	}
	if _, ok, _ := s.DebitCoins(ctx, alice, 1); ok {
		t.Error("debit from empty balance should fail")
	}
}

func TestRepo_ClaimDaily(t *testing.T) {
	alice, _ := keys(t)
	ctx := context.Background()
	s := NewStore(testPool, nil)
	if _, err := s.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	window := now.Add(-2 * time.Hour)

	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok, err := s.ClaimDaily(ctx, alice, 400, window, now)
			if err != nil {
				t.Error(err)
			}
			if ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Errorf("exactly one concurrent claim should win, got %d", wins.Load())
	}
	p, _ := s.GetPlayer(ctx, alice)
	if p.Coins != domain.StartingCoins+400 || p.DailyClaimedAt == nil || !p.DailyClaimedAt.Equal(now) {
		t.Errorf("player after claim = %+v", p)
	}

	nextWindow := now.Add(22 * time.Hour)
	bal, ok, err := s.ClaimDaily(ctx, alice, 400, nextWindow, nextWindow.Add(time.Minute))
	if err != nil || !ok || bal != domain.StartingCoins+800 {
		t.Errorf("claim in next window = %d %v %v", bal, ok, err)
	}
}

func TestRepo_Inventory(t *testing.T) {
	alice, bob := keys(t)
	ctx := context.Background()
	s := NewStore(testPool, nil)
	for _, k := range []domain.PlayerKey{alice, bob} {
		if _, err := s.EnsurePlayer(ctx, k); err != nil {
			t.Fatal(err)
		}
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if ins, err := s.AddOwned(ctx, alice, owned("rem", base)); err != nil || !ins {
		t.Fatalf("first insert = %v %v", ins, err)
	}
	if ins, err := s.AddOwned(ctx, alice, owned("rem", base)); err != nil || ins {
		t.Fatalf("duplicate insert = %v %v", ins, err)
	}
	for i, slug := range []string{"ram", "emilia", "100%_odd", "beatrice"} {
		if _, err := s.AddOwned(ctx, alice, owned(slug, base.Add(time.Duration(i+1)*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}

	got, err := s.GetOwned(ctx, alice, "rem")
	if err != nil || got.Slug != "rem" || got.UUID != "u-rem" || got.Likes != 3 || got.Trash != 1 || !got.AcquiredAt.Equal(base) {
		t.Errorf("GetOwned = %+v %v", got, err)
	}
	if _, err := s.GetOwned(ctx, alice, "nope"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("GetOwned missing = %v", err)
	}

	slugs, err := s.OwnedSlugs(ctx, alice, []string{"rem", "ram", "nope"})
	if err != nil || len(slugs) != 2 {
		t.Errorf("OwnedSlugs = %v %v", slugs, err)
	}
	if empty, err := s.OwnedSlugs(ctx, alice, nil); err != nil || len(empty) != 0 {
		t.Errorf("OwnedSlugs(nil) = %v %v", empty, err)
	}

	all, err := s.ListOwned(ctx, alice, "", 0)
	if err != nil || len(all) != 5 || all[0].Slug != "rem" || all[4].Slug != "beatrice" {
		t.Errorf("ListOwned all = %+v %v", all, err)
	}
	limited, _ := s.ListOwned(ctx, alice, "", 2)
	if len(limited) != 2 {
		t.Errorf("ListOwned limit = %d", len(limited))
	}
	prefixed, _ := s.ListOwned(ctx, alice, "NAME R", 0)
	if len(prefixed) != 2 {
		t.Errorf("ListOwned prefix (case-insensitive) = %d", len(prefixed))
	}
	escaped, _ := s.ListOwned(ctx, alice, "Name 100%", 0)
	if len(escaped) != 1 || escaped[0].Slug != "100%_odd" {
		t.Errorf("ListOwned should escape LIKE wildcards: %+v", escaped)
	}
	wild, _ := s.ListOwned(ctx, alice, "%", 0)
	if len(wild) != 0 {
		t.Errorf("a bare %% must not match everything: %d", len(wild))
	}
	if n, err := s.CountOwned(ctx, alice); err != nil || n != 5 {
		t.Errorf("CountOwned = %d %v", n, err)
	}

	bal, ok, err := s.SellOwned(ctx, alice, "rem", domain.SellPrice)
	if err != nil || !ok || bal != domain.StartingCoins+domain.SellPrice {
		t.Errorf("SellOwned = %d %v %v", bal, ok, err)
	}
	if _, ok, _ := s.SellOwned(ctx, alice, "rem", domain.SellPrice); ok {
		t.Error("selling twice should fail")
	}
	if _, ok, _ := s.SellOwned(ctx, bob, "ram", domain.SellPrice); ok {
		t.Error("selling someone else's waifu should fail")
	}
	p, _ := s.GetPlayer(ctx, bob)
	if p.Coins != domain.StartingCoins {
		t.Error("failed sale must not credit")
	}
}

func TestRepo_SellOwned_ConcurrentSellsOnce(t *testing.T) {
	alice, _ := keys(t)
	ctx := context.Background()
	s := NewStore(testPool, nil)
	if _, err := s.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddOwned(ctx, alice, owned("rem", time.Now())); err != nil {
		t.Fatal(err)
	}
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok, err := s.SellOwned(ctx, alice, "rem", domain.SellPrice)
			if err != nil {
				t.Error(err)
			}
			if ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	p, _ := s.GetPlayer(ctx, alice)
	if wins.Load() != 1 || p.Coins != domain.StartingCoins+domain.SellPrice {
		t.Errorf("wins=%d coins=%d", wins.Load(), p.Coins)
	}
}

func TestRepo_TransferAndLock(t *testing.T) {
	alice, bob := keys(t)
	ctx := context.Background()
	s := NewStore(testPool, nil)
	for _, k := range []domain.PlayerKey{alice, bob} {
		if _, err := s.EnsurePlayer(ctx, k); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()
	for _, slug := range []string{"rem", "shared"} {
		if _, err := s.AddOwned(ctx, alice, owned(slug, now)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.AddOwned(ctx, bob, owned("shared", now)); err != nil {
		t.Fatal(err)
	}

	moved, err := s.TransferOwned(ctx, alice, bob, []string{"rem", "missing"})
	if err != nil || moved != 1 {
		t.Errorf("TransferOwned = %d %v", moved, err)
	}
	if got, _ := s.OwnedSlugs(ctx, bob, []string{"rem"}); len(got) != 1 {
		t.Error("rem should now belong to bob")
	}
	if _, err := s.TransferOwned(ctx, alice, bob, []string{"shared"}); !errors.Is(err, app.ErrAlreadyOwned) {
		t.Errorf("duplicate transfer err = %v", err)
	}
	if n, err := s.TransferOwned(ctx, alice, bob, nil); err != nil || n != 0 {
		t.Errorf("empty transfer = %d %v", n, err)
	}
	if _, err := s.TransferOwned(ctx, alice, domain.PlayerKey{GuildID: "elsewhere", UserID: "x"}, []string{"shared"}); err == nil {
		t.Error("cross-guild transfer should fail")
	}

	players, err := s.LockPlayers(ctx, bob, alice)
	if err != nil || len(players) != 2 || players[0].Key != alice || players[1].Key != bob {
		t.Errorf("LockPlayers = %+v %v", players, err)
	}
	if _, err := s.LockPlayers(ctx, alice, domain.PlayerKey{GuildID: alice.GuildID, UserID: "ghost"}); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("LockPlayers with missing = %v", err)
	}
	if _, err := s.LockPlayers(ctx, alice, domain.PlayerKey{GuildID: "other", UserID: "x"}); err == nil {
		t.Error("LockPlayers across guilds should fail")
	}
	if got, err := s.LockPlayers(ctx); err != nil || got != nil {
		t.Errorf("LockPlayers() = %v %v", got, err)
	}
}

func TestStore_WithinTx(t *testing.T) {
	alice, _ := keys(t)
	ctx := context.Background()
	s := NewStore(testPool, nil)
	if _, err := s.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := s.WithinTx(ctx, func(r app.PlayerRepo) error {
		if _, ok, err := r.DebitCoins(ctx, alice, 100); err != nil || !ok {
			t.Fatal("debit in tx failed")
		}
		if _, err := r.AddOwned(ctx, alice, owned("rem", time.Now())); err != nil {
			t.Fatal(err)
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	p, _ := s.GetPlayer(ctx, alice)
	n, _ := s.CountOwned(ctx, alice)
	if p.Coins != domain.StartingCoins || n != 0 {
		t.Errorf("rollback failed: coins=%d owned=%d", p.Coins, n)
	}

	err = s.WithinTx(ctx, func(r app.PlayerRepo) error {
		if _, err := r.LockPlayers(ctx, alice); err != nil {
			return err
		}
		if _, ok, err := r.DebitCoins(ctx, alice, 100); err != nil || !ok {
			return errors.New("debit failed")
		}
		_, err := r.AddOwned(ctx, alice, owned("rem", time.Now()))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetPlayer(ctx, alice)
	n, _ = s.CountOwned(ctx, alice)
	if p.Coins != 100 || n != 1 {
		t.Errorf("commit failed: coins=%d owned=%d", p.Coins, n)
	}

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.WithinTx(cancelled, func(app.PlayerRepo) error { return nil }); err == nil {
		t.Error("begin on cancelled context should fail")
	}
}

func TestRankingStore_SaveLoadPrune(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	rs := NewRankingStore(testPool)

	build := func(n int, fetched time.Time) *domain.Ranking {
		rows := make([]domain.WaifuSummary, n)
		for i := range rows {
			rows[i] = domain.WaifuSummary{Slug: fmt.Sprintf("s%03d", i), UUID: "u", Name: "N", OriginalName: "O", RomajiName: "R", PictureURL: "P", Likes: 5000 - i, Trash: 10}
		}
		return domain.BuildRanking(rows, domain.DefaultMinVotes, fetched, 900+n)
	}
	base := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	for i, n := range []int{50, 100, 200} {
		if err := rs.Save(ctx, build(n, base.Add(time.Duration(i)*24*time.Hour))); err != nil {
			t.Fatal(err)
		}
	}

	latest, err := rs.LoadLatest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if latest.Len() != 200 || latest.CutoffPage != 1100 || !latest.FetchedAt.Equal(base.Add(48*time.Hour)) {
		t.Errorf("latest = len %d cutoff %d fetched %v", latest.Len(), latest.CutoffPage, latest.FetchedAt)
	}
	rows := latest.Rows()
	if rows[0].Slug != "s000" || rows[0].Position != 1 || rows[0].Stars != 5 || rows[199].Stars != 1 || rows[0].Score != domain.Score(5000, 10) {
		t.Errorf("row 0 = %+v, row 199 stars = %d", rows[0], rows[199].Stars)
	}
	if row, ok := latest.Lookup("s150"); !ok || row.Position != 151 || row.UUID != "u" || row.OriginalName != "O" {
		t.Errorf("Lookup = %+v %v", row, ok)
	}

	var snapshots int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM ranking_snapshots").Scan(&snapshots); err != nil {
		t.Fatal(err)
	}
	if snapshots != snapshotsToKeep {
		t.Errorf("snapshots kept = %d, want %d", snapshots, snapshotsToKeep)
	}

	if err := rs.Save(ctx, domain.BuildRanking(nil, domain.DefaultMinVotes, base.Add(72*time.Hour), 1)); err != nil {
		t.Fatalf("saving an empty ranking: %v", err)
	}
	empty, err := rs.LoadLatest(ctx)
	if err != nil || empty.Len() != 0 {
		t.Errorf("empty latest = %v %v", empty, err)
	}
}

func TestRankingStore_LoadLatestEmpty(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	if _, err := testPool.Exec(ctx, "DELETE FROM ranking_snapshots"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRankingStore(testPool).LoadLatest(ctx); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestDailyStore(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	ds := NewDailyStore(testPool)
	day := time.Date(2026, 9, 16, 0, 0, 0, 0, time.FixedZone("CST", -6*3600))

	if _, err := ds.Get(ctx, day); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("Get on empty = %v", err)
	}
	first := domain.WaifuSummary{Slug: "first", UUID: "u1", Name: "First", OriginalName: "O", RomajiName: "R", PictureURL: "P", Likes: 10, Trash: 2}
	got, err := ds.Put(ctx, day, first)
	if err != nil || got != first {
		t.Fatalf("Put = %+v %v", got, err)
	}
	second, err := ds.Put(ctx, day, domain.WaifuSummary{Slug: "second", Name: "Second"})
	if err != nil || second.Slug != "first" {
		t.Errorf("second Put should return the first winner: %+v %v", second, err)
	}
	sameDayUTC := time.Date(2026, 9, 16, 23, 0, 0, 0, time.UTC)
	if got, err := ds.Get(ctx, sameDayUTC); err != nil || got.Slug != "first" {
		t.Errorf("Get by calendar date = %+v %v", got, err)
	}
}

func TestBannerStore(t *testing.T) {
	requireDB(t)
	ctx := context.Background()
	bs := NewBannerStore(testPool)
	week := time.Date(2026, 6, 1, 10, 0, 0, 0, time.FixedZone("CDT", -5*3600))

	if _, err := bs.Get(ctx, week); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("Get on empty = %v", err)
	}
	chars := []domain.RankedWaifu{
		{WaifuSummary: domain.WaifuSummary{Slug: "rem", UUID: "u1", Name: "Rem", OriginalName: "レム", RomajiName: "Remu", PictureURL: "P", Likes: 5000, Trash: 100}, Position: 3, Stars: 5},
		{WaifuSummary: domain.WaifuSummary{Slug: "ram", Name: "Ram", Likes: 2000, Trash: 200}, Position: 40, Stars: 3},
	}
	first := domain.NewBanner(week, domain.Series{Slug: "re-zero", UUID: "s1", Name: "Re:Zero", URL: "U", PictureURL: "SP", Description: "D"}, chars)
	got, err := bs.Put(ctx, first)
	if err != nil {
		t.Fatalf("Put = %v", err)
	}
	if !got.WeekStart.Equal(week) || got.Series != first.Series || len(got.Characters) != 2 {
		t.Fatalf("round trip = %+v", got)
	}
	if got.Characters[0].Slug != "rem" || got.Characters[0].Position != 3 || got.Characters[0].Stars != 5 || got.Characters[0].OriginalName != "レム" || got.Characters[0].Score != domain.Score(5000, 100) {
		t.Errorf("character round trip = %+v", got.Characters[0])
	}
	second, err := bs.Put(ctx, domain.NewBanner(week, domain.Series{Slug: "other", Name: "Other"}, nil))
	if err != nil || second.Series.Slug != "re-zero" {
		t.Errorf("second Put should return the first winner: %+v %v", second, err)
	}
	if got, err := bs.Get(ctx, week.UTC()); err != nil || got.Series.Slug != "re-zero" {
		t.Errorf("Get by the same instant in UTC = %+v %v", got, err)
	}
	if _, err := bs.Get(ctx, week.AddDate(0, 0, 7)); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("next week should be empty, got %v", err)
	}
	if err := bs.Replace(ctx, domain.NewBanner(week, domain.Series{Slug: "replaced", Name: "Replaced"}, chars[:1])); err != nil {
		t.Fatalf("Replace = %v", err)
	}
	if got, err := bs.Get(ctx, week); err != nil || got.Series.Slug != "replaced" || len(got.Characters) != 1 {
		t.Errorf("after replace = %+v %v", got, err)
	}
	if err := bs.Replace(ctx, domain.NewBanner(week.AddDate(0, 0, 14), domain.Series{Slug: "fresh", Name: "Fresh"}, nil)); err != nil {
		t.Fatalf("Replace on an empty week = %v", err)
	}
	if got, err := bs.Get(ctx, week.AddDate(0, 0, 14)); err != nil || got.Series.Slug != "fresh" {
		t.Errorf("replace should insert when missing: %+v %v", got, err)
	}
	if _, err := toBanner(gen.Banner{Characters: []byte("nope")}); err == nil {
		t.Error("corrupt payload should fail to decode")
	}
}

func TestServices_RollRaceOnPostgres(t *testing.T) {
	alice, _ := keys(t)
	ctx := context.Background()
	store := NewStore(testPool, nil)
	source := mocks.NewWaifuSource(t)
	var seq atomic.Int32
	source.EXPECT().Random(mock.Anything).RunAndReturn(func(context.Context) (domain.WaifuSummary, error) {
		n := seq.Add(1)
		return domain.WaifuSummary{Slug: fmt.Sprintf("w%d", n), Name: "W"}, nil
	}).Maybe()
	source.EXPECT().Get(mock.Anything, mock.Anything).RunAndReturn(func(_ context.Context, slug string) (domain.Waifu, error) {
		return domain.Waifu{WaifuSummary: domain.WaifuSummary{Slug: slug, Name: "W"}}, nil
	}).Maybe()

	if _, err := store.EnsurePlayer(ctx, alice); err != nil {
		t.Fatal(err)
	}
	rng := regularOnly{}
	svc := app.NewRollService(store, source, noRanking{}, nil, nil, app.SystemClock(), rng, 0, nil)

	var wins, broke atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.Roll(ctx, alice)
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, app.ErrInsufficientCoins):
				broke.Add(1)
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	p, _ := store.GetPlayer(ctx, alice)
	n, _ := store.CountOwned(ctx, alice)
	if wins.Load() != 1 || p.Coins != 0 || n != 1 {
		t.Errorf("wins=%d broke=%d coins=%d owned=%d", wins.Load(), broke.Load(), p.Coins, n)
	}
}

func TestServices_TradeAcceptRaceOnPostgres(t *testing.T) {
	alice, bob := keys(t)
	ctx := context.Background()
	store := NewStore(testPool, nil)
	for _, k := range []domain.PlayerKey{alice, bob} {
		if _, err := store.EnsurePlayer(ctx, k); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.AddOwned(ctx, alice, owned("rem", time.Now())); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddOwned(ctx, bob, owned("ram", time.Now())); err != nil {
		t.Fatal(err)
	}
	svc := app.NewTradeService(store)
	offer := domain.TradeOffer{Sender: alice, Target: bob, Give: []string{"rem"}, Receive: []string{"ram"}}

	var wins, conflicts atomic.Int32
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := svc.Accept(ctx, offer)
			switch {
			case err == nil:
				wins.Add(1)
			case errors.Is(err, app.ErrTradeConflict):
				conflicts.Add(1)
			default:
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 || conflicts.Load() != 5 {
		t.Errorf("wins=%d conflicts=%d", wins.Load(), conflicts.Load())
	}
	if got, _ := store.OwnedSlugs(ctx, bob, []string{"rem"}); len(got) != 1 {
		t.Error("rem should have moved to bob exactly once")
	}
	if got, _ := store.OwnedSlugs(ctx, alice, []string{"ram"}); len(got) != 1 {
		t.Error("ram should have moved to alice exactly once")
	}
}

type regularOnly struct{}

func (regularOnly) IntN(n int) int { return n - 1 }

type noRanking struct{}

func (noRanking) Current() *domain.Ranking { return nil }

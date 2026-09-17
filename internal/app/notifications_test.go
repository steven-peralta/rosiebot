package app_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/steven-peralta/rosiebot/internal/adapter/memory"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type fakeNotifier struct {
	sent   map[string][]string
	closed map[string]bool
	fail   map[string]error
}

func newFakeNotifier() *fakeNotifier {
	return &fakeNotifier{sent: map[string][]string{}, closed: map[string]bool{}, fail: map[string]error{}}
}

func (n *fakeNotifier) DirectMessage(_ context.Context, userID, content string) error {
	if n.closed[userID] {
		return app.ErrDMClosed
	}
	if err := n.fail[userID]; err != nil {
		return err
	}
	n.sent[userID] = append(n.sent[userID], content)
	return nil
}

type notifyFixture struct {
	*fixture
	favs     *memory.FavoriteStore
	alerts   *memory.AlertStore
	notifier *fakeNotifier
	svc      *app.NotificationService
	sleeps   []time.Duration
}

func newNotifyFixture(t *testing.T) *notifyFixture {
	f := &notifyFixture{fixture: newFixture(t), favs: memory.NewFavoriteStore(), alerts: memory.NewAlertStore(), notifier: newFakeNotifier()}
	f.svc = app.NewNotificationService(f.favs, f.alerts, f.notifier, f.clock, func(id string) string {
		if id == "guild" {
			return "Test Server"
		}
		return ""
	}, nil)
	f.svc.SetSleepForTest(func(_ context.Context, d time.Duration) error {
		f.sleeps = append(f.sleeps, d)
		return nil
	})
	return f
}

func (f *notifyFixture) fav(key domain.PlayerKey, kind domain.FavoriteKind, slug, name string) {
	f.t.Helper()
	if _, err := f.favs.Add(f.ctx, key, domain.Favorite{Kind: kind, Slug: slug, Name: name, AddedAt: f.clock.now}); err != nil {
		f.t.Fatal(err)
	}
}

var (
	carol       = domain.PlayerKey{GuildID: "guild", UserID: "carol"}
	aliceOther  = domain.PlayerKey{GuildID: "other", UserID: "alice"}
	testBanner  = domain.NewBanner(time.Date(2026, 9, 14, 15, 0, 0, 0, time.UTC), domain.Series{Slug: "re-zero", Name: "Re:Zero"}, []domain.RankedWaifu{{WaifuSummary: domain.WaifuSummary{Slug: "rem", Name: "Rem"}, Position: 1, Stars: 5}, {WaifuSummary: domain.WaifuSummary{Slug: "ram", Name: "Ram"}, Position: 2, Stars: 5}})
	testWotdDay = time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
)

func TestNotifications_BannerAlertsAndDedupe(t *testing.T) {
	f := newNotifyFixture(t)
	f.fav(alice, domain.FavoriteWaifu, "rem", "Rem")
	f.fav(alice, domain.FavoriteWaifu, "ram", "Ram")
	f.fav(aliceOther, domain.FavoriteWaifu, "rem", "Rem")
	f.fav(bob, domain.FavoriteSeries, "re-zero", "Re:Zero")
	f.fav(carol, domain.FavoriteWaifu, "emilia", "Emilia")

	f.svc.BannerPicked(f.ctx, testBanner)
	if got := f.notifier.sent["alice"]; len(got) != 1 || !strings.Contains(got[0], "features your favorites: Ram, Rem.") || !strings.Contains(got[0], "/favs alerts off") {
		t.Errorf("alice = %q", got)
	}
	if got := f.notifier.sent["bob"]; len(got) != 1 || !strings.HasPrefix(got[0], "Your favorite series **Re:Zero** is this week's banner!") {
		t.Errorf("bob = %q", got)
	}
	if len(f.notifier.sent["carol"]) != 0 {
		t.Error("carol has no matching favorite and should not be messaged")
	}
	if len(f.sleeps) != 1 || f.sleeps[0] != 250*time.Millisecond {
		t.Errorf("messages should be spaced: %v", f.sleeps)
	}

	f.svc.BannerPicked(f.ctx, testBanner)
	if len(f.notifier.sent["alice"]) != 1 || len(f.notifier.sent["bob"]) != 1 {
		t.Error("the same banner must not alert twice")
	}
	rerolled := domain.NewBanner(testBanner.WeekStart, domain.Series{Slug: "konosuba", Name: "KonoSuba"}, []domain.RankedWaifu{{WaifuSummary: domain.WaifuSummary{Slug: "rem", Name: "Rem"}, Position: 1}})
	f.svc.BannerPicked(f.ctx, rerolled)
	if got := f.notifier.sent["alice"]; len(got) != 2 || !strings.Contains(got[1], "**KonoSuba**") {
		t.Errorf("a rerolled banner is a new event: %q", got)
	}
}

func TestNotifications_WotdAndRollAlerts(t *testing.T) {
	f := newNotifyFixture(t)
	f.fav(alice, domain.FavoriteWaifu, "rem", "Rem")
	f.fav(bob, domain.FavoriteWaifu, "rem", "Rem")
	f.fav(aliceOther, domain.FavoriteWaifu, "rem", "Rem")

	f.svc.WotdPicked(f.ctx, testWotdDay, domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if got := f.notifier.sent["alice"]; len(got) != 1 || !strings.HasPrefix(got[0], "**Rem**, one of your favorites, is today's Waifu of the Day!") {
		t.Errorf("wotd alert = %q", got)
	}
	f.svc.WotdPicked(f.ctx, testWotdDay, domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if len(f.notifier.sent["alice"]) != 1 {
		t.Error("same day and pick must not repeat")
	}

	f.svc.Rolled(f.ctx, bob, domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if got := f.notifier.sent["alice"]; len(got) != 2 || got[1] != "<@bob> just rolled **Rem**, one of your favorites in Test Server. Ask them for a trade with `/waifu trade`.\n\n"+"Turn these off with /favs alerts off." {
		t.Errorf("roll alert = %q", got)
	}
	if len(f.notifier.sent["bob"]) != 1 {
		t.Error("the roller must not be told about their own roll")
	}
	f.clock.Advance(time.Second)
	f.svc.Rolled(f.ctx, domain.PlayerKey{GuildID: "other", UserID: "dave"}, domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if got := f.notifier.sent["alice"]; len(got) != 3 || !strings.Contains(got[2], "<@dave> just rolled **Rem**, one of your favorites. Ask") {
		t.Errorf("roll alert in another guild without a name = %q", got)
	}
	f.svc.Rolled(f.ctx, bob, domain.WaifuSummary{Slug: "nobody-likes", Name: "X"})
}

func TestNotifications_SettingsAndClosedDMs(t *testing.T) {
	f := newNotifyFixture(t)
	f.fav(alice, domain.FavoriteWaifu, "rem", "Rem")
	f.fav(bob, domain.FavoriteWaifu, "rem", "Rem")
	f.fav(carol, domain.FavoriteWaifu, "rem", "Rem")

	if st, err := f.svc.Setting(f.ctx, alice); err != nil || !st.Enabled || st.DMClosed {
		t.Fatalf("default setting = %+v %v", st, err)
	}
	if err := f.svc.SetEnabled(f.ctx, alice, false); err != nil {
		t.Fatal(err)
	}
	f.notifier.closed["bob"] = true
	f.notifier.fail["carol"] = errors.New("discord down")
	f.svc.WotdPicked(f.ctx, testWotdDay, domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if len(f.notifier.sent) != 0 {
		t.Errorf("nobody should have received a message: %+v", f.notifier.sent)
	}
	if st, _ := f.svc.Setting(f.ctx, bob); !st.DMClosed || !st.Enabled {
		t.Errorf("bob should be marked as having closed DMs: %+v", st)
	}
	if st, _ := f.svc.Setting(f.ctx, alice); st.Enabled {
		t.Error("alice should be disabled")
	}

	f.notifier.closed["bob"] = false
	f.svc.WotdPicked(f.ctx, testWotdDay.AddDate(0, 0, 1), domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if len(f.notifier.sent["bob"]) != 0 {
		t.Error("a user marked closed is skipped until they turn alerts back on")
	}
	if err := f.svc.SetEnabled(f.ctx, bob, true); err != nil {
		t.Fatal(err)
	}
	f.svc.WotdPicked(f.ctx, testWotdDay.AddDate(0, 0, 2), domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if len(f.notifier.sent["bob"]) != 1 {
		t.Error("turning alerts on clears the closed mark")
	}
	if err := f.svc.SetEnabled(f.ctx, alice, true); err != nil {
		t.Fatal(err)
	}
	f.svc.WotdPicked(f.ctx, testWotdDay.AddDate(0, 0, 3), domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if len(f.notifier.sent["alice"]) != 1 {
		t.Error("alice re-enabled should be messaged")
	}
	if len(f.notifier.sent["carol"]) != 0 {
		t.Error("carol's failing DM should not be recorded as sent")
	}

	ctx, cancel := context.WithCancel(f.ctx)
	cancel()
	f.svc.WotdPicked(ctx, testWotdDay.AddDate(0, 0, 4), domain.WaifuSummary{Slug: "rem", Name: "Rem"})
	if len(f.notifier.sent["alice"]) != 1 {
		t.Error("a cancelled context should stop delivery")
	}

	broken := app.NewNotificationService(failingFavorites{err: errors.New("boom")}, f.alerts, f.notifier, f.clock, nil, nil)
	broken.BannerPicked(f.ctx, testBanner)
	broken.WotdPicked(f.ctx, testWotdDay, domain.WaifuSummary{Slug: "rem"})
	broken.Rolled(f.ctx, alice, domain.WaifuSummary{Slug: "rem"})
	if len(f.notifier.sent["alice"]) != 1 {
		t.Error("lookup failures must not send anything")
	}
}

func TestNotifications_HooksFire(t *testing.T) {
	f := newFixture(t)
	f.ranking.Set(rankingOf(200))

	var picked []string
	wotd := f.wotd()
	wotd.OnPicked(func(day time.Time, w domain.WaifuSummary) { picked = append(picked, "wotd:"+w.Slug) })
	f.script(4)
	if _, err := wotd.Today(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := wotd.Today(f.ctx); err != nil {
		t.Fatal(err)
	}
	if len(picked) != 1 {
		t.Errorf("wotd hook should fire once per pick, got %v", picked)
	}

	banner := f.banner()
	banner.OnPicked(func(b domain.Banner) { picked = append(picked, "banner:"+b.Series.Slug) })
	f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	if _, err := banner.Ensure(f.ctx); err != nil {
		t.Fatal(err)
	}
	if len(picked) != 1 {
		t.Errorf("an existing banner must not fire the hook, got %v", picked)
	}

	roll := f.roll()
	roll.OnRolled(func(key domain.PlayerKey, w domain.WaifuSummary) {
		picked = append(picked, "roll:"+key.UserID+":"+w.Slug)
	})
	f.script(d100(50))
	f.source.EXPECT().Random(f.ctx).Return(summary("rem", 1, 0), nil).Once()
	f.source.EXPECT().Get(f.ctx, "rem").Return(detail("rem"), nil).Once()
	if _, err := roll.Roll(f.ctx, alice); err != nil {
		t.Fatal(err)
	}
	if len(picked) != 2 || picked[1] != "roll:alice:rem" {
		t.Errorf("roll hook = %v", picked)
	}
}

package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	alertFooter  = "Turn these off with /favs alerts off."
	alertSpacing = 250 * time.Millisecond
)

type NotificationService struct {
	favs      FavoriteStore
	alerts    AlertStore
	notifier  Notifier
	clock     Clock
	log       *slog.Logger
	sleep     func(context.Context, time.Duration) error
	guildName func(guildID string) string
	seq       atomic.Uint64
}

func NewNotificationService(favs FavoriteStore, alerts AlertStore, notifier Notifier, clock Clock, guildName func(string) string, log *slog.Logger) *NotificationService {
	if log == nil {
		log = slog.Default()
	}
	if guildName == nil {
		guildName = func(string) string { return "" }
	}
	return &NotificationService{favs: favs, alerts: alerts, notifier: notifier, clock: clock, log: log, sleep: sleepContext, guildName: guildName}
}

func (s *NotificationService) SetSleepForTest(fn func(context.Context, time.Duration) error) {
	s.sleep = fn
}

func (s *NotificationService) Setting(ctx context.Context, key domain.PlayerKey) (AlertSetting, error) {
	st, err := s.alerts.Setting(ctx, key)
	if err != nil {
		return AlertSetting{}, fmt.Errorf("alert setting: %w", err)
	}
	return st, nil
}

func (s *NotificationService) SetEnabled(ctx context.Context, key domain.PlayerKey, enabled bool) error {
	if err := s.alerts.SetEnabled(ctx, key, enabled, s.clock.Now()); err != nil {
		return fmt.Errorf("set alerts: %w", err)
	}
	if enabled {
		if err := s.alerts.ClearDMClosed(ctx, key.UserID); err != nil {
			return fmt.Errorf("reopen alerts: %w", err)
		}
	}
	return nil
}

type recipient struct {
	userID string
	guilds map[string]struct{}
	waifus []string
	series []string
}

func (s *NotificationService) BannerPicked(ctx context.Context, b domain.Banner) {
	event := fmt.Sprintf("banner:%d:%s", b.WeekStart.Unix(), b.Series.Slug)
	waifus, err := s.favs.Find(ctx, domain.FavoriteWaifu, b.Slugs(), "")
	if err != nil {
		s.log.Error("banner alert lookup failed", "err", err)
		return
	}
	series, err := s.favs.Find(ctx, domain.FavoriteSeries, []string{b.Series.Slug}, "")
	if err != nil {
		s.log.Error("banner alert lookup failed", "err", err)
		return
	}
	recipients := gather(waifus, series)
	s.deliver(ctx, event, recipients, func(r *recipient) string {
		var lines []string
		if len(r.series) > 0 {
			lines = append(lines, fmt.Sprintf("Your favorite series **%s** is this week's banner!", b.Series.Name))
		}
		if len(r.waifus) > 0 {
			lines = append(lines, fmt.Sprintf("This week's banner is **%s** and it features your favorites: %s.", b.Series.Name, joinNames(r.waifus)))
		}
		lines = append(lines, "Roll on it with `/waifu roll banner:true`.")
		return strings.Join(lines, "\n")
	})
}

func (s *NotificationService) WotdPicked(ctx context.Context, day time.Time, w domain.WaifuSummary) {
	event := fmt.Sprintf("wotd:%s:%s", day.Format("2006-01-02"), w.Slug)
	matches, err := s.favs.Find(ctx, domain.FavoriteWaifu, []string{w.Slug}, "")
	if err != nil {
		s.log.Error("wotd alert lookup failed", "err", err)
		return
	}
	s.deliver(ctx, event, gather(matches, nil), func(r *recipient) string {
		return fmt.Sprintf("**%s**, one of your favorites, is today's Waifu of the Day! A roll of 1 on the d100 pulls her today.", w.Name)
	})
}

func (s *NotificationService) Rolled(ctx context.Context, roller domain.PlayerKey, w domain.WaifuSummary) {
	event := fmt.Sprintf("roll:%s:%s:%s:%d:%d", roller.GuildID, roller.UserID, w.Slug, s.clock.Now().UnixNano(), s.seq.Add(1))
	matches, err := s.favs.Find(ctx, domain.FavoriteWaifu, []string{w.Slug}, roller.GuildID)
	if err != nil {
		s.log.Error("roll alert lookup failed", "err", err)
		return
	}
	kept := matches[:0]
	for _, m := range matches {
		if m.Key.UserID != roller.UserID {
			kept = append(kept, m)
		}
	}
	server := s.guildName(roller.GuildID)
	where := ""
	if server != "" {
		where = " in " + server
	}
	s.deliver(ctx, event, gather(kept, nil), func(r *recipient) string {
		return fmt.Sprintf("<@%s> just rolled **%s**, one of your favorites%s. Ask them for a trade with `/waifu trade`.", roller.UserID, w.Name, where)
	})
}

func gather(waifus, series []FavoriteMatch) []*recipient {
	byUser := map[string]*recipient{}
	get := func(m FavoriteMatch) *recipient {
		r, ok := byUser[m.Key.UserID]
		if !ok {
			r = &recipient{userID: m.Key.UserID, guilds: map[string]struct{}{}}
			byUser[m.Key.UserID] = r
		}
		r.guilds[m.Key.GuildID] = struct{}{}
		return r
	}
	for _, m := range waifus {
		r := get(m)
		if !contains(r.waifus, m.Favorite.Name) {
			r.waifus = append(r.waifus, m.Favorite.Name)
		}
	}
	for _, m := range series {
		r := get(m)
		r.series = append(r.series, m.Favorite.Name)
	}
	out := make([]*recipient, 0, len(byUser))
	for _, r := range byUser {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].userID < out[j].userID })
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func joinNames(names []string) string {
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func (s *NotificationService) deliver(ctx context.Context, event string, recipients []*recipient, compose func(*recipient) string) {
	sent := 0
	for _, r := range recipients {
		if ctx.Err() != nil {
			return
		}
		if !s.wants(ctx, r) {
			continue
		}
		first, err := s.alerts.MarkSent(ctx, event, r.userID, s.clock.Now())
		if err != nil {
			s.log.Error("alert dedupe failed", "event", event, "user", r.userID, "err", err)
			continue
		}
		if !first {
			continue
		}
		if sent > 0 {
			if err := s.sleep(ctx, alertSpacing); err != nil {
				return
			}
		}
		sent++
		err = s.notifier.DirectMessage(ctx, r.userID, compose(r)+"\n\n"+alertFooter)
		switch {
		case errors.Is(err, ErrDMClosed):
			s.log.Info("alert dm closed", "user", r.userID)
			if err := s.alerts.MarkDMClosed(ctx, r.userID, s.clock.Now()); err != nil {
				s.log.Error("marking dm closed failed", "user", r.userID, "err", err)
			}
		case err != nil:
			s.log.Warn("alert dm failed", "event", event, "user", r.userID, "err", err)
		}
	}
	if sent > 0 {
		s.log.Info("alerts sent", "event", event, "count", sent)
	}
}

func (s *NotificationService) wants(ctx context.Context, r *recipient) bool {
	for guildID := range r.guilds {
		st, err := s.alerts.Setting(ctx, domain.PlayerKey{GuildID: guildID, UserID: r.userID})
		if err != nil {
			s.log.Error("alert setting lookup failed", "user", r.userID, "err", err)
			continue
		}
		if st.DMClosed {
			return false
		}
		if st.Enabled {
			return true
		}
	}
	return false
}

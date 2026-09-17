package discord

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	commandFavs  = "favs"
	subFavWaifus = "waifus"
	subFavSeries = "series"
	subFavAlerts = "alerts"
	optState     = "state"

	favPrefix             = "fav:"
	mwlSeriesPathPrefix   = "/series/"
	alertTimeout          = 10 * time.Minute
	rankingRefreshTimeout = 2 * time.Hour
)

func favButton(kind domain.FavoriteKind, favorited bool) discordgo.Button {
	if favorited {
		return discordgo.Button{Style: discordgo.SecondaryButton, CustomID: favPrefix + string(kind), Emoji: &discordgo.ComponentEmoji{Name: "💔"}, Label: "Unfavorite"}
	}
	return discordgo.Button{Style: discordgo.SecondaryButton, CustomID: favPrefix + string(kind), Emoji: &discordgo.ComponentEmoji{Name: "🤍"}, Label: "Favorite"}
}

func cardActions(sellable, favorited bool) discordgo.ActionsRow {
	row := discordgo.ActionsRow{}
	if sellable {
		row.Components = append(row.Components, sellAskButton())
	}
	row.Components = append(row.Components, favButton(domain.FavoriteWaifu, favorited))
	return row
}

func (b *Bot) favorited(ctx context.Context, key domain.PlayerKey, kind domain.FavoriteKind, slug string) bool {
	if key.GuildID == "" || b.svc.Favorites == nil {
		return false
	}
	has, err := b.svc.Favorites.Has(ctx, key, kind, slug)
	if err != nil {
		b.log.Warn("favorite lookup failed", "slug", slug, "err", err)
		return false
	}
	return has
}

func swapFavButton(components []discordgo.MessageComponent, kind domain.FavoriteKind, favorited bool) []discordgo.MessageComponent {
	out := make([]discordgo.MessageComponent, 0, len(components))
	for _, c := range components {
		var inner []discordgo.MessageComponent
		switch row := c.(type) {
		case *discordgo.ActionsRow:
			inner = row.Components
		case discordgo.ActionsRow:
			inner = row.Components
		default:
			out = append(out, c)
			continue
		}
		swapped := make([]discordgo.MessageComponent, 0, len(inner))
		for _, comp := range inner {
			if buttonID(comp) == favPrefix+string(kind) {
				swapped = append(swapped, favButton(kind, favorited))
				continue
			}
			swapped = append(swapped, comp)
		}
		out = append(out, discordgo.ActionsRow{Components: swapped})
	}
	return out
}

func SeriesSlugFromEmbeds(embeds []*discordgo.MessageEmbed) (string, bool) {
	for _, e := range embeds {
		if e == nil || e.URL == "" {
			continue
		}
		u, err := url.Parse(e.URL)
		if err != nil || !strings.HasSuffix(strings.ToLower(u.Hostname()), "mywaifulist.moe") || !strings.HasPrefix(u.Path, mwlSeriesPathPrefix) {
			continue
		}
		slug := strings.Trim(strings.TrimPrefix(u.Path, mwlSeriesPathPrefix), "/")
		if slug != "" && !strings.Contains(slug, "/") {
			return slug, true
		}
	}
	return "", false
}

func embedTitleName(embeds []*discordgo.MessageEmbed) string {
	for _, e := range embeds {
		if e == nil || e.Title == "" {
			continue
		}
		lines := strings.Split(e.Title, "\n")
		name := strings.TrimSpace(strings.TrimPrefix(lines[len(lines)-1], "🔞"))
		if name != "" {
			return name
		}
	}
	return ""
}

func (b *Bot) favoriteButton(ctx context.Context, ic *interaction, raw string) {
	kind := domain.FavoriteKind(raw)
	if !kind.Valid() {
		b.log.Warn("unknown favorite kind", "kind", raw)
		return
	}
	if ic.GuildID == "" {
		b.replyEphemeral(ic, fmt.Sprintf(msgDMFmt, commandFavs))
		return
	}
	if ic.Message == nil || ic.Message.Author == nil || ic.Message.Author.ID != b.cfg.BotUserID {
		b.replyEphemeral(ic, msgNotAWaifu)
		return
	}
	fav, ok := b.favoriteFromMessage(ctx, kind, ic.Message)
	if !ok {
		b.replyEphemeral(ic, msgNotAWaifu)
		return
	}
	added, err := b.svc.Favorites.Toggle(ctx, ic.key(), fav)
	if err != nil {
		b.log.Error("favorite toggle failed", "user", ic.userID(), "err", err)
		b.replyEphemeral(ic, errorText(err))
		return
	}
	text := fmt.Sprintf(msgFavRemovedFmt, fav.Name)
	if added {
		text = fmt.Sprintf(msgFavAddedFmt, fav.Name)
	}
	components := swapFavButton(ic.Message.Components, kind, added)
	err = b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: ic.Message.Content, Embeds: ic.Message.Embeds, Components: components},
	})
	if err != nil {
		b.log.Warn("favorite button update failed", "err", err)
		b.replyEphemeral(ic, text)
		return
	}
	if _, err := b.s.FollowupMessageCreate(ic.Interaction, true, &discordgo.WebhookParams{Content: text, Flags: discordgo.MessageFlagsEphemeral}); err != nil {
		b.log.Warn("favorite followup failed", "err", err)
	}
}

func (b *Bot) favoriteFromMessage(ctx context.Context, kind domain.FavoriteKind, msg *discordgo.Message) (domain.Favorite, bool) {
	fav := domain.Favorite{Kind: kind, Name: embedTitleName(msg.Embeds)}
	switch kind {
	case domain.FavoriteWaifu:
		slug, ok := SlugFromEmbeds(msg.Embeds)
		if !ok {
			return domain.Favorite{}, false
		}
		fav.Slug, fav.URL = slug, waifuURL(slug)
		if w, err := b.svc.Search.Detail(ctx, slug); err == nil {
			fav.Name, fav.PictureURL = w.Name, w.PictureURL
			if w.URL != "" {
				fav.URL = w.URL
			}
		}
	case domain.FavoriteSeries:
		slug, ok := SeriesSlugFromEmbeds(msg.Embeds)
		if !ok {
			return domain.Favorite{}, false
		}
		fav.Slug = slug
		if s, err := b.svc.Search.SeriesDetail(ctx, slug); err == nil {
			fav.Name, fav.URL, fav.PictureURL = s.Name, s.URL, s.PictureURL
		}
	}
	if fav.Name == "" {
		fav.Name = fav.Slug
	}
	return fav, true
}

func (b *Bot) favs(ctx context.Context, ic *interaction, sub string, opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	kind := domain.FavoriteWaifu
	if sub == subFavSeries {
		kind = domain.FavoriteSeries
	}
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	key := ic.key()
	whose := "Your"
	if target := resolvedUser(opts, resolved, optUser); target != nil && target.ID != ic.userID() {
		key.UserID = target.ID
		whose = mention(target.ID) + "'s"
	}
	favs, err := b.svc.Favorites.List(ctx, key, kind)
	if err != nil {
		b.failed(ic, "favs", err)
		return
	}
	noun := "waifus"
	if kind == domain.FavoriteSeries {
		noun = "series"
	}
	if len(favs) == 0 {
		b.editText(ic, fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgFavNoneFmt, whose, noun)))
		return
	}
	content := fmt.Sprintf("%s %s favorite %s · %s", mention(ic.userID()), whose, noun, thousands(len(favs)))
	if kind == domain.FavoriteSeries {
		if stringOption(opts, optView) == viewCards {
			series := make([]domain.Series, len(favs))
			for i, f := range favs {
				series[i] = domain.Series{Slug: f.Slug, Name: f.Name, URL: f.URL, PictureURL: f.PictureURL}
			}
			b.openPager(ctx, ic, content, pagesFromSeries(series), false, b.cfg.Clock.Now().Sub(start))
			return
		}
		lines := make([]string, len(favs))
		for i, f := range favs {
			lines[i] = fmt.Sprintf("%d. %s", i+1, f.Name)
		}
		b.openPager(ctx, ic, content, pagesFromLines(lines, "Favorite series"), false, b.cfg.Clock.Now().Sub(start))
		return
	}
	summaries := make([]domain.WaifuSummary, len(favs))
	for i, f := range favs {
		summaries[i] = domain.WaifuSummary{Slug: f.Slug, Name: f.Name, PictureURL: f.PictureURL}
	}
	if stringOption(opts, optView) == viewCards {
		b.openPager(ctx, ic, content, pagesFromSummaries(summaries), false, b.cfg.Clock.Now().Sub(start))
		return
	}
	b.openPager(ctx, ic, content, pagesCompactFromSummaries(summaries, app.LookupFrom(b.svc.Ranking), "Favorite waifus"), false, b.cfg.Clock.Now().Sub(start))
}

func (b *Bot) favAlerts(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if b.svc.Notify == nil {
		b.replyEphemeral(ic, msgUnexpected)
		return
	}
	state := stringOption(opts, optState)
	if state != "on" && state != "off" {
		b.replyEphemeral(ic, msgUnexpected)
		return
	}
	if err := b.svc.Notify.SetEnabled(ctx, ic.key(), state == "on"); err != nil {
		b.log.Error("set alerts failed", "user", ic.userID(), "err", err)
		b.replyEphemeral(ic, errorText(err))
		return
	}
	if state == "off" {
		b.replyEphemeral(ic, msgAlertsOff)
		return
	}
	b.replyEphemeral(ic, msgAlertsOn)
}

func (b *Bot) DirectMessage(ctx context.Context, userID, content string) error {
	ch, err := b.s.UserChannelCreate(userID)
	if err != nil {
		return dmError(err)
	}
	if _, err := b.s.ChannelMessageSend(ch.ID, content); err != nil {
		return dmError(err)
	}
	return nil
}

func dmError(err error) error {
	var rest *discordgo.RESTError
	if errors.As(err, &rest) && rest.Message != nil && rest.Message.Code == discordgo.ErrCodeCannotSendMessagesToThisUser {
		return app.ErrDMClosed
	}
	return err
}

func (b *Bot) SetNotifier(n *app.NotificationService) {
	b.svc.Notify = n
}

func (b *Bot) WireAlerts(roll *app.RollService, wotd *app.WotdService, banner *app.BannerService) {
	if b.svc.Notify == nil {
		return
	}
	if roll != nil {
		roll.OnRolled(func(key domain.PlayerKey, w domain.WaifuSummary) {
			b.background(func(ctx context.Context) { b.svc.Notify.Rolled(ctx, key, w) })
		})
	}
	if wotd != nil {
		wotd.OnPicked(func(day time.Time, w domain.WaifuSummary) {
			b.background(func(ctx context.Context) { b.svc.Notify.WotdPicked(ctx, day, w) })
		})
	}
	if banner != nil {
		banner.OnPicked(func(bn domain.Banner) {
			b.background(func(ctx context.Context) { b.svc.Notify.BannerPicked(ctx, bn) })
		})
	}
}

func (b *Bot) background(fn func(context.Context)) {
	b.backgroundFor(alertTimeout, fn)
}

func (b *Bot) backgroundFor(timeout time.Duration, fn func(context.Context)) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		fn(ctx)
	}()
}

func (b *Bot) WaitBackground() {
	b.wg.Wait()
}

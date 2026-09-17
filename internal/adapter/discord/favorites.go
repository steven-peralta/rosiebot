package discord

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	commandFavs  = "favs"
	subFavWaifus = "waifus"
	subFavSeries = "series"

	favPrefix           = "fav:"
	mwlSeriesPathPrefix = "/series/"
)

func favButton(kind domain.FavoriteKind) discordgo.Button {
	return discordgo.Button{Style: discordgo.SecondaryButton, CustomID: favPrefix + string(kind), Emoji: &discordgo.ComponentEmoji{Name: "🤍"}, Label: "Favorite"}
}

func cardActions(sellable bool) discordgo.ActionsRow {
	row := discordgo.ActionsRow{}
	if sellable {
		row.Components = append(row.Components, sellAskButton())
	}
	row.Components = append(row.Components, favButton(domain.FavoriteWaifu))
	return row
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
	if added {
		b.replyEphemeral(ic, fmt.Sprintf(msgFavAddedFmt, fav.Name))
		return
	}
	b.replyEphemeral(ic, fmt.Sprintf(msgFavRemovedFmt, fav.Name))
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

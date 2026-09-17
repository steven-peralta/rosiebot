package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	commandSeries = "series"
	seriesPrefix  = "series:"
	seriesBrowse  = "browse"
)

func seriesCardComponents(slug string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.SecondaryButton, CustomID: seriesPrefix + seriesBrowse + ":" + slug, Emoji: &discordgo.ComponentEmoji{Name: "📖"}, Label: "Browse characters"},
	}}}
}

func (b *Bot) seriesSearch(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	input := stringOption(opts, optQuery)
	var (
		res app.SeriesResult
		err error
	)
	if slug, ok := directSlug(input); ok {
		res, err = b.svc.Search.SeriesBySlug(ctx, slug, app.Query{})
	} else {
		res, err = b.svc.Search.Series(ctx, input, app.Query{})
	}
	switch {
	case errors.Is(err, app.ErrNotFound):
		b.editText(ic, mention(ic.userID())+" "+msgSeriesNotFound)
		return
	case err != nil:
		b.failed(ic, "series search", err)
		return
	}
	e := seriesCardEmbed(res.Series, res.Waifus, app.LookupFrom(b.svc.Ranking))
	e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
	var components []discordgo.MessageComponent
	if len(res.Waifus) > 0 {
		components = seriesCardComponents(res.Series.Slug)
	}
	b.edit(ic, mention(ic.userID())+" "+msgSeriesFound, []*discordgo.MessageEmbed{e}, components)
}

func (b *Bot) seriesButton(ctx context.Context, ic *interaction, action string) {
	kind, slug, _ := strings.Cut(action, ":")
	if kind != seriesBrowse || slug == "" {
		b.log.Warn("unknown series action", "action", action)
		return
	}
	if !b.deferReply(ic, false) {
		return
	}
	b.searchWithinSeries(ctx, ic, slugChoicePrefix+slug, app.Query{}, b.cfg.Clock.Now())
}

func seriesCardEmbed(s domain.Series, waifus []domain.WaifuSummary, lookup app.RankLookup) *discordgo.MessageEmbed {
	e := seriesEmbed(s)
	if len(waifus) == 0 {
		return e
	}
	ranked := 0
	lines := make([]string, 0, min(len(waifus), domain.BannerCardLimit)+1)
	for i, w := range waifus {
		r, ok := lookup(w.Slug)
		if ok {
			ranked++
		}
		if i >= domain.BannerCardLimit {
			continue
		}
		if ok && r.Stars > 0 {
			lines = append(lines, fmt.Sprintf("%s %s · Rank #%s", strings.Repeat(":star:", r.Stars), w.Name, thousands(r.Position)))
		} else {
			lines = append(lines, w.Name+" · unranked")
		}
	}
	if extra := len(waifus) - domain.BannerCardLimit; extra > 0 {
		lines = append(lines, fmt.Sprintf("+%d more", extra))
	}
	name := fmt.Sprintf("Characters · %d ranked of %d", ranked, len(waifus))
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: name, Value: strings.Join(lines, "\n")})
	return e
}

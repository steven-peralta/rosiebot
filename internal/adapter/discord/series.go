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
	commandSeries   = "series"
	seriesPrefix    = "series:"
	seriesBrowse    = "browse"
	viewWaifuMenu   = "view:waifu"
	viewMenuLimit   = 25
	viewPlaceholder = "View a character's card"
)

type cardCharacter struct {
	summary domain.WaifuSummary
	ranked  *domain.RankedWaifu
}

func cardLine(c cardCharacter) string {
	if c.ranked != nil && c.ranked.Stars > 0 {
		return fmt.Sprintf("%s %s · Rank #%s", strings.Repeat(":star:", c.ranked.Stars), c.summary.Name, thousands(c.ranked.Position))
	}
	return c.summary.Name + " · unranked"
}

func viewMenuRow(chars []cardCharacter) discordgo.MessageComponent {
	options := make([]discordgo.SelectMenuOption, 0, min(len(chars), viewMenuLimit))
	for _, c := range chars {
		if len(options) == viewMenuLimit {
			break
		}
		desc := "unranked"
		if c.ranked != nil && c.ranked.Stars > 0 {
			desc = strings.Repeat("⭐", c.ranked.Stars) + " · rank #" + thousands(c.ranked.Position)
		}
		options = append(options, discordgo.SelectMenuOption{Label: truncate(c.summary.Name, maxChoiceLength), Value: c.summary.Slug, Description: desc})
	}
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{
		MenuType: discordgo.StringSelectMenu, CustomID: viewWaifuMenu, Placeholder: viewPlaceholder, Options: options,
	}}}
}

func seriesCardComponents(slug string, chars []cardCharacter) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		viewMenuRow(chars),
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Style: discordgo.SecondaryButton, CustomID: seriesPrefix + seriesBrowse + ":" + slug, Emoji: &discordgo.ComponentEmoji{Name: "📖"}, Label: "Browse characters"},
		}},
	}
}

func (b *Bot) viewWaifu(ctx context.Context, ic *interaction) {
	values := ic.MessageComponentData().Values
	if len(values) != 1 || values[0] == "" {
		b.log.Warn("view menu without a selection")
		return
	}
	if !b.deferReply(ic, true) {
		return
	}
	start := b.cfg.Clock.Now()
	w, err := b.svc.Search.Detail(ctx, values[0])
	if err != nil {
		b.failed(ic, "view waifu", err)
		return
	}
	b.edit(ic, "", []*discordgo.MessageEmbed{b.waifuEmbed(w, b.cfg.Clock.Now().Sub(start))}, nil)
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
	chars := cardCharacters(res.Waifus, app.LookupFrom(b.svc.Ranking))
	e := seriesCardEmbed(res.Series, chars)
	e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
	var components []discordgo.MessageComponent
	if len(chars) > 0 {
		components = seriesCardComponents(res.Series.Slug, chars)
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

func cardCharacters(waifus []domain.WaifuSummary, lookup app.RankLookup) []cardCharacter {
	out := make([]cardCharacter, len(waifus))
	for i, w := range waifus {
		out[i] = cardCharacter{summary: w}
		if r, ok := lookup(w.Slug); ok {
			out[i].ranked = &r
		}
	}
	return out
}

func cardLines(chars []cardCharacter) string {
	lines := make([]string, 0, min(len(chars), domain.BannerCardLimit)+1)
	for i, c := range chars {
		if i == domain.BannerCardLimit {
			lines = append(lines, fmt.Sprintf("+%d more", len(chars)-domain.BannerCardLimit))
			break
		}
		lines = append(lines, cardLine(c))
	}
	return strings.Join(lines, "\n")
}

func seriesCardEmbed(s domain.Series, chars []cardCharacter) *discordgo.MessageEmbed {
	e := seriesEmbed(s)
	if len(chars) == 0 {
		return e
	}
	ranked := 0
	for _, c := range chars {
		if c.ranked != nil {
			ranked++
		}
	}
	name := fmt.Sprintf("Characters · %d ranked of %d", ranked, len(chars))
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: name, Value: cardLines(chars)})
	return e
}

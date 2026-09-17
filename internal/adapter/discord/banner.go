package discord

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	bannerPrefix = "banner:"
	bannerRoll   = "roll"
	rollBanner   = "banner"
)

func bannerCardComponents(bn domain.Banner) []discordgo.MessageComponent {
	rows := []discordgo.MessageComponent{}
	if chars := bannerCharacters(bn); len(chars) > 0 {
		rows = append(rows, viewMenuRow(chars))
	}
	return append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.PrimaryButton, CustomID: bannerPrefix + bannerRoll, Emoji: &discordgo.ComponentEmoji{Name: "🎟️"}, Label: fmt.Sprintf("Roll on banner · %s coins", thousands(domain.BannerRollCost))},
	}})
}

func bannerCharacters(bn domain.Banner) []cardCharacter {
	out := make([]cardCharacter, len(bn.Characters))
	for i := range bn.Characters {
		out[i] = cardCharacter{summary: bn.Characters[i].WaifuSummary, ranked: &bn.Characters[i]}
	}
	return out
}

func bannerAgainComponents(userID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.PrimaryButton, CustomID: rollPrefix + rollBanner + ":" + userID, Emoji: &discordgo.ComponentEmoji{Name: "🎟️"}, Label: fmt.Sprintf("Roll on banner again · %s coins", thousands(domain.BannerRollCost))},
	}}}
}

func againComponents(res app.RollResult, userID string) []discordgo.MessageComponent {
	rows := rollAgainComponents(userID)
	if res.Banner {
		rows = bannerAgainComponents(userID)
	}
	row := rows[0].(discordgo.ActionsRow)
	row.Components = append(row.Components, sellAskButton(), favButton(domain.FavoriteWaifu))
	return []discordgo.MessageComponent{row}
}

func (b *Bot) banner(ctx context.Context, ic *interaction) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	res, err := b.svc.Banner.Current(ctx)
	if err != nil {
		if errors.Is(err, app.ErrNoBanner) {
			b.editText(ic, mention(ic.userID())+" "+msgNoBanner)
			return
		}
		b.failed(ic, "banner", err)
		return
	}
	content := fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgBannerFmt, domain.FormatCountdown(res.RefreshIn)))
	e := bannerEmbed(res.Banner)
	e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
	b.edit(ic, content, []*discordgo.MessageEmbed{e}, bannerCardComponents(res.Banner))
}

func (b *Bot) bannerButton(ctx context.Context, ic *interaction, action string) {
	if action != bannerRoll {
		b.log.Warn("unknown banner action", "action", action)
		return
	}
	if ic.GuildID == "" {
		b.replyEphemeral(ic, fmt.Sprintf(msgDMFmt, subRoll))
		return
	}
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	res, err := b.svc.Roll.RollBanner(ctx, ic.key())
	if err != nil {
		b.failed(ic, "banner roll", err)
		return
	}
	content, embed := b.rollResult(ic.userID(), res, b.cfg.Clock.Now().Sub(start))
	b.edit(ic, content, []*discordgo.MessageEmbed{embed}, againComponents(res, ic.userID()))
}

func bannerEmbed(bn domain.Banner) *discordgo.MessageEmbed {
	e := seriesEmbed(bn.Series)
	if len(bn.Characters) > 0 {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Featured this week", Value: cardLines(bannerCharacters(bn))})
	}
	return e
}

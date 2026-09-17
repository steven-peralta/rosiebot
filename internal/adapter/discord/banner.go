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
	bannerPrefix = "banner:"
	bannerRoll   = "roll"
	rollBanner   = "banner"
)

func bannerCardComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.PrimaryButton, CustomID: bannerPrefix + bannerRoll, Emoji: &discordgo.ComponentEmoji{Name: "🎟️"}, Label: fmt.Sprintf("Roll on banner · %d coins", domain.BannerRollCost)},
	}}}
}

func bannerAgainComponents(userID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.PrimaryButton, CustomID: rollPrefix + rollBanner + ":" + userID, Emoji: &discordgo.ComponentEmoji{Name: "🎟️"}, Label: fmt.Sprintf("Roll on banner again · %d coins", domain.BannerRollCost)},
	}}}
}

func againComponents(res app.RollResult, userID string) []discordgo.MessageComponent {
	if res.Banner {
		return bannerAgainComponents(userID)
	}
	return rollAgainComponents(userID)
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
	b.edit(ic, content, []*discordgo.MessageEmbed{e}, bannerCardComponents())
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
	lines := make([]string, 0, len(bn.Characters))
	for i, c := range bn.Characters {
		if i == domain.BannerCardLimit {
			lines = append(lines, fmt.Sprintf("+%d more", len(bn.Characters)-domain.BannerCardLimit))
			break
		}
		lines = append(lines, fmt.Sprintf("%s %s · Rank #%s", strings.Repeat(":star:", c.Stars), c.Name, thousands(c.Position)))
	}
	if len(lines) > 0 {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Featured this week", Value: strings.Join(lines, "\n")})
	}
	return e
}

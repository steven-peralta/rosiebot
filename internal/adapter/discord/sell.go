package discord

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	sellPrefix = "sell:"
	sellAsk    = "ask"
	sellOK     = "ok"
	sellNo     = "no"
	sellAll    = "all"
)

func sellAskButton() discordgo.Button {
	return discordgo.Button{Style: discordgo.SecondaryButton, CustomID: sellPrefix + sellAsk, Emoji: &discordgo.ComponentEmoji{Name: "💰"}, Label: "Sell"}
}

func sellConfirmComponents(channelID, messageID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.SuccessButton, CustomID: sellPrefix + sellOK + ":" + channelID + ":" + messageID, Emoji: &discordgo.ComponentEmoji{Name: "✅"}, Label: "Sell"},
		discordgo.Button{Style: discordgo.DangerButton, CustomID: sellPrefix + sellNo, Emoji: &discordgo.ComponentEmoji{Name: "🚫"}, Label: "Cancel"},
	}}}
}

func (b *Bot) sellContext(ctx context.Context, ic *interaction, data discordgo.ApplicationCommandInteractionData) {
	var target *discordgo.Message
	if data.Resolved != nil {
		target = data.Resolved.Messages[data.TargetID]
	}
	b.offerSale(ctx, ic, target)
}

func (b *Bot) offerSale(ctx context.Context, ic *interaction, target *discordgo.Message) {
	if ic.GuildID == "" {
		b.replyEphemeral(ic, fmt.Sprintf(msgDMFmt, "sell"))
		return
	}
	slug, ok := b.slugFromMessage(target)
	if !ok {
		b.replyEphemeral(ic, msgNotAWaifu)
		return
	}
	owned, has, err := b.svc.Inventory.Owns(ctx, ic.key(), slug)
	if err != nil {
		b.replyEphemeral(ic, msgUnexpected)
		return
	}
	if !has {
		b.replyEphemeral(ic, fmt.Sprintf(msgNotOwnedFmt, slug))
		return
	}
	price, _ := b.svc.Inventory.Price(slug)
	b.replyEphemeralComponents(ic, fmt.Sprintf(msgSellConfirmFmt, owned.Name, coins(price)), sellConfirmComponents(target.ChannelID, target.ID))
}

func sellAllChoices() []*discordgo.ApplicationCommandOptionChoice {
	out := []*discordgo.ApplicationCommandOptionChoice{{Name: "Unranked only", Value: 0}}
	for n := 1; n <= domain.MaxStars; n++ {
		label := strings.Repeat("⭐", n) + " and below"
		if n == domain.MaxStars {
			label = "Everything"
		}
		out = append(out, &discordgo.ApplicationCommandOptionChoice{Name: label, Value: n})
	}
	return out
}

func sellAllComponents(maxStars int) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.DangerButton, CustomID: fmt.Sprintf("%s%s:%d", sellPrefix, sellAll, maxStars), Emoji: &discordgo.ComponentEmoji{Name: "💰"}, Label: "Sell them all"},
		discordgo.Button{Style: discordgo.SecondaryButton, CustomID: sellPrefix + sellNo, Emoji: &discordgo.ComponentEmoji{Name: "🚫"}, Label: "Cancel"},
	}}}
}

func tierBreakdown(q app.BulkQuote) string {
	parts := []string{}
	if q.Tiers[0] > 0 {
		parts = append(parts, thousands(q.Tiers[0])+" unranked")
	}
	for n := 1; n <= domain.MaxStars; n++ {
		if q.Tiers[n] > 0 {
			parts = append(parts, thousands(q.Tiers[n])+" "+strings.Repeat("⭐", n))
		}
	}
	return strings.Join(parts, ", ")
}

func (b *Bot) sellAllCommand(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	maxStars, ok := intOption(opts, optMaxStars)
	if !ok || maxStars < 0 || maxStars > domain.MaxStars {
		b.replyEphemeral(ic, msgUnexpected)
		return
	}
	q, err := b.svc.Inventory.QuoteBelow(ctx, ic.key(), maxStars)
	if err != nil {
		b.log.Error("sell all quote failed", "user", ic.userID(), "err", err)
		b.replyEphemeral(ic, errorText(err))
		return
	}
	if q.Count == 0 {
		b.replyEphemeral(ic, msgSellAllNone)
		return
	}
	b.replyEphemeralComponents(ic, fmt.Sprintf(msgSellAllConfirmFmt, thousands(q.Count), tierBreakdown(q), coins(q.Total)), sellAllComponents(maxStars))
}

func (b *Bot) sellAllConfirm(ctx context.Context, ic *interaction, raw string) {
	maxStars, err := strconv.Atoi(raw)
	if err != nil || maxStars < 0 || maxStars > domain.MaxStars {
		b.log.Warn("bad sell all threshold", "raw", raw)
		b.updateMessage(ic, msgUnexpected, nil, nil)
		return
	}
	res, err := b.svc.Inventory.SellBelow(ctx, ic.key(), maxStars)
	if err != nil {
		b.log.Error("sell all failed", "user", ic.userID(), "err", err)
		b.updateMessage(ic, errorText(err), nil, nil)
		return
	}
	if res.Count == 0 {
		b.updateMessage(ic, msgSellAllNone, nil, nil)
		return
	}
	b.log.Info("sold collection tier", "user", ic.userID(), "guild", ic.GuildID, "max_stars", maxStars, "count", res.Count, "total", res.Total)
	b.updateMessage(ic, fmt.Sprintf(msgSellAllDoneFmt, thousands(res.Count), coins(res.Total), coins(res.Balance)), nil, nil)
}

func (b *Bot) slugFromMessage(msg *discordgo.Message) (string, bool) {
	if msg == nil || msg.Author == nil || msg.Author.ID != b.cfg.BotUserID {
		return "", false
	}
	return SlugFromEmbeds(msg.Embeds)
}

func (b *Bot) sellButton(ctx context.Context, ic *interaction, action string) {
	switch action {
	case sellAsk:
		b.offerSale(ctx, ic, ic.Message)
		return
	case sellNo:
		b.updateMessage(ic, msgSellCancelled, nil, nil)
		return
	}
	if rest, ok := strings.CutPrefix(action, sellAll+":"); ok {
		b.sellAllConfirm(ctx, ic, rest)
		return
	}
	parts := strings.Split(action, ":")
	if len(parts) != 3 || parts[0] != sellOK {
		b.log.Warn("unknown sell action", "action", action)
		return
	}
	channelID, messageID := parts[1], parts[2]
	parent, err := b.s.ChannelMessage(channelID, messageID)
	if err != nil {
		b.updateMessage(ic, msgNotAWaifu, nil, nil)
		return
	}
	slug, ok := b.slugFromMessage(parent)
	if !ok {
		b.updateMessage(ic, msgNotAWaifu, nil, nil)
		return
	}
	res, err := b.svc.Inventory.Sell(ctx, ic.key(), slug)
	if errors.Is(err, app.ErrNotOwned) {
		b.updateMessage(ic, fmt.Sprintf(msgNotOwnedFmt, slug), nil, nil)
		return
	}
	if err != nil {
		b.log.Error("sell failed", "err", err)
		b.updateMessage(ic, msgUnexpected, nil, nil)
		return
	}
	b.updateMessage(ic, fmt.Sprintf(msgSoldFmt, res.Waifu.Name, coins(res.Price), coins(res.Balance)), nil, nil)
	b.removeFromPager(ctx, channelID, messageID, slug)
}

func (b *Bot) removeFromPager(ctx context.Context, channelID, messageID, slug string) {
	s, ok := b.sessions.Get(messageID)
	if !ok || s.Kind != SessionPager {
		b.stripSellButton(channelID, messageID)
		return
	}
	kept := s.Pages[:0]
	for _, p := range s.Pages {
		if p.summary.Slug != slug {
			kept = append(kept, p)
		}
	}
	if len(kept) == len(s.Pages) {
		return
	}
	s.Pages = kept
	if len(s.Pages) == 0 {
		b.sessions.Delete(messageID)
		b.editChannelMessage(s.ChannelID, messageID, strings.TrimSpace(s.Content)+" "+msgOwnsNothing, nil, nil)
		return
	}
	content, embeds, components := b.renderPage(ctx, s, 0)
	b.editChannelMessage(s.ChannelID, messageID, content, embeds, components)
}

func (b *Bot) stripSellButton(channelID, messageID string) {
	msg, err := b.s.ChannelMessage(channelID, messageID)
	if err != nil || len(msg.Components) == 0 {
		return
	}
	b.editChannelMessage(channelID, messageID, strings.TrimSpace(msg.Content)+" (sold)", msg.Embeds, nil)
}

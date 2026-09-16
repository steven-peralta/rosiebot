package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
)

const (
	sellPrefix = "sell:"
	sellAsk    = "ask"
	sellOK     = "ok"
	sellNo     = "no"
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
	b.replyEphemeralComponents(ic, fmt.Sprintf(msgSellConfirmFmt, owned.Name, b.sellPrice()), sellConfirmComponents(target.ChannelID, target.ID))
}

func (b *Bot) sellPrice() int64 {
	return 100
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
	b.updateMessage(ic, fmt.Sprintf(msgSoldFmt, res.Waifu.Name, res.Price, res.Balance), nil, nil)
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

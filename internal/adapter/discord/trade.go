package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	tradePrefix     = "trade:"
	tradeAccept     = "accept"
	tradeCounter    = "counter"
	tradeDecline    = "decline"
	maxChoices      = 25
	maxChoiceLength = 100
	maxTradeItems   = 25
)

func splitSlugs(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) > maxTradeItems {
		out = out[:maxTradeItems]
	}
	return out
}

func tradeComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.SuccessButton, CustomID: tradePrefix + tradeAccept, Emoji: &discordgo.ComponentEmoji{Name: "✅"}, Label: "Accept"},
		discordgo.Button{Style: discordgo.PrimaryButton, CustomID: tradePrefix + tradeCounter, Emoji: &discordgo.ComponentEmoji{Name: "🔁"}, Label: "Counter"},
		discordgo.Button{Style: discordgo.DangerButton, CustomID: tradePrefix + tradeDecline, Emoji: &discordgo.ComponentEmoji{Name: "🚫"}, Label: "Decline"},
	}}}
}

func (b *Bot) trade(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	target := resolvedUser(opts, resolved, optUser)
	give, receive := splitSlugs(stringOption(opts, optGive)), splitSlugs(stringOption(opts, optReceive))
	builder := len(give)+len(receive) == 0
	respond := func(text string) {
		if builder {
			b.replyEphemeral(ic, text)
		} else {
			b.editText(ic, text)
		}
	}
	if !builder && !b.deferReply(ic, false) {
		return
	}
	if target == nil {
		respond(mention(ic.userID()) + " " + msgUserNotFound)
		return
	}
	if target.Bot {
		respond(mention(ic.userID()) + " You can't trade with a bot.")
		return
	}
	if target.ID == ic.userID() {
		respond(mention(ic.userID()) + " You can't trade with yourself.")
		return
	}
	if builder {
		b.openTradeBuilder(ctx, ic, target, domain.TradeOffer{}, "")
		return
	}
	offer := domain.TradeOffer{
		Sender:  ic.key(),
		Target:  domain.PlayerKey{GuildID: ic.GuildID, UserID: target.ID},
		Give:    give,
		Receive: receive,
	}
	proposal, err := b.svc.Trade.Propose(ctx, offer)
	if err != nil {
		var violation *domain.TradeViolation
		if errors.As(err, &violation) {
			b.editText(ic, fmt.Sprintf("%s, %s.", mention(ic.userID()), violationText(violation, mention(target.ID), "")))
			return
		}
		b.failed(ic, "trade", err)
		return
	}
	content := fmt.Sprintf(msgTradeOfferFmt, mention(target.ID), mention(ic.userID()))
	msg := b.edit(ic, content, []*discordgo.MessageEmbed{b.tradeEmbed(proposal.Give, proposal.Receive)}, tradeComponents())
	if msg == nil {
		return
	}
	b.sessions.Put(msg.ID, &Session{
		Kind:      SessionTrade,
		OwnerID:   target.ID,
		ChannelID: ic.ChannelID,
		Content:   content,
		Offer:     proposal.Offer,
		Give:      proposal.Give,
		Receive:   proposal.Receive,
		SenderID:  ic.userID(),
		TargetID:  target.ID,
	}, b.cfg.TradeTTL)
}

func (b *Bot) tradeButton(ctx context.Context, ic *interaction, action string) {
	if ic.Message == nil {
		return
	}
	s, ok := b.sessions.Get(ic.Message.ID)
	if !ok || s.Kind != SessionTrade {
		b.replyEphemeral(ic, msgTradeExpired)
		b.editChannelMessage(ic.ChannelID, ic.Message.ID, msgTradeExpired, nil, nil)
		return
	}
	user := ic.userID()
	switch action {
	case tradeCounter:
		if user != s.TargetID {
			b.replyEphemeral(ic, msgNotYourMenu)
			return
		}
		b.startCounter(ctx, ic, s)
	case tradeDecline:
		if user != s.TargetID && user != s.SenderID {
			b.replyEphemeral(ic, msgNotYourMenu)
			return
		}
		b.sessions.Delete(ic.Message.ID)
		b.updateMessage(ic, fmt.Sprintf("%s %s %s", mention(s.SenderID), mention(s.TargetID), msgTradeDenied), nil, nil)
	case tradeAccept:
		if user != s.TargetID {
			b.replyEphemeral(ic, msgNotYourMenu)
			return
		}
		if !s.beginAccept() {
			b.replyEphemeral(ic, "This trade is already being processed.")
			return
		}
		if err := b.svc.Trade.Accept(ctx, s.Offer); err != nil {
			s.reopen()
			var violation *domain.TradeViolation
			text := errorText(err)
			if errors.As(err, &violation) {
				text = msgTradeConflict + " " + violationText(violation, mention(s.TargetID), "") + "."
			}
			b.sessions.Delete(ic.Message.ID)
			b.updateMessage(ic, fmt.Sprintf("%s %s %s", mention(s.SenderID), mention(s.TargetID), text), nil, nil)
			return
		}
		s.finish()
		b.sessions.Delete(ic.Message.ID)
		b.updateMessage(ic, fmt.Sprintf("%s %s %s", mention(s.SenderID), mention(s.TargetID), msgTradeAccepted), nil, nil)
	default:
		b.log.Warn("unknown trade action", "action", action)
	}
}

func (b *Bot) tradeAutocomplete(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	var focused *discordgo.ApplicationCommandInteractionDataOption
	for _, o := range opts {
		if o.Focused {
			focused = o
			break
		}
	}
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	if focused != nil && ic.GuildID != "" {
		key := ic.key()
		if focused.Name == optReceive {
			if target := resolvedUser(opts, resolved, optUser); target != nil {
				key.UserID = target.ID
			} else {
				key = domain.PlayerKey{}
			}
		}
		if key.UserID != "" {
			typed, _ := focused.Value.(string)
			choices = b.suggest(ctx, key, typed)
		}
	}
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
	if err != nil {
		b.log.Warn("autocomplete respond failed", "err", err)
	}
}

func (b *Bot) suggest(ctx context.Context, key domain.PlayerKey, typed string) []*discordgo.ApplicationCommandOptionChoice {
	prefixParts := splitSlugs(typed)
	last := ""
	if idx := strings.LastIndex(typed, ","); idx >= 0 {
		last = strings.TrimSpace(typed[idx+1:])
	} else {
		last = strings.TrimSpace(typed)
		prefixParts = nil
	}
	if strings.HasSuffix(strings.TrimSpace(typed), ",") {
		last = ""
	} else if len(prefixParts) > 0 {
		prefixParts = prefixParts[:len(prefixParts)-1]
	}
	items, err := b.svc.Inventory.Suggest(ctx, key, last, maxChoices)
	if err != nil {
		b.log.Warn("autocomplete lookup failed", "err", err)
		return nil
	}
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(items))
	for _, it := range items {
		value := strings.Join(append(append([]string{}, prefixParts...), it.Slug), ",")
		if len(value) > maxChoiceLength {
			continue
		}
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: truncate(it.Name, maxChoiceLength), Value: value})
	}
	return choices
}

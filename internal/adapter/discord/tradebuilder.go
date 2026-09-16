package discord

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	builderPrefix      = "tb:"
	builderGive        = "give"
	builderGet         = "get"
	builderGivePrev    = "gprev"
	builderGiveNext    = "gnext"
	builderGetPrev     = "tprev"
	builderGetNext     = "tnext"
	builderSend        = "send"
	builderFilter      = "filter"
	builderCancel      = "cancel"
	builderFilterModal = "tb:filtermodal"
	builderFilterInput = "name"
	builderPageSize    = 25

	msgBuilderTitleFmt   = "Trade with %s"
	msgBuilderNothing    = "Pick at least one waifu on either side, then press Send."
	msgBuilderTooMany    = "At most 25 waifus per side."
	msgBuilderSent       = "Offer sent."
	msgBuilderCancelled  = "Trade cancelled."
	msgBuilderEmptySide  = "*nothing yet*"
	msgTradeEmptySide    = "*nothing*"
	msgBuilderNoMatches  = "No waifus match the current filter."
	msgOfferCounteredFmt = "%s countered this offer; see the new request below."
)

type builderSide struct {
	all      []domain.OwnedWaifu
	selected map[string]bool
	page     int
}

func newBuilderSide(items []domain.OwnedWaifu) *builderSide {
	return &builderSide{all: items, selected: map[string]bool{}}
}

func (s *builderSide) visible(filter string) []domain.OwnedWaifu {
	if filter == "" {
		return s.all
	}
	filter = strings.ToLower(filter)
	out := make([]domain.OwnedWaifu, 0, len(s.all))
	for _, w := range s.all {
		if strings.Contains(strings.ToLower(w.Name), filter) || strings.Contains(strings.ToLower(w.Slug), filter) {
			out = append(out, w)
		}
	}
	return out
}

func (s *builderSide) pageOf(filter string) ([]domain.OwnedWaifu, int, int) {
	items := s.visible(filter)
	pages := max((len(items)+builderPageSize-1)/builderPageSize, 1)
	if s.page >= pages {
		s.page = pages - 1
	}
	if s.page < 0 {
		s.page = 0
	}
	start := s.page * builderPageSize
	end := min(start+builderPageSize, len(items))
	if start > len(items) {
		return nil, s.page, pages
	}
	return items[start:end], s.page, pages
}

func (s *builderSide) apply(pageItems []domain.OwnedWaifu, values []string) {
	chosen := map[string]bool{}
	for _, v := range values {
		chosen[v] = true
	}
	for _, w := range pageItems {
		if chosen[w.Slug] {
			s.selected[w.Slug] = true
		} else {
			delete(s.selected, w.Slug)
		}
	}
}

func (s *builderSide) picked() []domain.OwnedWaifu {
	out := make([]domain.OwnedWaifu, 0, len(s.selected))
	for _, w := range s.all {
		if s.selected[w.Slug] {
			out = append(out, w)
		}
	}
	return out
}

func (s *builderSide) slugs() []string {
	picked := s.picked()
	out := make([]string, len(picked))
	for i, w := range picked {
		out[i] = w.Slug
	}
	sort.Strings(out)
	return out
}

type tradeBuilder struct {
	give      *builderSide
	get       *builderSide
	filter    string
	counterOf string
	notice    string
}

func (b *Bot) openTradeBuilder(ctx context.Context, ic *interaction, target *discordgo.User, prefill domain.TradeOffer, counterOf string) {
	if !b.deferReply(ic, true) {
		return
	}
	sender := ic.key()
	targetKey := domain.PlayerKey{GuildID: ic.GuildID, UserID: target.ID}
	mine, err := b.svc.Inventory.List(ctx, sender)
	if err != nil {
		b.failed(ic, "trade builder", err)
		return
	}
	theirs, err := b.svc.Inventory.List(ctx, targetKey)
	if err != nil {
		b.failed(ic, "trade builder", err)
		return
	}
	tb := &tradeBuilder{give: newBuilderSide(mine), get: newBuilderSide(theirs), counterOf: counterOf}
	for _, slug := range prefill.Give {
		tb.give.selected[slug] = true
	}
	for _, slug := range prefill.Receive {
		tb.get.selected[slug] = true
	}
	s := &Session{Kind: SessionTradeBuilder, OwnerID: ic.userID(), ChannelID: ic.ChannelID, SenderID: ic.userID(), TargetID: target.ID, Builder: tb}
	content, embeds, components := b.renderBuilder(s)
	msg := b.edit(ic, content, embeds, components)
	if msg == nil {
		return
	}
	b.sessions.Put(msg.ID, s, b.cfg.TradeTTL)
}

func (b *Bot) renderBuilder(s *Session) (string, []*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	tb := s.Builder
	givePage, gIdx, gPages := tb.give.pageOf(tb.filter)
	getPage, tIdx, tPages := tb.get.pageOf(tb.filter)

	embed := &discordgo.MessageEmbed{
		Title: "Trade builder",
		Color: brandingColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "You give", Value: b.builderList(tb.give.picked(), msgBuilderEmptySide), Inline: true},
			{Name: "You get", Value: b.builderList(tb.get.picked(), msgBuilderEmptySide), Inline: true},
		},
	}
	desc := []string{}
	if tb.filter != "" {
		desc = append(desc, fmt.Sprintf("Filter: `%s`", tb.filter))
	}
	if tb.notice != "" {
		desc = append(desc, tb.notice)
	}
	embed.Description = strings.Join(desc, "\n")
	embed.Footer = &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Yours: page %d/%d · Theirs: page %d/%d", gIdx+1, gPages, tIdx+1, tPages)}

	rows := []discordgo.MessageComponent{}
	if menu, ok := b.builderMenu(builderGive, "Your waifus to give", givePage, tb.give); ok {
		rows = append(rows, menu)
	}
	if menu, ok := b.builderMenu(builderGet, "Their waifus you want", getPage, tb.get); ok {
		rows = append(rows, menu)
	}
	nav := func(id, label string, disabled bool) discordgo.Button {
		return discordgo.Button{Style: discordgo.SecondaryButton, CustomID: builderPrefix + id, Label: label, Disabled: disabled}
	}
	rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		nav(builderGivePrev, "◀ Yours", gIdx == 0),
		nav(builderGiveNext, "Yours ▶", gIdx >= gPages-1),
		nav(builderGetPrev, "◀ Theirs", tIdx == 0),
		nav(builderGetNext, "Theirs ▶", tIdx >= tPages-1),
	}})
	rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.SuccessButton, CustomID: builderPrefix + builderSend, Label: "Send offer", Emoji: &discordgo.ComponentEmoji{Name: "📨"}, Disabled: len(tb.give.selected)+len(tb.get.selected) == 0},
		discordgo.Button{Style: discordgo.SecondaryButton, CustomID: builderPrefix + builderFilter, Label: "Filter by name", Emoji: &discordgo.ComponentEmoji{Name: "🔍"}},
		discordgo.Button{Style: discordgo.DangerButton, CustomID: builderPrefix + builderCancel, Label: "Cancel"},
	}})
	return fmt.Sprintf(msgBuilderTitleFmt, mention(s.TargetID)), []*discordgo.MessageEmbed{embed}, rows
}

func (b *Bot) builderMenu(id, placeholder string, page []domain.OwnedWaifu, side *builderSide) (discordgo.MessageComponent, bool) {
	if len(page) == 0 {
		return nil, false
	}
	ranking := b.svc.Ranking.Current()
	options := make([]discordgo.SelectMenuOption, 0, len(page))
	for _, w := range page {
		desc := fmt.Sprintf("❤️ %s · 🗑️ %s", thousands(w.Likes), thousands(w.Trash))
		if r, ok := ranking.Lookup(w.Slug); ok && r.Stars > 0 {
			desc = strings.Repeat("⭐", r.Stars) + " #" + thousands(r.Position) + " · " + desc
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncate(w.Name, maxChoiceLength),
			Description: truncate(desc, maxChoiceLength),
			Value:       w.Slug,
			Default:     side.selected[w.Slug],
		})
	}
	zero := 0
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{
		MenuType:    discordgo.StringSelectMenu,
		CustomID:    builderPrefix + id,
		Placeholder: placeholder,
		MinValues:   &zero,
		MaxValues:   len(options),
		Options:     options,
	}}}, true
}

func (b *Bot) builderList(items []domain.OwnedWaifu, empty string) string {
	if len(items) == 0 {
		return empty
	}
	ranking := b.svc.Ranking.Current()
	lines := make([]string, 0, len(items))
	for _, w := range items {
		line := "• " + w.Name
		if r, ok := ranking.Lookup(w.Slug); ok && r.Stars > 0 {
			line += " " + strings.Repeat("⭐", r.Stars)
		}
		lines = append(lines, line)
	}
	return truncate(strings.Join(lines, "\n"), 1000)
}

func (b *Bot) builderSession(ic *interaction) (*Session, bool) {
	if ic.Message == nil {
		return nil, false
	}
	s, ok := b.sessions.Get(ic.Message.ID)
	if !ok || s.Kind != SessionTradeBuilder {
		b.replyEphemeral(ic, msgExpired)
		return nil, false
	}
	if s.OwnerID != ic.userID() {
		b.replyEphemeral(ic, msgNotYourMenu)
		return nil, false
	}
	return s, true
}

func (b *Bot) builderComponent(ctx context.Context, ic *interaction, action string) {
	s, ok := b.builderSession(ic)
	if !ok {
		return
	}
	tb := s.Builder
	tb.notice = ""
	switch action {
	case builderGive:
		page, _, _ := tb.give.pageOf(tb.filter)
		tb.give.apply(page, ic.MessageComponentData().Values)
	case builderGet:
		page, _, _ := tb.get.pageOf(tb.filter)
		tb.get.apply(page, ic.MessageComponentData().Values)
	case builderGivePrev:
		tb.give.page--
	case builderGiveNext:
		tb.give.page++
	case builderGetPrev:
		tb.get.page--
	case builderGetNext:
		tb.get.page++
	case builderFilter:
		b.openFilterModal(ic, tb.filter)
		return
	case builderCancel:
		b.sessions.Delete(ic.Message.ID)
		b.updateMessage(ic, msgBuilderCancelled, nil, nil)
		return
	case builderSend:
		b.sendBuilderOffer(ctx, ic, s)
		return
	default:
		b.log.Warn("unknown trade builder action", "action", action)
		return
	}
	content, embeds, components := b.renderBuilder(s)
	b.updateMessage(ic, content, embeds, components)
}

func (b *Bot) openFilterModal(ic *interaction, current string) {
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: builderFilterModal,
			Title:    "Filter waifus by name",
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{
				CustomID:    builderFilterInput,
				Label:       "Name contains (leave empty to clear)",
				Style:       discordgo.TextInputShort,
				Required:    false,
				MaxLength:   50,
				Value:       current,
				Placeholder: "rem",
			}}}},
		},
	})
	if err != nil {
		b.log.Error("open filter modal failed", "err", err)
	}
}

func (b *Bot) builderFilterSubmit(ic *interaction, data discordgo.ModalSubmitInteractionData) {
	s, ok := b.builderSession(ic)
	if !ok {
		return
	}
	tb := s.Builder
	tb.filter = strings.TrimSpace(modalValue(data, builderFilterInput))
	tb.give.page, tb.get.page = 0, 0
	tb.notice = ""
	if tb.filter != "" && len(tb.give.visible(tb.filter))+len(tb.get.visible(tb.filter)) == 0 {
		tb.notice = msgBuilderNoMatches
	}
	content, embeds, components := b.renderBuilder(s)
	b.updateMessage(ic, content, embeds, components)
}

func (b *Bot) sendBuilderOffer(ctx context.Context, ic *interaction, s *Session) {
	tb := s.Builder
	give, get := tb.give.slugs(), tb.get.slugs()
	if len(give)+len(get) == 0 {
		tb.notice = msgBuilderNothing
	} else if len(give) > maxTradeItems || len(get) > maxTradeItems {
		tb.notice = msgBuilderTooMany
	}
	if tb.notice != "" {
		content, embeds, components := b.renderBuilder(s)
		b.updateMessage(ic, content, embeds, components)
		return
	}
	offer := domain.TradeOffer{
		Sender:  domain.PlayerKey{GuildID: ic.GuildID, UserID: s.SenderID},
		Target:  domain.PlayerKey{GuildID: ic.GuildID, UserID: s.TargetID},
		Give:    give,
		Receive: get,
	}
	proposal, err := b.svc.Trade.Propose(ctx, offer)
	if err != nil {
		var violation *domain.TradeViolation
		if errors.As(err, &violation) {
			tb.notice = violationText(violation, mention(s.TargetID), "") + "."
		} else {
			tb.notice = errorText(err)
		}
		content, embeds, components := b.renderBuilder(s)
		b.updateMessage(ic, content, embeds, components)
		return
	}

	b.updateMessage(ic, msgBuilderSent, nil, nil)
	b.sessions.Delete(ic.Message.ID)

	content := fmt.Sprintf(msgTradeOfferFmt, mention(s.TargetID), mention(s.SenderID))
	msg, err := b.s.FollowupMessageCreate(ic.Interaction, true, &discordgo.WebhookParams{
		Content:    content,
		Embeds:     []*discordgo.MessageEmbed{b.tradeEmbed(proposal.Give, proposal.Receive)},
		Components: tradeComponents(),
	})
	if err != nil {
		b.log.Error("posting trade offer failed", "err", err)
		return
	}
	b.sessions.Put(msg.ID, &Session{
		Kind:      SessionTrade,
		OwnerID:   s.TargetID,
		ChannelID: msg.ChannelID,
		Content:   content,
		Offer:     proposal.Offer,
		Give:      proposal.Give,
		Receive:   proposal.Receive,
		SenderID:  s.SenderID,
		TargetID:  s.TargetID,
	}, b.cfg.TradeTTL)

	if tb.counterOf != "" {
		if original, ok := b.sessions.Get(tb.counterOf); ok && original.Kind == SessionTrade {
			b.sessions.Delete(tb.counterOf)
			b.editChannelMessage(original.ChannelID, tb.counterOf, fmt.Sprintf(msgOfferCounteredFmt, mention(s.SenderID)), nil, nil)
		}
	}
}

func (b *Bot) tradeEmbed(give, receive []domain.OwnedWaifu) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Trade Request",
		Color: brandingColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "**Has**", Value: b.builderList(give, msgTradeEmptySide), Inline: true},
			{Name: "**Wants**", Value: b.builderList(receive, msgTradeEmptySide), Inline: true},
		},
	}
}

func (b *Bot) startCounter(ctx context.Context, ic *interaction, s *Session) {
	reversed := domain.TradeOffer{Give: s.Offer.Receive, Receive: s.Offer.Give}
	b.openTradeBuilder(ctx, ic, &discordgo.User{ID: s.SenderID}, reversed, ic.Message.ID)
}

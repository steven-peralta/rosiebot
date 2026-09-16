package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	pagerPrefix    = "pg:"
	pagerFirst     = "first"
	pagerPrev      = "prev"
	pagerJump      = "jump"
	pagerNext      = "next"
	pagerLast      = "last"
	pagerJumpModal = "pg:jumpmodal"
	pagerJumpInput = "page"
	janitorPeriod  = time.Minute
)

func pagerComponents(page, total int, sellable bool) []discordgo.MessageComponent {
	var rows []discordgo.MessageComponent
	if total > 1 {
		btn := func(id, emoji string, disabled bool) discordgo.Button {
			return discordgo.Button{Style: discordgo.SecondaryButton, CustomID: pagerPrefix + id, Emoji: &discordgo.ComponentEmoji{Name: emoji}, Disabled: disabled}
		}
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			btn(pagerFirst, "⏮", page == 0),
			btn(pagerPrev, "◀️", page == 0),
			btn(pagerJump, "⤴️", false),
			btn(pagerNext, "▶️", page >= total-1),
			btn(pagerLast, "⏭", page >= total-1),
		}})
	}
	if sellable {
		rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{sellAskButton()}})
	}
	return rows
}

func pagesFromSummaries(items []domain.WaifuSummary) []pageRef {
	pages := make([]pageRef, len(items))
	for i, s := range items {
		pages[i] = pageRef{summary: s}
	}
	return pages
}

func pagesFromOwned(items []domain.OwnedWaifu) []pageRef {
	pages := make([]pageRef, len(items))
	for i, o := range items {
		pages[i] = pageRef{summary: domain.WaifuSummary{Slug: o.Slug, UUID: o.UUID, Name: o.Name, PictureURL: o.PictureURL, Likes: o.Likes, Trash: o.Trash}}
	}
	return pages
}

func (b *Bot) renderPage(ctx context.Context, s *Session, elapsed time.Duration) (string, []*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	if len(s.Pages) == 0 {
		return s.Content, nil, nil
	}
	if s.Page < 0 {
		s.Page = 0
	}
	if s.Page > len(s.Pages)-1 {
		s.Page = len(s.Pages) - 1
	}
	ref := &s.Pages[s.Page]
	if ref.detail == nil {
		w, err := b.svc.Search.Detail(ctx, ref.summary.Slug)
		if err != nil {
			b.log.Warn("page detail fetch failed, using summary", "slug", ref.summary.Slug, "err", err)
		} else {
			ref.detail = &w
		}
	}
	var embed *discordgo.MessageEmbed
	if ref.detail != nil {
		embed = b.waifuEmbed(*ref.detail, elapsed)
	} else {
		var ranked *domain.RankedWaifu
		if r, ok := b.svc.Ranking.Current().Lookup(ref.summary.Slug); ok {
			ranked = &r
		}
		embed = summaryEmbed(ref.summary, ranked)
		embed.Footer = b.footer(elapsed)
	}
	content := s.Content
	if len(s.Pages) > 1 {
		content = strings.TrimRight(content, "\n") + fmt.Sprintf("\nPage %d out of %d", s.Page+1, len(s.Pages))
	}
	return content, []*discordgo.MessageEmbed{embed}, pagerComponents(s.Page, len(s.Pages), s.Sellable)
}

func (b *Bot) openPager(ctx context.Context, ic *interaction, content string, pages []pageRef, sellable bool, elapsed time.Duration) {
	s := &Session{Kind: SessionPager, OwnerID: ic.userID(), ChannelID: ic.ChannelID, Content: content, Pages: pages, Sellable: sellable}
	c, embeds, components := b.renderPage(ctx, s, elapsed)
	msg := b.edit(ic, c, embeds, components)
	if msg == nil || len(pages) <= 1 {
		return
	}
	b.sessions.Put(msg.ID, s, b.cfg.PagerTTL)
}

func (b *Bot) pagerSession(ic *interaction) (*Session, bool) {
	if ic.Message == nil {
		return nil, false
	}
	s, ok := b.sessions.Get(ic.Message.ID)
	if !ok || s.Kind != SessionPager {
		b.replyEphemeral(ic, msgExpired)
		if ic.Message != nil {
			b.editChannelMessage(ic.ChannelID, ic.Message.ID, ic.Message.Content, ic.Message.Embeds, nil)
		}
		return nil, false
	}
	if s.OwnerID != ic.userID() {
		b.replyEphemeral(ic, msgNotYourMenu)
		return nil, false
	}
	return s, true
}

func (b *Bot) pagerButton(ctx context.Context, ic *interaction, action string) {
	s, ok := b.pagerSession(ic)
	if !ok {
		return
	}
	if action == pagerJump {
		b.openJumpModal(ic, len(s.Pages))
		return
	}
	switch action {
	case pagerFirst:
		s.Page = 0
	case pagerPrev:
		s.Page--
	case pagerNext:
		s.Page++
	case pagerLast:
		s.Page = len(s.Pages) - 1
	default:
		b.log.Warn("unknown pager action", "action", action)
		return
	}
	content, embeds, components := b.renderPage(ctx, s, 0)
	b.updateMessage(ic, content, embeds, components)
}

func (b *Bot) openJumpModal(ic *interaction, total int) {
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: pagerJumpModal,
			Title:    "Jump to page",
			Components: []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.TextInput{
				CustomID:    pagerJumpInput,
				Label:       fmt.Sprintf("Page number (1-%d)", total),
				Style:       discordgo.TextInputShort,
				Required:    true,
				MinLength:   1,
				MaxLength:   6,
				Placeholder: "1",
			}}}},
		},
	})
	if err != nil {
		b.log.Error("open modal failed", "err", err)
	}
}

func (b *Bot) pagerJumpSubmit(ctx context.Context, ic *interaction, data discordgo.ModalSubmitInteractionData) {
	s, ok := b.pagerSession(ic)
	if !ok {
		return
	}
	value := strings.TrimSpace(modalValue(data, pagerJumpInput))
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > len(s.Pages) {
		b.replyEphemeral(ic, fmt.Sprintf("Please enter a page number between 1 and %d.", len(s.Pages)))
		return
	}
	s.Page = n - 1
	content, embeds, components := b.renderPage(ctx, s, 0)
	b.updateMessage(ic, content, embeds, components)
}

func modalValue(data discordgo.ModalSubmitInteractionData, customID string) string {
	for _, row := range data.Components {
		ar, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, c := range ar.Components {
			if ti, ok := c.(*discordgo.TextInput); ok && ti.CustomID == customID {
				return ti.Value
			}
		}
	}
	return ""
}

func (b *Bot) RunJanitor(ctx context.Context) {
	ticker := time.NewTicker(janitorPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.SweepExpired()
		}
	}
}

func (b *Bot) SweepExpired() int {
	expired := b.sessions.Expired()
	for _, s := range expired {
		switch s.Kind {
		case SessionTrade:
			b.editChannelMessage(s.ChannelID, s.MessageID, msgTradeExpired, nil, nil)
		default:
			msg, err := b.s.ChannelMessage(s.ChannelID, s.MessageID)
			if err != nil {
				continue
			}
			b.editChannelMessage(s.ChannelID, s.MessageID, msg.Content, msg.Embeds, nil)
		}
	}
	return len(expired)
}

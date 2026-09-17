package discord

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	pagerPrefix    = "pg:"
	pagerFirst     = "first"
	pagerPrev      = "prev"
	pagerJump      = "jump"
	pagerNext      = "next"
	pagerLast      = "last"
	pagerSelect    = "select"
	pagerJumpModal = "pg:jumpmodal"
	pagerJumpInput = "page"
	janitorPeriod  = time.Minute
)

func pagerComponents(page, total int, sellable bool) []discordgo.MessageComponent {
	return append(pagerRows(page, total, nil), cardActions(sellable, false))
}

func pagerRows(page, total int, options []discordgo.SelectMenuOption) []discordgo.MessageComponent {
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
		if len(options) > 1 {
			rows = append(rows, discordgo.ActionsRow{Components: []discordgo.MessageComponent{discordgo.SelectMenu{
				MenuType:    discordgo.StringSelectMenu,
				CustomID:    pagerPrefix + pagerSelect,
				Placeholder: "Jump to a result",
				Options:     options,
			}}})
		}
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
	if ref.series != nil {
		embed := seriesEmbed(*ref.series)
		embed.Footer = b.footer(elapsed)
		content := s.Content
		if len(s.Pages) > 1 {
			content = strings.TrimRight(content, "\n") + fmt.Sprintf("\nPage %d out of %d", s.Page+1, len(s.Pages))
		}
		fav := b.favorited(ctx, domain.PlayerKey{GuildID: s.GuildID, UserID: s.OwnerID}, domain.FavoriteSeries, ref.series.Slug)
		rows := append(pagerRows(s.Page, len(s.Pages), b.selectOptions(s)), discordgo.ActionsRow{Components: []discordgo.MessageComponent{favButton(domain.FavoriteSeries, fav)}})
		return content, []*discordgo.MessageEmbed{embed}, append(rows, s.Extra...)
	}
	if ref.group != nil {
		embed := compactEmbed(ref.group)
		embed.Footer = b.footer(elapsed)
		content := s.Content
		if len(s.Pages) > 1 {
			content = strings.TrimRight(content, "\n") + fmt.Sprintf("\nPage %d out of %d", s.Page+1, len(s.Pages))
		}
		return content, []*discordgo.MessageEmbed{embed}, append(pagerRows(s.Page, len(s.Pages), b.selectOptions(s)), s.Extra...)
	}
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
	fav := b.favorited(ctx, domain.PlayerKey{GuildID: s.GuildID, UserID: s.OwnerID}, domain.FavoriteWaifu, ref.summary.Slug)
	rows := append(pagerRows(s.Page, len(s.Pages), b.selectOptions(s)), cardActions(s.Sellable, fav))
	return content, []*discordgo.MessageEmbed{embed}, append(rows, s.Extra...)
}

func (b *Bot) selectOptions(s *Session) []discordgo.SelectMenuOption {
	if len(s.Pages) <= 1 {
		return nil
	}
	ranking := b.svc.Ranking.Current()
	start, end := selectWindow(s.Page, len(s.Pages), maxSuggestions)
	options := make([]discordgo.SelectMenuOption, 0, end-start)
	for i := start; i < end; i++ {
		if sr := s.Pages[i].series; sr != nil {
			options = append(options, discordgo.SelectMenuOption{
				Label:   truncate(fmt.Sprintf("%d. %s", i+1, sr.Name), maxChoiceLength),
				Value:   strconv.Itoa(i),
				Default: i == s.Page,
			})
			continue
		}
		if g := s.Pages[i].group; g != nil {
			count := len(g.items)
			if g.lines != nil {
				count = len(g.lines)
			}
			options = append(options, discordgo.SelectMenuOption{
				Label:   fmt.Sprintf("Page %d · %d to %d", i+1, g.first+1, g.first+count),
				Value:   strconv.Itoa(i),
				Default: i == s.Page,
			})
			continue
		}
		summary := s.Pages[i].summary
		desc := fmt.Sprintf("❤️ %s · 🗑️ %s", thousands(summary.Likes), thousands(summary.Trash))
		if r, ok := ranking.Lookup(summary.Slug); ok && r.Stars > 0 {
			desc = strings.Repeat("⭐", r.Stars) + " #" + thousands(r.Position) + " · " + desc
		}
		options = append(options, discordgo.SelectMenuOption{
			Label:       truncate(fmt.Sprintf("%d. %s", i+1, summary.Name), maxChoiceLength),
			Description: truncate(desc, maxChoiceLength),
			Value:       strconv.Itoa(i),
			Default:     i == s.Page,
		})
	}
	return options
}

func (b *Bot) openPager(ctx context.Context, ic *interaction, content string, pages []pageRef, sellable bool, elapsed time.Duration) {
	b.openPagerWith(ctx, ic, content, pages, sellable, nil, elapsed)
}

func (b *Bot) openPagerWith(ctx context.Context, ic *interaction, content string, pages []pageRef, sellable bool, extra []discordgo.MessageComponent, elapsed time.Duration) {
	s := &Session{Kind: SessionPager, OwnerID: ic.userID(), GuildID: ic.GuildID, ChannelID: ic.ChannelID, Content: content, Pages: pages, Sellable: sellable, Extra: extra}
	c, embeds, components := b.renderPage(ctx, s, elapsed)
	msg := b.edit(ic, c, embeds, components)
	if msg == nil || len(pages) <= 1 {
		return
	}
	b.sessions.Put(msg.ID, s, b.cfg.PagerTTL)
}

func selectWindow(page, total, size int) (int, int) {
	if total <= size {
		return 0, total
	}
	start := page - size/2
	start = max(start, 0)
	start = min(start, total-size)
	return start, start + size
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
	case pagerSelect:
		values := ic.MessageComponentData().Values
		if len(values) != 1 {
			return
		}
		n, err := strconv.Atoi(values[0])
		if err != nil {
			return
		}
		s.Page = n
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

const compactPageSize = 20

func pagesCompact(items []domain.OwnedWaifu, lookup app.RankLookup) []pageRef {
	return pagesCompactTitled(items, lookup, "")
}

func pagesCompactTitled(items []domain.OwnedWaifu, lookup app.RankLookup, title string) []pageRef {
	summaries := make([]domain.WaifuSummary, len(items))
	for i, o := range items {
		summaries[i] = domain.WaifuSummary{Slug: o.Slug, UUID: o.UUID, Name: o.Name, PictureURL: o.PictureURL, Likes: o.Likes, Trash: o.Trash}
	}
	return pagesCompactFromSummaries(summaries, lookup, title)
}

func pagesFromLines(lines []string, title string) []pageRef {
	pages := make([]pageRef, 0, (len(lines)+compactPageSize-1)/compactPageSize)
	for start := 0; start < len(lines); start += compactPageSize {
		end := min(start+compactPageSize, len(lines))
		pages = append(pages, pageRef{group: &compactGroup{first: start, lines: lines[start:end], title: title}})
	}
	return pages
}

func pagesCompactFromSummaries(summaries []domain.WaifuSummary, lookup app.RankLookup, title string) []pageRef {
	chars := cardCharacters(summaries, lookup)
	pages := make([]pageRef, 0, (len(chars)+compactPageSize-1)/compactPageSize)
	for start := 0; start < len(chars); start += compactPageSize {
		end := min(start+compactPageSize, len(chars))
		pages = append(pages, pageRef{group: &compactGroup{first: start, items: chars[start:end], title: title}})
	}
	return pages
}

func compactEmbed(g *compactGroup) *discordgo.MessageEmbed {
	lines := g.lines
	if lines == nil {
		lines = make([]string, len(g.items))
		for i, c := range g.items {
			lines[i] = fmt.Sprintf("%d. %s", g.first+i+1, cardLine(c))
		}
	}
	title := g.title
	if title == "" {
		title = "Collection"
	}
	return &discordgo.MessageEmbed{Title: title, Color: brandingColor, Description: strings.Join(lines, "\n")}
}

func collectionSummary(items []domain.OwnedWaifu, price func(slug string) (int64, int)) string {
	ranked := 0
	var worth int64
	for _, it := range items {
		p, stars := price(it.Slug)
		worth += p
		if stars > 0 {
			ranked++
		}
	}
	return fmt.Sprintf("%s waifus · %s ranked · worth :coin: %s coins if sold", thousands(len(items)), thousands(ranked), coins(worth))
}

func pagesFromSeries(items []domain.Series) []pageRef {
	pages := make([]pageRef, len(items))
	for i := range items {
		pages[i] = pageRef{series: &items[i]}
	}
	return pages
}

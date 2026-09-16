package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	rollPrefix = "roll:"
	rollAgain  = "again"
)

func rollAgainComponents(userID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Style: discordgo.PrimaryButton, CustomID: rollPrefix + rollAgain + ":" + userID, Emoji: &discordgo.ComponentEmoji{Name: "🎲"}, Label: fmt.Sprintf("Roll again · %d coins", domain.RollCost)},
	}}}
}

func (b *Bot) roll(ctx context.Context, ic *interaction) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	res, err := b.svc.Roll.Roll(ctx, ic.key())
	if err != nil {
		b.failed(ic, "roll", err)
		return
	}
	content, embed := b.rollResult(ic.userID(), res, b.cfg.Clock.Now().Sub(start))
	b.edit(ic, content, []*discordgo.MessageEmbed{embed}, rollAgainComponents(ic.userID()))
}

func (b *Bot) rollResult(userID string, res app.RollResult, elapsed time.Duration) (string, *discordgo.MessageEmbed) {
	content := mention(userID)
	switch res.Kind {
	case domain.RollCritical:
		content += " " + msgCritical
	case domain.RollWaifuOfTheDay:
		content += " " + msgWotdRoll
	default:
	}
	content += " " + msgRolled + "\n"
	return content, b.waifuEmbed(res.Waifu, elapsed)
}

func (b *Bot) rollAgain(ctx context.Context, ic *interaction, action string) {
	kind, owner, _ := strings.Cut(action, ":")
	if kind != rollAgain {
		b.log.Warn("unknown roll action", "action", action)
		return
	}
	if ic.GuildID == "" {
		b.replyEphemeral(ic, fmt.Sprintf(msgDMFmt, subRoll))
		return
	}
	if owner != ic.userID() {
		b.replyEphemeral(ic, msgNotYourRoll)
		return
	}
	if err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate}); err != nil {
		b.log.Error("defer roll again failed", "err", err)
		return
	}
	start := b.cfg.Clock.Now()
	res, err := b.svc.Roll.Roll(ctx, ic.key())
	if err != nil {
		b.log.Warn("roll again failed", "user", ic.userID(), "err", err)
		if _, ferr := b.s.FollowupMessageCreate(ic.Interaction, true, &discordgo.WebhookParams{Content: mention(ic.userID()) + " " + errorText(err), Flags: discordgo.MessageFlagsEphemeral}); ferr != nil {
			b.log.Error("roll again followup failed", "err", ferr)
		}
		return
	}
	content, embed := b.rollResult(ic.userID(), res, b.cfg.Clock.Now().Sub(start))
	b.edit(ic, content, []*discordgo.MessageEmbed{embed}, rollAgainComponents(ic.userID()))
}

func (b *Bot) daily(ctx context.Context, ic *interaction) {
	if !b.deferReply(ic, false) {
		return
	}
	res, err := b.svc.Daily.Claim(ctx, ic.key())
	if err != nil {
		b.failed(ic, "daily", err)
		return
	}
	who := mention(ic.userID())
	if res.Critical() {
		b.editText(ic, fmt.Sprintf("%s %s", who, fmt.Sprintf(msgDailyCriticalFmt, msgCritical, res.Coins)))
		return
	}
	b.editText(ic, fmt.Sprintf("%s %s", who, fmt.Sprintf(msgDailyFmt, res.Coins)))
}

func coinWord(n int64) string {
	if n == 1 {
		return "1 coin"
	}
	return fmt.Sprintf("%d coins", n)
}

func (b *Bot) coins(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	if !b.deferReply(ic, false) {
		return
	}
	target := resolvedUser(opts, resolved, optUser)
	key := ic.key()
	if target != nil {
		key.UserID = target.ID
	}
	balance, err := b.svc.Coins.Balance(ctx, key)
	if err != nil {
		b.failed(ic, "coins", err)
		return
	}
	if target == nil || target.ID == ic.userID() {
		b.editText(ic, fmt.Sprintf("%s You have :coin: %s", mention(ic.userID()), coinWord(balance)))
		return
	}
	b.editText(ic, fmt.Sprintf("%s: %s has :coin: %s", mention(ic.userID()), mention(target.ID), coinWord(balance)))
}

func (b *Bot) owned(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	target := resolvedUser(opts, resolved, optUser)
	key := ic.key()
	if target != nil {
		key.UserID = target.ID
	}
	items, err := b.svc.Inventory.List(ctx, key)
	if err != nil {
		b.failed(ic, "owned", err)
		return
	}
	if len(items) == 0 {
		b.editText(ic, mention(ic.userID())+" "+msgOwnsNothing)
		return
	}
	pages := pagesFromOwned(sortOwned(items, stringOption(opts, optSort), app.LookupFrom(b.svc.Ranking)))
	b.openPager(ctx, ic, mention(ic.userID()), pages, key.UserID == ic.userID(), b.cfg.Clock.Now().Sub(start))
}

func sortOwned(items []domain.OwnedWaifu, sortValue string, lookup app.RankLookup) []domain.OwnedWaifu {
	choice, ok := findSort(ownedSorts, sortValue)
	if !ok || choice.value == "oldest" {
		return items
	}
	if choice.value == "newest" {
		out := make([]domain.OwnedWaifu, len(items))
		for i, it := range items {
			out[len(items)-1-i] = it
		}
		return out
	}
	bySlug := make(map[string]domain.OwnedWaifu, len(items))
	summaries := make([]domain.WaifuSummary, len(items))
	for i, it := range items {
		bySlug[it.Slug] = it
		summaries[i] = domain.WaifuSummary{Slug: it.Slug, UUID: it.UUID, Name: it.Name, PictureURL: it.PictureURL, Likes: it.Likes, Trash: it.Trash}
	}
	sorted := app.Query{SortBy: choice.field, Descending: choice.desc}.Apply(summaries, lookup)
	out := make([]domain.OwnedWaifu, len(sorted))
	for i, s := range sorted {
		out[i] = bySlug[s.Slug]
	}
	return out
}

func (b *Bot) search(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	query := queryFromOptions(opts)
	if seriesInput := strings.TrimSpace(stringOption(opts, optSeries)); seriesInput != "" {
		b.searchWithinSeries(ctx, ic, seriesInput, query, start)
		return
	}
	if slug, ok := directSlug(query.Term); ok {
		w, err := b.svc.Search.Detail(ctx, slug)
		if errors.Is(err, app.ErrNotFound) {
			b.editText(ic, mention(ic.userID())+" "+msgWaifuNotFound)
			return
		}
		if err != nil {
			b.failed(ic, "search", err)
			return
		}
		if hasSearchOptions(opts) && !b.svc.Search.Matches(w.WaifuSummary, query) {
			b.editText(ic, fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgPickFilteredFmt, w.Name, b.ratingSummary(w))))
			return
		}
		b.edit(ic, mention(ic.userID()), []*discordgo.MessageEmbed{b.waifuEmbed(w, b.cfg.Clock.Now().Sub(start))}, nil)
		return
	}
	b.runSearch(ctx, ic, query, start)
}

func (b *Bot) list(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	query := queryFromOptions(opts)
	query.Term = ""
	if seriesInput := strings.TrimSpace(stringOption(opts, optSeries)); seriesInput != "" {
		b.searchWithinSeries(ctx, ic, seriesInput, query, start)
		return
	}
	b.runSearch(ctx, ic, query, start)
}

func (b *Bot) runSearch(ctx context.Context, ic *interaction, query app.Query, start time.Time) {
	results, err := b.svc.Search.Waifus(ctx, query)
	var filtered *app.FilteredOutError
	if errors.As(err, &filtered) {
		b.editText(ic, fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgFilteredOutFmt, filtered.Found)))
		return
	}
	if errors.Is(err, app.ErrNotFound) {
		b.editText(ic, mention(ic.userID())+" "+msgWaifuNotFound)
		return
	}
	if err != nil {
		b.failed(ic, "search", err)
		return
	}
	b.openPager(ctx, ic, mention(ic.userID()), pagesFromSummaries(results), false, b.cfg.Clock.Now().Sub(start))
}

func (b *Bot) searchWithinSeries(ctx context.Context, ic *interaction, seriesInput string, query app.Query, start time.Time) {
	if slug, ok := directSlug(query.Term); ok {
		query.Term = slug
	}
	var (
		res app.SeriesResult
		err error
	)
	if slug, ok := directSlug(seriesInput); ok {
		res, err = b.svc.Search.SeriesBySlug(ctx, slug, query)
	} else {
		res, err = b.svc.Search.Series(ctx, seriesInput, query)
	}
	var filtered *app.FilteredOutError
	switch {
	case errors.As(err, &filtered):
		b.editText(ic, fmt.Sprintf("%s %s %s", mention(ic.userID()), fmt.Sprintf(msgSeriesHeaderFmt, res.Series.Name)+":", fmt.Sprintf(msgFilteredOutFmt, filtered.Found)))
		return
	case errors.Is(err, app.ErrNotFound):
		b.editText(ic, mention(ic.userID())+" "+msgSeriesNotFound)
		return
	case err != nil:
		b.failed(ic, "series search", err)
		return
	}
	if len(res.Waifus) == 0 {
		b.edit(ic, mention(ic.userID())+" "+msgNoData, []*discordgo.MessageEmbed{seriesEmbed(res.Series)}, nil)
		return
	}
	content := fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgSeriesHeaderFmt, res.Series.Name))
	b.openPager(ctx, ic, content, pagesFromSummaries(res.Waifus), false, b.cfg.Clock.Now().Sub(start))
}

func (b *Bot) ratingSummary(w domain.Waifu) string {
	parts := []string{}
	if r, ok := b.svc.Search.Rank(w.Slug); ok && r.Stars > 0 {
		parts = append(parts, strings.Repeat("⭐", r.Stars), "rank #"+thousands(r.Position))
	} else {
		parts = append(parts, "unranked")
	}
	parts = append(parts, fmt.Sprintf("%s likes", thousands(w.Likes)), fmt.Sprintf("%s trash", thousands(w.Trash)))
	return strings.Join(parts, ", ")
}

func (b *Bot) searchAutocomplete(ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	typed := ""
	for _, o := range opts {
		if o.Focused && o.Name == optQuery {
			typed, _ = o.Value.(string)
		}
	}
	choices := suggestionChoices(b.svc.Search.Suggest(typed, maxSuggestions))
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
	if err != nil {
		b.log.Warn("search autocomplete respond failed", "err", err)
	}
}

func (b *Bot) random(ctx context.Context, ic *interaction) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	w, err := b.svc.Search.Random(ctx)
	if err != nil {
		b.failed(ic, "random", err)
		return
	}
	b.edit(ic, "", []*discordgo.MessageEmbed{b.waifuEmbed(w, b.cfg.Clock.Now().Sub(start))}, nil)
}

func (b *Bot) today(ctx context.Context, ic *interaction) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	res, err := b.svc.Wotd.Today(ctx)
	if err != nil {
		b.failed(ic, "today", err)
		return
	}
	content := fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgWotdFmt, domain.FormatCountdown(res.RefreshIn)))
	detail, err := b.svc.Search.Detail(ctx, res.Waifu.Slug)
	if err != nil {
		b.log.Warn("waifu of the day detail fetch failed, using summary", "slug", res.Waifu.Slug, "err", err)
		var ranked *domain.RankedWaifu
		if r, ok := b.svc.Ranking.Current().Lookup(res.Waifu.Slug); ok {
			ranked = &r
		}
		e := summaryEmbed(res.Waifu, ranked)
		e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
		b.edit(ic, content, []*discordgo.MessageEmbed{e}, nil)
		return
	}
	b.edit(ic, content, []*discordgo.MessageEmbed{b.waifuEmbed(detail, b.cfg.Clock.Now().Sub(start))}, nil)
}

func (b *Bot) seriesAutocomplete(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	typed := ""
	for _, o := range opts {
		if o.Focused && o.Name == optSeries {
			typed, _ = o.Value.(string)
		}
	}
	series, err := b.svc.Search.SuggestSeries(ctx, typed, maxSuggestions)
	if err != nil {
		b.log.Warn("series autocomplete lookup failed", "err", err)
	}
	if err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: seriesChoices(series)},
	}); err != nil {
		b.log.Warn("series autocomplete respond failed", "err", err)
	}
}

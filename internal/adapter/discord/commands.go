package discord

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

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
	content := mention(ic.userID())
	switch res.Kind {
	case domain.RollCritical:
		content += " " + msgCritical
	case domain.RollWaifuOfTheDay:
		content += " " + msgWotdRoll
	default:
	}
	content += " " + msgRolled + "\n"
	b.edit(ic, content, []*discordgo.MessageEmbed{b.waifuEmbed(res.Waifu, b.cfg.Clock.Now().Sub(start))}, nil)
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
	b.openPager(ctx, ic, mention(ic.userID()), pagesFromOwned(items), b.cfg.Clock.Now().Sub(start))
}

func (b *Bot) search(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	results, err := b.svc.Search.Waifus(ctx, stringOption(opts, optQuery))
	if errors.Is(err, app.ErrNotFound) {
		b.editText(ic, mention(ic.userID())+" "+msgWaifuNotFound)
		return
	}
	if errors.Is(err, app.ErrBadQuery) {
		b.editText(ic, mention(ic.userID())+" "+err.Error())
		return
	}
	if err != nil {
		b.failed(ic, "search", err)
		return
	}
	b.openPager(ctx, ic, mention(ic.userID()), pagesFromSummaries(results), b.cfg.Clock.Now().Sub(start))
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

func (b *Bot) seriesSearch(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	res, err := b.svc.Search.Series(ctx, stringOption(opts, optQuery))
	if errors.Is(err, app.ErrNotFound) {
		b.editText(ic, mention(ic.userID())+" "+msgSeriesNotFound)
		return
	}
	if err != nil {
		b.failed(ic, "series search", err)
		return
	}
	if len(res.Waifus) == 0 {
		b.edit(ic, mention(ic.userID())+" "+msgNoData, []*discordgo.MessageEmbed{seriesEmbed(res.Series)}, nil)
		return
	}
	content := fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgSeriesHeaderFmt, res.Series.Name))
	b.openPager(ctx, ic, content, pagesFromSummaries(res.Waifus), b.cfg.Clock.Now().Sub(start))
}

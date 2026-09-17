package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	commandProfile     = "profile"
	commandLeaderboard = "leaderboard"
	subHistory         = "history"
	optBy              = "by"
	leaderboardSize    = 10
	historySize        = 10
)

var metricChoices = []*discordgo.ApplicationCommandOptionChoice{
	{Name: "Collection value", Value: string(domain.MetricValue)},
	{Name: "Coins", Value: string(domain.MetricCoins)},
	{Name: "Collection size", Value: string(domain.MetricCollection)},
	{Name: "4-star and 5-star waifus", Value: string(domain.MetricStars)},
}

func metricLabel(m domain.Metric) string {
	for _, c := range metricChoices {
		if c.Value == string(m) {
			return c.Name
		}
	}
	return string(m)
}

func metricValue(m domain.Metric, v int64) string {
	switch m {
	case domain.MetricCoins, domain.MetricValue:
		return ":coin: " + coins(v)
	default:
		return coins(v)
	}
}

func (b *Bot) profile(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	target := ic.user()
	if u := resolvedUser(opts, resolved, optUser); u != nil {
		target = u
	}
	key := domain.PlayerKey{GuildID: ic.GuildID, UserID: target.ID}
	p, err := b.svc.Stats.Profile(ctx, key)
	if err != nil {
		b.failed(ic, "profile", err)
		return
	}
	e := profileEmbed(target, p)
	e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
	b.edit(ic, "", []*discordgo.MessageEmbed{e}, nil)
}

func profileEmbed(u *discordgo.User, p app.Profile) *discordgo.MessageEmbed {
	name := u.Username
	if u.GlobalName != "" {
		name = u.GlobalName
	}
	e := &discordgo.MessageEmbed{Title: name + "'s profile", Color: brandingColor}
	if u.Avatar != "" {
		e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: u.AvatarURL("256")}
	}
	tiers := make([]string, 0, 6)
	for stars := domain.MaxStars; stars >= 1; stars-- {
		if p.Tiers[stars] > 0 {
			tiers = append(tiers, fmt.Sprintf("%s %s", starBar(stars), thousands(p.Tiers[stars])))
		}
	}
	if p.Tiers[0] > 0 {
		tiers = append(tiers, fmt.Sprintf("%s %s unranked", starBar(0), thousands(p.Tiers[0])))
	}
	if len(tiers) == 0 {
		tiers = append(tiers, "No waifus yet.")
	}
	e.Fields = append(e.Fields,
		&discordgo.MessageEmbedField{Name: "Coins", Value: ":coin: " + coins(p.Coins), Inline: true},
		&discordgo.MessageEmbedField{Name: "Collection", Value: fmt.Sprintf("%s waifus · worth :coin: %s", thousands(p.Owned), coins(p.Value)), Inline: true},
		&discordgo.MessageEmbedField{Name: "Favorites", Value: thousands(p.Favorites), Inline: true},
		&discordgo.MessageEmbedField{Name: "By rating", Value: strings.Join(tiers, "\n")},
	)
	if p.Best != nil {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Rarest pull", Value: fmt.Sprintf("%s %s · Rank #%s", starBar(p.Best.Stars), p.Best.Name, thousands(p.Best.Position)), Inline: true})
	}
	rolls := fmt.Sprintf("%s total", thousands(p.Rolls))
	if p.LastRoll != nil {
		rolls += fmt.Sprintf("\nLast: %s (%s)", p.LastRoll.Name, p.LastRoll.Kind)
	}
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Rolls", Value: rolls, Inline: true})
	if p.Players > 0 {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Server rank", Value: fmt.Sprintf("#%d by collection value · #%d by collection size · of %s players", p.ValueRank, p.CollectionRank, thousands(p.Players)), Inline: false})
	}
	return e
}

func (b *Bot) leaderboard(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	metric := domain.Metric(stringOption(opts, optBy))
	if metric == "" {
		metric = domain.MetricValue
	}
	board, err := b.svc.Stats.Leaderboard(ctx, ic.GuildID, metric)
	if err != nil {
		b.failed(ic, "leaderboard", err)
		return
	}
	if len(board.Standings) == 0 {
		b.editText(ic, mention(ic.userID())+" "+msgLeaderboardEmpty)
		return
	}
	e := leaderboardEmbed(board, ic.userID())
	e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
	b.edit(ic, "", []*discordgo.MessageEmbed{e}, nil)
}

func leaderboardEmbed(board app.Board, viewerID string) *discordgo.MessageEmbed {
	lines := make([]string, 0, leaderboardSize+1)
	for i, s := range board.Standings {
		if i == leaderboardSize {
			break
		}
		lines = append(lines, fmt.Sprintf("%d. <@%s> · %s", i+1, s.UserID, metricValue(board.Metric, s.Value)))
	}
	if rank, ok := board.Rank(viewerID); ok && rank > leaderboardSize {
		lines = append(lines, "", fmt.Sprintf("You're #%d · %s", rank, metricValue(board.Metric, board.Standings[rank-1].Value)))
	}
	return &discordgo.MessageEmbed{Title: "Leaderboard · " + metricLabel(board.Metric), Color: brandingColor, Description: strings.Join(lines, "\n")}
}

func (b *Bot) history(ctx context.Context, ic *interaction, opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved) {
	if !b.deferReply(ic, false) {
		return
	}
	start := b.cfg.Clock.Now()
	key := ic.key()
	whose := "Your"
	if u := resolvedUser(opts, resolved, optUser); u != nil && u.ID != ic.userID() {
		key.UserID = u.ID
		whose = mention(u.ID) + "'s"
	}
	recent, total, err := b.svc.Inventory.History(ctx, key, historySize)
	if err != nil {
		b.failed(ic, "history", err)
		return
	}
	if total == 0 {
		b.editText(ic, fmt.Sprintf("%s %s", mention(ic.userID()), fmt.Sprintf(msgHistoryNoneFmt, whose)))
		return
	}
	now := b.cfg.Clock.Now()
	lookup := app.LookupFrom(b.svc.Ranking)
	lines := make([]string, len(recent))
	for i, r := range recent {
		stars := 0
		if rk, ok := lookup(r.Slug); ok {
			stars = rk.Stars
		}
		lines[i] = fmt.Sprintf("%s %s · %s · %s ago", starBar(stars), r.Name, rollKindLabel(r.Kind), humanDuration(now.Sub(r.At)))
	}
	e := &discordgo.MessageEmbed{Title: "Recent rolls", Color: brandingColor, Description: strings.Join(lines, "\n")}
	e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
	b.edit(ic, fmt.Sprintf("%s %s last %s of %s rolls", mention(ic.userID()), whose, thousands(len(recent)), thousands(total)), []*discordgo.MessageEmbed{e}, nil)
}

func rollKindLabel(k domain.RollKind) string {
	switch k {
	case domain.RollCritical:
		return "critical"
	case domain.RollWaifuOfTheDay:
		return "waifu of the day"
	case domain.RollBanner:
		return "banner"
	default:
		return "regular"
	}
}

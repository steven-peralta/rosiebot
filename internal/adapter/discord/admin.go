package discord

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	commandAdmin = "admin"

	groupCoins   = "coins"
	groupWaifu   = "waifu"
	groupBanner  = "banner"
	groupRanking = "ranking"

	adminSet       = "set"
	adminIncrement = "increment"
	adminDecrement = "decrement"
	adminAdd       = "add"
	adminRemove    = "remove"
	adminReroll    = "reroll"
	adminStatus    = "status"
	adminRefresh   = "refresh"

	optAmount = "amount"
	optWaifu  = "waifu"
)

func adminCommand() *discordgo.ApplicationCommand {
	guildOnly := []discordgo.InteractionContextType{discordgo.InteractionContextGuild}
	integrations := []discordgo.ApplicationIntegrationType{discordgo.ApplicationIntegrationGuildInstall}
	user := func(desc string) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionUser, Name: optUser, Description: desc, Required: true}
	}
	amount := func(desc string, minimum float64) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionInteger, Name: optAmount, Description: desc, Required: true, MinValue: &minimum}
	}
	waifu := func(desc string) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optWaifu, Description: desc, Required: true, Autocomplete: true}
	}
	sub := func(name, desc string, opts ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: desc, Options: opts}
	}
	return &discordgo.ApplicationCommand{
		Name:             commandAdmin,
		Description:      "Bot owner tools for coins, collections, and banners",
		Contexts:         &guildOnly,
		IntegrationTypes: &integrations,
		Options: []*discordgo.ApplicationCommandOption{
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: groupCoins, Description: "Change a player's balance", Options: []*discordgo.ApplicationCommandOption{
				sub(adminSet, "Set a player's balance", user("Whose balance to set"), amount("New balance", 0)),
				sub(adminIncrement, "Add coins to a player", user("Who receives the coins"), amount("Coins to add", 1)),
				sub(adminDecrement, "Take coins from a player", user("Who loses the coins"), amount("Coins to take", 1)),
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: groupWaifu, Description: "Change a player's collection", Options: []*discordgo.ApplicationCommandOption{
				sub(adminAdd, "Give a waifu to a player", user("Who receives the waifu"), waifu("Waifu to give")),
				sub(adminRemove, "Take a waifu from a player", user("Who loses the waifu"), waifu("Waifu to take")),
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: groupBanner, Description: "Manage this week's banner", Options: []*discordgo.ApplicationCommandOption{
				sub(adminReroll, "Pick a different series for this week's banner"),
			}},
			{Type: discordgo.ApplicationCommandOptionSubCommandGroup, Name: groupRanking, Description: "Inspect the star ranking", Options: []*discordgo.ApplicationCommandOption{
				sub(adminStatus, "Show the ranking snapshot and refresh progress"),
				sub(adminRefresh, "Walk MyWaifuList's rankings now instead of waiting for the daily refresh"),
			}},
			sub(subHelp, "Explain the admin tools"),
		},
	}
}

func groupSubcommand(data discordgo.ApplicationCommandInteractionData) (group, sub string, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(data.Options) == 0 {
		return "", "", nil
	}
	if data.Options[0].Type == discordgo.ApplicationCommandOptionSubCommand {
		return "", data.Options[0].Name, data.Options[0].Options
	}
	if data.Options[0].Type != discordgo.ApplicationCommandOptionSubCommandGroup || len(data.Options[0].Options) == 0 {
		return "", "", nil
	}
	g := data.Options[0]
	s := g.Options[0]
	return g.Name, s.Name, s.Options
}

func (b *Bot) isOwner(ic *interaction) bool {
	id := ic.userID()
	for _, owner := range b.cfg.OwnerIDs {
		if owner == id {
			return true
		}
	}
	return false
}

func (b *Bot) admin(ctx context.Context, ic *interaction, data discordgo.ApplicationCommandInteractionData) {
	group, sub, opts := groupSubcommand(data)
	b.log.Info("command", "user", ic.userID(), "guild", ic.GuildID, "name", commandAdmin, "sub", group+" "+sub, "options", describeOptions(opts))
	if ic.GuildID == "" {
		b.replyEphemeral(ic, fmt.Sprintf(msgDMFmt, commandAdmin))
		return
	}
	if len(b.cfg.OwnerIDs) == 0 {
		b.replyEphemeral(ic, msgNoOwner)
		return
	}
	if !b.isOwner(ic) {
		b.log.Warn("admin command refused", "user", ic.userID(), "guild", ic.GuildID)
		b.replyEphemeral(ic, msgAdminOnly)
		return
	}
	if group == "" && sub == subHelp {
		b.help(ic, adminHelpEmbed())
		return
	}
	if group == groupBanner {
		b.adminBanner(ctx, ic, sub)
		return
	}
	if group == groupRanking {
		b.adminRanking(ic, sub)
		return
	}
	target := resolvedUser(opts, data.Resolved, optUser)
	if target == nil {
		b.replyEphemeral(ic, msgUserNotFound)
		return
	}
	if !b.deferReply(ic, true) {
		return
	}
	actor := ic.key()
	key := domain.PlayerKey{GuildID: ic.GuildID, UserID: target.ID}
	who := mention(target.ID)
	amount, _ := intOption(opts, optAmount)
	var (
		text string
		err  error
	)
	switch group + "/" + sub {
	case groupCoins + "/" + adminSet:
		var balance int64
		balance, err = b.svc.Admin.SetCoins(ctx, actor, key, int64(amount))
		text = fmt.Sprintf(msgAdminSetFmt, who, coins(balance))
	case groupCoins + "/" + adminIncrement:
		var balance int64
		balance, err = b.svc.Admin.AdjustCoins(ctx, actor, key, int64(amount))
		text = fmt.Sprintf(msgAdminAddedFmt, thousands(amount), who, coins(balance))
	case groupCoins + "/" + adminDecrement:
		var balance int64
		balance, err = b.svc.Admin.AdjustCoins(ctx, actor, key, -int64(amount))
		text = fmt.Sprintf(msgAdminRemovedFmt, thousands(amount), who, coins(balance))
	case groupWaifu + "/" + adminAdd:
		var w domain.Waifu
		w, err = b.svc.Admin.GrantWaifu(ctx, actor, key, waifuInput(stringOption(opts, optWaifu)))
		text = fmt.Sprintf(msgAdminGrantedFmt, w.Name, who)
	case groupWaifu + "/" + adminRemove:
		var w domain.OwnedWaifu
		w, err = b.svc.Admin.RevokeWaifu(ctx, actor, key, waifuInput(stringOption(opts, optWaifu)))
		text = fmt.Sprintf(msgAdminRevokedFmt, w.Name, who)
	default:
		b.log.Warn("unknown admin subcommand", "group", group, "sub", sub)
		b.editText(ic, msgUnexpected)
		return
	}
	if err != nil {
		b.editText(ic, adminErrorText(err, who))
		return
	}
	b.editText(ic, text)
}

func (b *Bot) adminBanner(ctx context.Context, ic *interaction, sub string) {
	if sub != adminReroll {
		b.log.Warn("unknown admin banner subcommand", "sub", sub)
		b.replyEphemeral(ic, msgUnexpected)
		return
	}
	if !b.deferReply(ic, true) {
		return
	}
	start := b.cfg.Clock.Now()
	bn, err := b.svc.Banner.Reroll(ctx)
	if err != nil {
		switch {
		case errors.Is(err, app.ErrNoRanking):
			b.editText(ic, msgAdminNoRanking)
		case errors.Is(err, app.ErrNoEligibleSeries):
			b.editText(ic, msgAdminNoSeries)
		default:
			b.failed(ic, "banner reroll", err)
		}
		return
	}
	e := bannerEmbed(bn)
	e.Footer = b.footer(b.cfg.Clock.Now().Sub(start))
	b.edit(ic, fmt.Sprintf(msgAdminRerolledFmt, bn.Series.Name), []*discordgo.MessageEmbed{e}, nil)
}

func (b *Bot) adminRanking(ic *interaction, sub string) {
	switch sub {
	case adminStatus:
	case adminRefresh:
		b.adminRankingRefresh(ic)
		return
	default:
		b.log.Warn("unknown admin ranking subcommand", "sub", sub)
		b.replyEphemeral(ic, msgUnexpected)
		return
	}
	if b.svc.Status == nil {
		b.replyEphemeral(ic, msgRankingStatusUnavailable)
		return
	}
	e := rankingStatusEmbed(b.svc.Status.Status(), b.cfg.Clock.Now())
	err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Embeds: []*discordgo.MessageEmbed{e}, Flags: discordgo.MessageFlagsEphemeral},
	})
	if err != nil {
		b.log.Error("ranking status reply failed", "err", err)
	}
}

func (b *Bot) adminRankingRefresh(ic *interaction) {
	if b.svc.Refresher == nil || b.svc.Status == nil {
		b.replyEphemeral(ic, msgRankingStatusUnavailable)
		return
	}
	if b.svc.Status.Status().Refreshing {
		b.replyEphemeral(ic, msgRefreshRunning)
		return
	}
	b.log.Info("admin ranking refresh", "actor", ic.userID(), "guild", ic.GuildID)
	b.backgroundFor(rankingRefreshTimeout, func(ctx context.Context) {
		err := b.svc.Refresher.Refresh(ctx)
		switch {
		case errors.Is(err, app.ErrRefreshInProgress):
			b.log.Info("admin ranking refresh skipped, already running")
		case err != nil:
			b.log.Error("admin ranking refresh failed", "err", err)
		}
	})
	b.replyEphemeral(ic, msgRefreshStarted)
}

func rankingStatusEmbed(st app.RankingStatus, now time.Time) *discordgo.MessageEmbed {
	e := &discordgo.MessageEmbed{Title: "Ranking status", Color: brandingColor}
	snapshot := "No snapshot yet. Stars and critical rolls use the live popular pages until the first walk finishes."
	if st.Loaded {
		snapshot = fmt.Sprintf("%s ranked characters, fetched %s ago (cutoff page %d).", thousands(st.Rows), humanDuration(now.Sub(st.FetchedAt)), st.CutoffPage)
	}
	e.Fields = append(e.Fields, helpField("Snapshot", snapshot))

	refresh := "Idle."
	if st.Refreshing {
		elapsed := now.Sub(st.StartedAt)
		refresh = fmt.Sprintf("Running for %s: page %d", humanDuration(elapsed), st.Page)
		if st.LastPage > 0 {
			refresh += fmt.Sprintf(" of %d", st.LastPage)
		}
		refresh += fmt.Sprintf(", %s rows collected so far.", thousands(st.Collected))
		if st.Page > 0 && st.LastPage > st.Page && elapsed > 0 {
			remaining := time.Duration(float64(elapsed) / float64(st.Page) * float64(st.LastPage-st.Page))
			refresh += fmt.Sprintf(" Roughly %s to go if it runs to the last page; it usually stops earlier at the 100-vote cutoff.", humanDuration(remaining))
		}
	} else if !st.NextRefresh.IsZero() {
		if wait := st.NextRefresh.Sub(now); wait > 0 {
			refresh = fmt.Sprintf("Idle. Next refresh in %s.", humanDuration(wait))
		} else {
			refresh = "Due now; it starts as soon as the scheduler wakes."
		}
	}
	e.Fields = append(e.Fields, helpField("Refresh", refresh))
	if st.LastError != "" {
		e.Fields = append(e.Fields, helpField("Last error", fmt.Sprintf("%s (%s ago)", truncate(st.LastError, 900), humanDuration(now.Sub(st.LastErrorAt)))))
	}
	return e
}

func humanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd %dh", int(d.Hours())/24, int(d.Hours())%24)
	}
}

func waifuInput(raw string) string {
	if slug, ok := directSlug(raw); ok {
		return slug
	}
	return raw
}

func adminErrorText(err error, who string) string {
	switch {
	case errors.Is(err, app.ErrAlreadyOwned):
		return fmt.Sprintf(msgAdminAlreadyOwnsFmt, who)
	case errors.Is(err, app.ErrNotOwned):
		return fmt.Sprintf(msgAdminNotOwnedFmt, who)
	case errors.Is(err, app.ErrInsufficientCoins):
		return fmt.Sprintf(msgAdminBelowZeroFmt, who)
	case errors.Is(err, app.ErrNotFound):
		return msgWaifuNotFound
	default:
		return errorText(err)
	}
}

func (b *Bot) adminAutocomplete(ctx context.Context, ic *interaction, data discordgo.ApplicationCommandInteractionData) {
	group, sub, opts := groupSubcommand(data)
	choices := []*discordgo.ApplicationCommandOptionChoice{}
	if o := focusedOption(opts); o != nil && o.Name == optWaifu && group == groupWaifu {
		typed, _ := o.Value.(string)
		switch sub {
		case adminAdd:
			choices = suggestionChoices(b.svc.Search.Suggest(typed, maxSuggestions))
		case adminRemove:
			if target := resolvedUser(opts, data.Resolved, optUser); target != nil && ic.GuildID != "" {
				choices = b.suggest(ctx, domain.PlayerKey{GuildID: ic.GuildID, UserID: target.ID}, typed)
			}
		}
	}
	if err := b.s.InteractionRespond(ic.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	}); err != nil {
		b.log.Warn("admin autocomplete respond failed", "err", err)
	}
}

package discord

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	commandAdmin = "admin"

	groupCoins  = "coins"
	groupWaifu  = "waifu"
	groupBanner = "banner"

	adminSet       = "set"
	adminIncrement = "increment"
	adminDecrement = "decrement"
	adminAdd       = "add"
	adminRemove    = "remove"
	adminReroll    = "reroll"

	optAmount = "amount"
	optWaifu  = "waifu"
)

func adminCommand() *discordgo.ApplicationCommand {
	guildOnly := []discordgo.InteractionContextType{discordgo.InteractionContextGuild}
	integrations := []discordgo.ApplicationIntegrationType{discordgo.ApplicationIntegrationGuildInstall}
	adminPerms := int64(discordgo.PermissionAdministrator)
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
		Name:                     commandAdmin,
		Description:              "Administrator tools for coins and collections",
		Contexts:                 &guildOnly,
		IntegrationTypes:         &integrations,
		DefaultMemberPermissions: &adminPerms,
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
		},
	}
}

func groupSubcommand(data discordgo.ApplicationCommandInteractionData) (group, sub string, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(data.Options) == 0 || data.Options[0].Type != discordgo.ApplicationCommandOptionSubCommandGroup || len(data.Options[0].Options) == 0 {
		return "", "", nil
	}
	g := data.Options[0]
	s := g.Options[0]
	return g.Name, s.Name, s.Options
}

func (ic *interaction) isAdmin() bool {
	return ic.Member != nil && ic.Member.Permissions&discordgo.PermissionAdministrator != 0
}

func (b *Bot) admin(ctx context.Context, ic *interaction, data discordgo.ApplicationCommandInteractionData) {
	group, sub, opts := groupSubcommand(data)
	b.log.Info("command", "user", ic.userID(), "guild", ic.GuildID, "name", commandAdmin, "sub", group+" "+sub, "options", describeOptions(opts))
	if ic.GuildID == "" {
		b.replyEphemeral(ic, fmt.Sprintf(msgDMFmt, commandAdmin))
		return
	}
	if !ic.isAdmin() {
		b.replyEphemeral(ic, msgAdminOnly)
		return
	}
	if group == groupBanner {
		b.adminBanner(ctx, ic, sub)
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
		text = fmt.Sprintf(msgAdminSetFmt, who, balance)
	case groupCoins + "/" + adminIncrement:
		var balance int64
		balance, err = b.svc.Admin.AdjustCoins(ctx, actor, key, int64(amount))
		text = fmt.Sprintf(msgAdminAddedFmt, amount, who, balance)
	case groupCoins + "/" + adminDecrement:
		var balance int64
		balance, err = b.svc.Admin.AdjustCoins(ctx, actor, key, -int64(amount))
		text = fmt.Sprintf(msgAdminRemovedFmt, amount, who, balance)
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

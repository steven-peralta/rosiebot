package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type API interface {
	InteractionRespond(*discordgo.Interaction, *discordgo.InteractionResponse, ...discordgo.RequestOption) error
	InteractionResponseEdit(*discordgo.Interaction, *discordgo.WebhookEdit, ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreate(*discordgo.Interaction, bool, *discordgo.WebhookParams, ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessage(string, string, ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEditComplex(*discordgo.MessageEdit, ...discordgo.RequestOption) (*discordgo.Message, error)
	ApplicationCommandBulkOverwrite(string, string, []*discordgo.ApplicationCommand, ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error)
}

type Services struct {
	Roll      *app.RollService
	Daily     *app.DailyService
	Coins     *app.CoinsService
	Inventory *app.InventoryService
	Search    *app.SearchService
	Trade     *app.TradeService
	Wotd      *app.WotdService
	Banner    *app.BannerService
	Admin     *app.AdminService
	Ranking   app.RankingProvider
	Status    app.RankingStatusProvider
}

type Config struct {
	AppID          string
	BotUserID      string
	Version        string
	Clock          app.Clock
	Logger         *slog.Logger
	PagerTTL       time.Duration
	TradeTTL       time.Duration
	CommandTimeout time.Duration
}

const (
	defaultPagerTTL       = 15 * time.Minute
	defaultTradeTTL       = 10 * time.Minute
	defaultCommandTimeout = 60 * time.Second

	commandWaifu  = "waifu"
	commandWAlias = "w"
	commandSAlias = "s"
	commandWotd   = "wotd"
	commandSell   = "Sell Waifu"

	subRoll   = "roll"
	subDaily  = "daily"
	subCoins  = "coins"
	subOwned  = "owned"
	subSearch = "search"
	subList   = "list"
	subRandom = "random"
	subToday  = "today"
	subTrade  = "trade"
	subBanner = "banner"

	optUser    = "user"
	optQuery   = "query"
	optGive    = "give"
	optReceive = "receive"
	optBanner  = "banner"
)

type Bot struct {
	s        API
	svc      Services
	cfg      Config
	sessions *SessionStore
	log      *slog.Logger
}

func New(s API, svc Services, cfg Config) *Bot {
	if cfg.Clock == nil {
		cfg.Clock = app.SystemClock()
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.PagerTTL <= 0 {
		cfg.PagerTTL = defaultPagerTTL
	}
	if cfg.TradeTTL <= 0 {
		cfg.TradeTTL = defaultTradeTTL
	}
	if cfg.CommandTimeout <= 0 {
		cfg.CommandTimeout = defaultCommandTimeout
	}
	if cfg.Version == "" {
		cfg.Version = "dev"
	}
	return &Bot{s: s, svc: svc, cfg: cfg, sessions: NewSessionStore(cfg.Clock), log: cfg.Logger}
}

func (b *Bot) Sessions() *SessionStore { return b.sessions }

func (b *Bot) Commands() []*discordgo.ApplicationCommand {
	userOpt := func(desc string) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionUser, Name: optUser, Description: desc}
	}
	allContexts := []discordgo.InteractionContextType{discordgo.InteractionContextGuild, discordgo.InteractionContextBotDM, discordgo.InteractionContextPrivateChannel}
	guildOnly := []discordgo.InteractionContextType{discordgo.InteractionContextGuild}
	integrations := []discordgo.ApplicationIntegrationType{discordgo.ApplicationIntegrationGuildInstall}
	sub := func(name, desc string, opts ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: desc, Options: opts}
	}
	seriesOptions := func() []*discordgo.ApplicationCommandOption {
		return []*discordgo.ApplicationCommandOption{
			sub(subSearch, "Show a series card with its ranked characters",
				&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optQuery, Description: "Series name", Required: true, Autocomplete: true}),
			sub(subHelp, "Explain how the series commands work"),
		}
	}
	waifuOptions := func() []*discordgo.ApplicationCommandOption {
		return []*discordgo.ApplicationCommandOption{
			sub(subRoll, "Roll for a waifu",
				&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionBoolean, Name: optBanner, Description: fmt.Sprintf("Spend %d coins on a banner roll: 8%% featured, 12%% critical", domain.BannerRollCost)}),
			sub(subDaily, "Get your daily dose of waifu coins"),
			sub(subCoins, "See how many coins you or another user has", userOpt("Whose balance to show")),
			sub(subOwned, "See the waifus that you or another user owns", userOpt("Whose collection to show"),
				&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optSort, Description: "Order of the collection", Choices: choicesFor(ownedSorts)}),
			sub(subSearch, "Search for a waifu by name", searchOptions()...),
			sub(subList, "Browse waifus by rank, series, or filters", listOptions()...),
			sub(subRandom, "Pull a random waifu"),
			sub(subToday, "Show the waifu of the day"),
			sub(subBanner, "Show this week's banner series"),
			sub(subTrade, "Trade waifus with another user",
				&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionUser, Name: optUser, Description: "Who to trade with", Required: true},
				&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optGive, Description: "Waifus you give, comma separated", Autocomplete: true},
				&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optReceive, Description: "Waifus you receive, comma separated", Autocomplete: true},
			),
			sub(subHelp, "Explain rolls, odds, coins, trading, and stars"),
		}
	}
	return []*discordgo.ApplicationCommand{
		{
			Name:             commandWaifu,
			Description:      "Roll, collect, and trade waifus",
			Contexts:         &allContexts,
			IntegrationTypes: &integrations,
			Options:          waifuOptions(),
		},
		{
			Name:             commandWAlias,
			Description:      "Shorthand for /waifu",
			Contexts:         &allContexts,
			IntegrationTypes: &integrations,
			Options:          waifuOptions(),
		},
		{
			Name:             commandSeries,
			Description:      "Look up an anime, game, or other series",
			Contexts:         &allContexts,
			IntegrationTypes: &integrations,
			Options:          seriesOptions(),
		},
		{
			Name:             commandSAlias,
			Description:      "Shorthand for /series",
			Contexts:         &allContexts,
			IntegrationTypes: &integrations,
			Options:          seriesOptions(),
		},
		{
			Name:             commandWotd,
			Description:      "Show the waifu of the day",
			Contexts:         &allContexts,
			IntegrationTypes: &integrations,
		},
		adminCommand(),
		{
			Type:             discordgo.MessageApplicationCommand,
			Name:             commandSell,
			Contexts:         &guildOnly,
			IntegrationTypes: &integrations,
		},
	}
}

func (b *Bot) Register(guildID string) error {
	if _, err := b.s.ApplicationCommandBulkOverwrite(b.cfg.AppID, guildID, b.Commands()); err != nil {
		return fmt.Errorf("register commands: %w", err)
	}
	return nil
}

func (b *Bot) Handle(i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), b.cfg.CommandTimeout)
	defer cancel()
	ic := &interaction{Interaction: i.Interaction}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleCommand(ctx, ic)
	case discordgo.InteractionApplicationCommandAutocomplete:
		b.handleAutocomplete(ctx, ic)
	case discordgo.InteractionMessageComponent:
		b.handleComponent(ctx, ic)
	case discordgo.InteractionModalSubmit:
		b.handleModal(ctx, ic)
	default:
		b.log.Debug("ignoring interaction", "type", i.Type)
	}
}

func (b *Bot) handleCommand(ctx context.Context, ic *interaction) {
	data := ic.ApplicationCommandData()
	if data.CommandType == discordgo.MessageApplicationCommand {
		if data.Name == commandSell {
			b.sellContext(ctx, ic, data)
		}
		return
	}
	if data.Name == commandAdmin {
		b.admin(ctx, ic, data)
		return
	}
	sub, opts := subcommand(data)
	b.log.Info("command", "user", ic.userID(), "guild", ic.GuildID, "name", data.Name, "sub", sub, "options", describeOptions(opts))
	switch data.Name {
	case commandWaifu, commandWAlias:
		switch sub {
		case subRoll:
			b.guildOnly(ctx, ic, subRoll, func(ctx context.Context, ic *interaction) { b.roll(ctx, ic, opts) })
		case subDaily:
			b.guildOnly(ctx, ic, subDaily, b.daily)
		case subCoins:
			b.guildOnly(ctx, ic, subCoins, func(ctx context.Context, ic *interaction) { b.coins(ctx, ic, opts, data.Resolved) })
		case subOwned:
			b.guildOnly(ctx, ic, subOwned, func(ctx context.Context, ic *interaction) { b.owned(ctx, ic, opts, data.Resolved) })
		case subSearch:
			b.search(ctx, ic, opts)
		case subList:
			b.list(ctx, ic, opts)
		case subRandom:
			b.random(ctx, ic)
		case subToday:
			b.today(ctx, ic)
		case subBanner:
			b.banner(ctx, ic)
		case subHelp:
			b.help(ic, waifuHelpEmbed())
		case subTrade:
			b.guildOnly(ctx, ic, subTrade, func(ctx context.Context, ic *interaction) { b.trade(ctx, ic, opts, data.Resolved) })
		default:
			b.log.Warn("unknown waifu subcommand", "sub", sub)
		}
	case commandSeries, commandSAlias:
		switch sub {
		case subSearch:
			b.seriesSearch(ctx, ic, opts)
		case subHelp:
			b.help(ic, seriesHelpEmbed())
		default:
			b.log.Warn("unknown series subcommand", "sub", sub)
		}
	case commandWotd:
		b.today(ctx, ic)
	default:
		b.log.Warn("unknown command", "name", data.Name)
	}
}

func (b *Bot) guildOnly(ctx context.Context, ic *interaction, name string, fn func(context.Context, *interaction)) {
	if ic.GuildID == "" {
		b.replyEphemeral(ic, fmt.Sprintf(msgDMFmt, name))
		return
	}
	fn(ctx, ic)
}

func (b *Bot) handleComponent(ctx context.Context, ic *interaction) {
	id := ic.MessageComponentData().CustomID
	switch {
	case strings.HasPrefix(id, pagerPrefix):
		b.pagerButton(ctx, ic, strings.TrimPrefix(id, pagerPrefix))
	case strings.HasPrefix(id, sellPrefix):
		b.sellButton(ctx, ic, strings.TrimPrefix(id, sellPrefix))
	case strings.HasPrefix(id, tradePrefix):
		b.tradeButton(ctx, ic, strings.TrimPrefix(id, tradePrefix))
	case strings.HasPrefix(id, rollPrefix):
		b.rollAgain(ctx, ic, strings.TrimPrefix(id, rollPrefix))
	case strings.HasPrefix(id, builderPrefix):
		b.builderComponent(ctx, ic, strings.TrimPrefix(id, builderPrefix))
	case strings.HasPrefix(id, bannerPrefix):
		b.bannerButton(ctx, ic, strings.TrimPrefix(id, bannerPrefix))
	case strings.HasPrefix(id, seriesPrefix):
		b.seriesButton(ctx, ic, strings.TrimPrefix(id, seriesPrefix))
	case id == viewWaifuMenu:
		b.viewWaifu(ctx, ic)
	default:
		b.log.Warn("unknown component", "custom_id", id)
	}
}

func (b *Bot) handleModal(ctx context.Context, ic *interaction) {
	data := ic.ModalSubmitData()
	switch data.CustomID {
	case pagerJumpModal:
		b.pagerJumpSubmit(ctx, ic, data)
	case builderFilterModal:
		b.builderFilterSubmit(ic, data)
	}
}

func (b *Bot) handleAutocomplete(ctx context.Context, ic *interaction) {
	data := ic.ApplicationCommandData()
	if data.Name == commandAdmin {
		b.adminAutocomplete(ctx, ic, data)
		return
	}
	sub, opts := subcommand(data)
	if data.Name == commandSeries || data.Name == commandSAlias {
		if sub == subSearch {
			b.seriesAutocomplete(ctx, ic, opts)
		}
		return
	}
	if data.Name != commandWaifu && data.Name != commandWAlias {
		return
	}
	switch sub {
	case subTrade:
		b.tradeAutocomplete(ctx, ic, opts, data.Resolved)
	case subSearch, subList:
		if o := focusedOption(opts); o != nil && o.Name == optSeries {
			b.seriesAutocomplete(ctx, ic, opts)
		} else {
			b.searchAutocomplete(ic, opts)
		}
	}
}

type interaction struct {
	*discordgo.Interaction
}

func (ic *interaction) user() *discordgo.User {
	if ic.Member != nil && ic.Member.User != nil {
		return ic.Member.User
	}
	return ic.User
}

func (ic *interaction) userID() string {
	if u := ic.user(); u != nil {
		return u.ID
	}
	return ""
}

func (ic *interaction) key() domain.PlayerKey {
	return domain.PlayerKey{GuildID: ic.GuildID, UserID: ic.userID()}
}

func subcommand(data discordgo.ApplicationCommandInteractionData) (string, []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(data.Options) == 0 {
		return "", nil
	}
	first := data.Options[0]
	if first.Type == discordgo.ApplicationCommandOptionSubCommand {
		return first.Name, first.Options
	}
	return "", data.Options
}

func focusedOption(opts []*discordgo.ApplicationCommandInteractionDataOption) *discordgo.ApplicationCommandInteractionDataOption {
	for _, o := range opts {
		if o.Focused {
			return o
		}
	}
	return nil
}

func describeOptions(opts []*discordgo.ApplicationCommandInteractionDataOption) string {
	parts := make([]string, 0, len(opts))
	for _, o := range opts {
		parts = append(parts, fmt.Sprintf("%s=%v", o.Name, o.Value))
	}
	return strings.Join(parts, " ")
}

func option(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *discordgo.ApplicationCommandInteractionDataOption {
	for _, o := range opts {
		if o.Name == name {
			return o
		}
	}
	return nil
}

func stringOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	if o := option(opts, name); o != nil {
		if s, ok := o.Value.(string); ok {
			return s
		}
	}
	return ""
}

func resolvedUser(opts []*discordgo.ApplicationCommandInteractionDataOption, resolved *discordgo.ApplicationCommandInteractionDataResolved, name string) *discordgo.User {
	o := option(opts, name)
	if o == nil {
		return nil
	}
	id, _ := o.Value.(string)
	if resolved != nil && resolved.Users != nil {
		if u, ok := resolved.Users[id]; ok {
			return u
		}
	}
	if id != "" {
		return &discordgo.User{ID: id}
	}
	return nil
}

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
	Ranking   app.RankingProvider
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
	commandSeries = "series"
	commandSell   = "Sell Waifu"

	subRoll   = "roll"
	subDaily  = "daily"
	subCoins  = "coins"
	subOwned  = "owned"
	subSearch = "search"
	subRandom = "random"
	subToday  = "today"
	subTrade  = "trade"

	optUser    = "user"
	optQuery   = "query"
	optGive    = "give"
	optReceive = "receive"
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
	queryOpt := func(desc string, required bool) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optQuery, Description: desc, Required: required}
	}
	allContexts := []discordgo.InteractionContextType{discordgo.InteractionContextGuild, discordgo.InteractionContextBotDM, discordgo.InteractionContextPrivateChannel}
	guildOnly := []discordgo.InteractionContextType{discordgo.InteractionContextGuild}
	integrations := []discordgo.ApplicationIntegrationType{discordgo.ApplicationIntegrationGuildInstall}
	sub := func(name, desc string, opts ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
		return &discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionSubCommand, Name: name, Description: desc, Options: opts}
	}
	return []*discordgo.ApplicationCommand{
		{
			Name:             commandWaifu,
			Description:      "Roll, collect, and trade waifus",
			Contexts:         &allContexts,
			IntegrationTypes: &integrations,
			Options: []*discordgo.ApplicationCommandOption{
				sub(subRoll, "Roll for a waifu"),
				sub(subDaily, "Get your daily dose of waifu coins"),
				sub(subCoins, "See how many coins you or another user has", userOpt("Whose balance to show")),
				sub(subOwned, "See the waifus that you or another user owns", userOpt("Whose collection to show")),
				sub(subSearch, "Search for a waifu", queryOpt("Name, plus optional sortby:-rank or likes:>100 tokens; empty lists the catalog", false)),
				sub(subRandom, "Pull a random waifu"),
				sub(subToday, "Show the waifu of the day"),
				sub(subTrade, "Trade waifus with another user",
					&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionUser, Name: optUser, Description: "Who to trade with", Required: true},
					&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optGive, Description: "Waifus you give, comma separated", Autocomplete: true},
					&discordgo.ApplicationCommandOption{Type: discordgo.ApplicationCommandOptionString, Name: optReceive, Description: "Waifus you receive, comma separated", Autocomplete: true},
				),
			},
		},
		{
			Name:             commandSeries,
			Description:      "Look up series",
			Contexts:         &allContexts,
			IntegrationTypes: &integrations,
			Options: []*discordgo.ApplicationCommandOption{
				sub(subSearch, "Search for a series and list its waifus", queryOpt("Series name to search for", true)),
			},
		},
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
	sub, opts := subcommand(data)
	switch data.Name {
	case commandWaifu:
		switch sub {
		case subRoll:
			b.guildOnly(ctx, ic, subRoll, b.roll)
		case subDaily:
			b.guildOnly(ctx, ic, subDaily, b.daily)
		case subCoins:
			b.guildOnly(ctx, ic, subCoins, func(ctx context.Context, ic *interaction) { b.coins(ctx, ic, opts, data.Resolved) })
		case subOwned:
			b.guildOnly(ctx, ic, subOwned, func(ctx context.Context, ic *interaction) { b.owned(ctx, ic, opts, data.Resolved) })
		case subSearch:
			b.search(ctx, ic, opts)
		case subRandom:
			b.random(ctx, ic)
		case subToday:
			b.today(ctx, ic)
		case subTrade:
			b.guildOnly(ctx, ic, subTrade, func(ctx context.Context, ic *interaction) { b.trade(ctx, ic, opts, data.Resolved) })
		default:
			b.log.Warn("unknown waifu subcommand", "sub", sub)
		}
	case commandSeries:
		if sub == subSearch {
			b.seriesSearch(ctx, ic, opts)
		}
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
	default:
		b.log.Warn("unknown component", "custom_id", id)
	}
}

func (b *Bot) handleModal(ctx context.Context, ic *interaction) {
	data := ic.ModalSubmitData()
	if data.CustomID == pagerJumpModal {
		b.pagerJumpSubmit(ctx, ic, data)
	}
}

func (b *Bot) handleAutocomplete(ctx context.Context, ic *interaction) {
	data := ic.ApplicationCommandData()
	sub, opts := subcommand(data)
	if data.Name == commandWaifu && sub == subTrade {
		b.tradeAutocomplete(ctx, ic, opts, data.Resolved)
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

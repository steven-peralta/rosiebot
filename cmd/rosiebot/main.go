package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/adapter/discord"
	"github.com/steven-peralta/rosiebot/internal/adapter/mwl"
	"github.com/steven-peralta/rosiebot/internal/adapter/postgres"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/config"
)

var version = "dev"

type systemRandom struct{}

func (systemRandom) IntN(n int) int { return rand.IntN(n) }

func main() {
	dryRun := flag.Bool("dry-run", false, "validate configuration, migrate, and wire everything, then exit without connecting to Discord")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, *dryRun); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dryRun bool) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	logger.Info("starting rosiebot", "version", version, "timezone", cfg.Timezone.String(), "dry_run", dryRun)

	if cfg.MigrateOnStart {
		if err := postgres.Migrate(cfg.DatabaseURL); err != nil {
			return err
		}
		logger.Info("database migrated")
	}
	pool, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	clock := app.SystemClock()
	rng := systemRandom{}
	source, err := mwl.New(mwl.Config{APIKey: cfg.WaifuAPIKey, Logger: logger})
	if err != nil {
		return err
	}

	players := postgres.NewStore(pool, clock)
	ranking := app.NewRankingService(postgres.NewRankingStore(pool), source, clock, app.RankingConfig{RefreshInterval: cfg.RankingRefresh, MinVotes: cfg.RankingMinVotes}, logger)
	wotd := app.NewWotdService(postgres.NewDailyStore(pool), ranking, source, clock, rng, cfg.Timezone)
	services := discord.Services{
		Roll:      app.NewRollService(players, source, ranking, wotd, clock, rng, cfg.RankingMinVotes, logger),
		Daily:     app.NewDailyService(players, clock, rng, cfg.Timezone),
		Coins:     app.NewCoinsService(players),
		Inventory: app.NewInventoryService(players),
		Search:    app.NewSearchService(source, ranking),
		Trade:     app.NewTradeService(players),
		Wotd:      wotd,
		Ranking:   ranking,
	}

	if dryRun {
		if err := ranking.Load(ctx); err != nil {
			return err
		}
		logger.Info("dry run complete", "ranking_rows", ranking.Current().Len())
		return nil
	}

	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return fmt.Errorf("discord session: %w", err)
	}
	session.Identify.Intents = discordgo.IntentsGuilds
	me, err := session.User("@me")
	if err != nil {
		return fmt.Errorf("discord identity: %w", err)
	}
	bot := discord.New(session, services, discord.Config{AppID: me.ID, BotUserID: me.ID, Version: version, Clock: clock, Logger: logger})
	session.AddHandler(func(_ *discordgo.Session, i *discordgo.InteractionCreate) { bot.Handle(i) })
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Info("discord ready", "user", r.User.Username, "guilds", len(r.Guilds))
	})

	if err := session.Open(); err != nil {
		return fmt.Errorf("discord open: %w", err)
	}
	defer func() { _ = session.Close() }()

	if err := bot.Register(cfg.DevGuildID); err != nil {
		return err
	}
	logger.Info("commands registered", "scope", scopeName(cfg.DevGuildID))

	go func() {
		if err := ranking.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("ranking service stopped", "err", err)
		}
	}()
	go bot.RunJanitor(ctx)

	<-ctx.Done()
	logger.Info("shutting down")
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	<-shutdown.Done()
	return nil
}

func scopeName(guildID string) string {
	if guildID == "" {
		return "global"
	}
	return "guild " + guildID
}

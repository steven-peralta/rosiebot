package config

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultTimezone       = "America/Chicago"
	DefaultRankingRefresh = 24 * time.Hour
	DefaultMinVotes       = 100
	DefaultWaifuCacheTTL  = 24 * time.Hour
	DefaultSearchCacheTTL = time.Hour
)

type Config struct {
	DiscordToken    string
	WaifuAPIKey     string
	DatabaseURL     string
	Timezone        *time.Location
	DevGuildID      string
	RankingRefresh  time.Duration
	RankingMinVotes int
	WaifuCacheTTL   time.Duration
	SearchCacheTTL  time.Duration
	MigrateOnStart  bool
	LogLevel        slog.Level
}

type Getenv func(string) string

func Load(getenv Getenv) (Config, error) {
	var errs []error
	cfg := Config{
		DiscordToken:    firstNonEmpty(getenv("DISCORD_TOKEN"), getenv("DISCORD_TOKEN_KEY")),
		WaifuAPIKey:     getenv("WAIFU_API_KEY"),
		DatabaseURL:     getenv("DATABASE_URL"),
		DevGuildID:      strings.TrimSpace(getenv("DEV_GUILD_ID")),
		RankingRefresh:  DefaultRankingRefresh,
		RankingMinVotes: DefaultMinVotes,
		WaifuCacheTTL:   DefaultWaifuCacheTTL,
		SearchCacheTTL:  DefaultSearchCacheTTL,
		MigrateOnStart:  true,
		LogLevel:        slog.LevelInfo,
	}
	for name, value := range map[string]string{"DISCORD_TOKEN": cfg.DiscordToken, "WAIFU_API_KEY": cfg.WaifuAPIKey, "DATABASE_URL": cfg.DatabaseURL} {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("%s is required", name))
		}
	}

	tz := firstNonEmpty(getenv("BOT_TIMEZONE"), DefaultTimezone)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		errs = append(errs, fmt.Errorf("BOT_TIMEZONE %q: %w", tz, err))
	} else {
		cfg.Timezone = loc
	}

	for name, target := range map[string]*time.Duration{"RANKING_REFRESH": &cfg.RankingRefresh, "WAIFU_CACHE_TTL": &cfg.WaifuCacheTTL, "SEARCH_CACHE_TTL": &cfg.SearchCacheTTL} {
		v := getenv(name)
		if v == "" {
			continue
		}
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			errs = append(errs, fmt.Errorf("%s %q must be a positive duration", name, v))
		} else {
			*target = d
		}
	}
	if v := getenv("RANKING_MIN_VOTES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			errs = append(errs, fmt.Errorf("RANKING_MIN_VOTES %q must be a non-negative integer", v))
		} else {
			cfg.RankingMinVotes = n
		}
	}
	if v := getenv("MIGRATE_ON_START"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs = append(errs, fmt.Errorf("MIGRATE_ON_START %q must be a boolean", v))
		} else {
			cfg.MigrateOnStart = b
		}
	}
	if v := getenv("LOG_LEVEL"); v != "" {
		var level slog.Level
		if err := level.UnmarshalText([]byte(strings.ToUpper(v))); err != nil {
			errs = append(errs, fmt.Errorf("LOG_LEVEL %q: %w", v, err))
		} else {
			cfg.LogLevel = level
		}
	}
	return cfg, errors.Join(errs...)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

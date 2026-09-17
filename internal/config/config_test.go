package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) Getenv {
	return func(k string) string { return m[k] }
}

var required = map[string]string{"DISCORD_TOKEN": "t", "WAIFU_API_KEY": "k", "DATABASE_URL": "postgres://x"}

func with(extra map[string]string) map[string]string {
	m := map[string]string{}
	for k, v := range required {
		m[k] = v
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load(env(required))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timezone.String() != DefaultTimezone || cfg.RankingRefresh != DefaultRankingRefresh || cfg.RankingMinVotes != DefaultMinVotes || !cfg.MigrateOnStart || cfg.LogLevel != slog.LevelInfo || cfg.DevGuildID != "" || cfg.WaifuCacheTTL != DefaultWaifuCacheTTL || cfg.SearchCacheTTL != DefaultSearchCacheTTL || len(cfg.OwnerIDs) != 0 {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestLoad_Overrides(t *testing.T) {
	cfg, err := Load(env(with(map[string]string{
		"BOT_TIMEZONE": "UTC", "RANKING_REFRESH": "6h", "RANKING_MIN_VOTES": "250", "MIGRATE_ON_START": "false", "LOG_LEVEL": "debug", "DEV_GUILD_ID": " 123 ", "WAIFU_CACHE_TTL": "12h", "SEARCH_CACHE_TTL": "30m", "BOT_OWNER_IDS": " 1, 2 ,,3 ",
	})))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Timezone != time.UTC || cfg.RankingRefresh != 6*time.Hour || cfg.RankingMinVotes != 250 || cfg.MigrateOnStart || cfg.LogLevel != slog.LevelDebug || cfg.DevGuildID != "123" || cfg.WaifuCacheTTL != 12*time.Hour || cfg.SearchCacheTTL != 30*time.Minute || strings.Join(cfg.OwnerIDs, "|") != "1|2|3" {
		t.Errorf("overrides = %+v", cfg)
	}
}

func TestLoad_V1TokenFallback(t *testing.T) {
	m := with(nil)
	delete(m, "DISCORD_TOKEN")
	m["DISCORD_TOKEN_KEY"] = "legacy"
	cfg, err := Load(env(m))
	if err != nil || cfg.DiscordToken != "legacy" {
		t.Errorf("fallback = %q %v", cfg.DiscordToken, err)
	}
}

func TestLoad_RequiredAndInvalid(t *testing.T) {
	_, err := Load(env(map[string]string{}))
	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{"DISCORD_TOKEN", "WAIFU_API_KEY", "DATABASE_URL"} {
		if !strings.Contains(err.Error(), name+" is required") {
			t.Errorf("missing %s not reported: %v", name, err)
		}
	}

	bad := map[string]string{
		"BOT_TIMEZONE":      "Mars/Olympus",
		"RANKING_REFRESH":   "soon",
		"RANKING_MIN_VOTES": "-1",
		"MIGRATE_ON_START":  "maybe",
		"LOG_LEVEL":         "loud",
		"WAIFU_CACHE_TTL":   "0",
		"SEARCH_CACHE_TTL":  "later",
	}
	for k, v := range bad {
		_, err := Load(env(with(map[string]string{k: v})))
		if err == nil || !strings.Contains(err.Error(), k) {
			t.Errorf("%s=%q should fail with a named error, got %v", k, v, err)
		}
	}
	if _, err := Load(env(with(map[string]string{"RANKING_REFRESH": "-1h"}))); err == nil {
		t.Error("negative refresh should fail")
	}
}

package app

import (
	"context"
	"fmt"
	"sort"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

type Standing struct {
	UserID string
	Value  int64
}

type Board struct {
	Metric    domain.Metric
	Standings []Standing
}

func (b Board) Rank(userID string) (int, bool) {
	for i, s := range b.Standings {
		if s.UserID == userID {
			return i + 1, true
		}
	}
	return 0, false
}

type Profile struct {
	Key            domain.PlayerKey
	Coins          int64
	Owned          int
	Tiers          [domain.MaxStars + 1]int
	Value          int64
	Best           *domain.RankedWaifu
	Rolls          int
	LastRoll       *domain.RollRecord
	Favorites      int
	ValueRank      int
	CollectionRank int
	Players        int
}

type StatsService struct {
	players PlayerStore
	ranking RankingProvider
	favs    FavoriteStore
}

func NewStatsService(players PlayerStore, ranking RankingProvider, favs FavoriteStore) *StatsService {
	return &StatsService{players: players, ranking: ranking, favs: favs}
}

func (s *StatsService) stars(slug string) int {
	if r, ok := s.ranking.Current().Lookup(slug); ok {
		return r.Stars
	}
	return 0
}

func (s *StatsService) Profile(ctx context.Context, key domain.PlayerKey) (Profile, error) {
	player, err := s.players.EnsurePlayer(ctx, key)
	if err != nil {
		return Profile{}, fmt.Errorf("ensure player: %w", err)
	}
	owned, err := s.players.ListOwned(ctx, key, "", 0)
	if err != nil {
		return Profile{}, fmt.Errorf("list owned: %w", err)
	}
	p := Profile{Key: key, Coins: player.Coins, Owned: len(owned)}
	ranking := s.ranking.Current()
	for _, w := range owned {
		stars := 0
		if r, ok := ranking.Lookup(w.Slug); ok {
			stars = r.Stars
			if p.Best == nil || r.Position < p.Best.Position {
				best := r
				p.Best = &best
			}
		}
		p.Tiers[stars]++
		p.Value += domain.SellPriceFor(stars)
	}
	recent, err := s.players.RecentRolls(ctx, key, 1)
	if err != nil {
		return Profile{}, fmt.Errorf("recent rolls: %w", err)
	}
	if len(recent) > 0 {
		p.LastRoll = &recent[0]
	}
	if p.Rolls, err = s.players.CountRolls(ctx, key); err != nil {
		return Profile{}, fmt.Errorf("count rolls: %w", err)
	}
	favs, err := s.favs.List(ctx, key, domain.FavoriteWaifu)
	if err != nil {
		return Profile{}, fmt.Errorf("list favorites: %w", err)
	}
	p.Favorites = len(favs)
	value, err := s.Leaderboard(ctx, key.GuildID, domain.MetricValue)
	if err != nil {
		return Profile{}, err
	}
	p.ValueRank, _ = value.Rank(key.UserID)
	p.Players = len(value.Standings)
	collection, err := s.Leaderboard(ctx, key.GuildID, domain.MetricCollection)
	if err != nil {
		return Profile{}, err
	}
	p.CollectionRank, _ = collection.Rank(key.UserID)
	return p, nil
}

func (s *StatsService) Leaderboard(ctx context.Context, guildID string, metric domain.Metric) (Board, error) {
	if !metric.Valid() {
		return Board{}, fmt.Errorf("leaderboard metric %q: %w", metric, ErrNotFound)
	}
	players, err := s.players.GuildPlayers(ctx, guildID)
	if err != nil {
		return Board{}, fmt.Errorf("guild players: %w", err)
	}
	values := make(map[string]int64, len(players))
	for _, p := range players {
		values[p.Key.UserID] = 0
		if metric == domain.MetricCoins {
			values[p.Key.UserID] = p.Coins
		}
	}
	if metric != domain.MetricCoins {
		rows, err := s.players.GuildInventory(ctx, guildID)
		if err != nil {
			return Board{}, fmt.Errorf("guild inventory: %w", err)
		}
		for _, row := range rows {
			switch metric {
			case domain.MetricCollection:
				values[row.UserID]++
			case domain.MetricValue:
				values[row.UserID] += domain.SellPriceFor(s.stars(row.Slug))
			case domain.MetricStars:
				if s.stars(row.Slug) >= 4 {
					values[row.UserID]++
				}
			}
		}
	}
	board := Board{Metric: metric, Standings: make([]Standing, 0, len(values))}
	for id, v := range values {
		board.Standings = append(board.Standings, Standing{UserID: id, Value: v})
	}
	sort.Slice(board.Standings, func(i, j int) bool {
		if board.Standings[i].Value != board.Standings[j].Value {
			return board.Standings[i].Value > board.Standings[j].Value
		}
		return board.Standings[i].UserID < board.Standings[j].UserID
	})
	return board, nil
}

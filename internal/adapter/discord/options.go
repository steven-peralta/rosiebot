package discord

import (
	"strings"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	optSeries   = "series"
	optSort     = "sort"
	optMinStars = "min_stars"
	optMinLikes = "min_likes"
	optMaxTrash = "max_trash"
	optRanked   = "ranked"

	slugChoicePrefix = "slug:"
	maxSuggestions   = 25
)

type sortChoice struct {
	value string
	label string
	field app.SortField
	desc  bool
}

var searchSorts = []sortChoice{
	{value: "rank_asc", label: "Rank: best first", field: app.SortRank},
	{value: "rank_desc", label: "Rank: worst first", field: app.SortRank, desc: true},
	{value: "stars_desc", label: "Stars: most first", field: app.SortStars, desc: true},
	{value: "likes_desc", label: "Likes: most first", field: app.SortLikes, desc: true},
	{value: "likes_asc", label: "Likes: fewest first", field: app.SortLikes},
	{value: "trash_desc", label: "Trash: most first", field: app.SortTrash, desc: true},
	{value: "trash_asc", label: "Trash: fewest first", field: app.SortTrash},
	{value: "total_desc", label: "Votes: most first", field: app.SortTotal, desc: true},
	{value: "name_asc", label: "Name: A to Z", field: app.SortName},
	{value: "name_desc", label: "Name: Z to A", field: app.SortName, desc: true},
}

var ownedSorts = []sortChoice{
	{value: "oldest", label: "Oldest first"},
	{value: "newest", label: "Newest first"},
	{value: "rank_asc", label: "Rank: best first", field: app.SortRank},
	{value: "likes_desc", label: "Likes: most first", field: app.SortLikes, desc: true},
	{value: "name_asc", label: "Name: A to Z", field: app.SortName},
}

func choicesFor(sorts []sortChoice) []*discordgo.ApplicationCommandOptionChoice {
	out := make([]*discordgo.ApplicationCommandOptionChoice, len(sorts))
	for i, s := range sorts {
		out[i] = &discordgo.ApplicationCommandOptionChoice{Name: s.label, Value: s.value}
	}
	return out
}

func findSort(sorts []sortChoice, value string) (sortChoice, bool) {
	for _, s := range sorts {
		if s.value == value {
			return s, true
		}
	}
	return sortChoice{}, false
}

func starChoices() []*discordgo.ApplicationCommandOptionChoice {
	out := make([]*discordgo.ApplicationCommandOptionChoice, 0, domain.MaxStars)
	for n := domain.MaxStars; n >= 1; n-- {
		out = append(out, &discordgo.ApplicationCommandOptionChoice{Name: strings.Repeat("★", n) + strings.Repeat("☆", domain.MaxStars-n), Value: n})
	}
	return out
}

func searchOptions() []*discordgo.ApplicationCommandOption {
	minZero := float64(0)
	return []*discordgo.ApplicationCommandOption{
		{Type: discordgo.ApplicationCommandOptionString, Name: optQuery, Description: "Name to search for", Required: true, Autocomplete: true, MaxLength: 100},
		{Type: discordgo.ApplicationCommandOptionString, Name: optSeries, Description: "Limit results to one series", Autocomplete: true, MaxLength: 100},
		{Type: discordgo.ApplicationCommandOptionString, Name: optSort, Description: "Order of the results (default: best rank first, then name)", Choices: choicesFor(searchSorts)},
		{Type: discordgo.ApplicationCommandOptionInteger, Name: optMinStars, Description: "Only characters rated at least this many stars", Choices: starChoices()},
		{Type: discordgo.ApplicationCommandOptionInteger, Name: optMinLikes, Description: "Only characters with at least this many likes", MinValue: &minZero},
		{Type: discordgo.ApplicationCommandOptionInteger, Name: optMaxTrash, Description: "Only characters with at most this many trash votes", MinValue: &minZero},
		{Type: discordgo.ApplicationCommandOptionBoolean, Name: optRanked, Description: "True: only ranked characters. False: only unranked ones"},
	}
}

func listOptions() []*discordgo.ApplicationCommandOption {
	return searchOptions()[1:]
}

func seriesChoices(series []domain.Series) []*discordgo.ApplicationCommandOptionChoice {
	out := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(series))
	for _, s := range series {
		value := slugChoicePrefix + s.Slug
		if len(value) > maxChoiceLength || s.Name == "" {
			continue
		}
		out = append(out, &discordgo.ApplicationCommandOptionChoice{Name: truncate(s.Name, maxChoiceLength), Value: value})
	}
	return out
}

func intOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) (int, bool) {
	o := option(opts, name)
	if o == nil {
		return 0, false
	}
	switch v := o.Value.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}

func queryFromOptions(opts []*discordgo.ApplicationCommandInteractionDataOption) app.Query {
	q := app.Query{Term: stringOption(opts, optQuery)}
	if s, ok := findSort(searchSorts, stringOption(opts, optSort)); ok {
		q.SortBy, q.Descending = s.field, s.desc
	}
	if n, ok := intOption(opts, optMinStars); ok && n > 0 {
		q = q.WithFilter(app.SortStars, ">=", n)
	}
	if n, ok := intOption(opts, optMinLikes); ok {
		q = q.WithFilter(app.SortLikes, ">=", n)
	}
	if n, ok := intOption(opts, optMaxTrash); ok {
		q = q.WithFilter(app.SortTrash, "<=", n)
	}
	if o := option(opts, optRanked); o != nil {
		if v, ok := o.Value.(bool); ok {
			if v {
				q = q.WithFilter(app.SortRank, ">=", 1)
			} else {
				q = q.WithFilter(app.SortRank, app.OpAbsent, 0)
			}
		}
	}
	return q
}

func hasSearchOptions(opts []*discordgo.ApplicationCommandInteractionDataOption) bool {
	for _, name := range []string{optSeries, optSort, optMinStars, optMinLikes, optMaxTrash, optRanked} {
		if option(opts, name) != nil {
			return true
		}
	}
	return false
}

func directSlug(query string) (string, bool) {
	if strings.HasPrefix(query, slugChoicePrefix) && len(query) > len(slugChoicePrefix) {
		return strings.TrimPrefix(query, slugChoicePrefix), true
	}
	return "", false
}

func suggestionChoices(rows []domain.RankedWaifu) []*discordgo.ApplicationCommandOptionChoice {
	out := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(rows))
	for _, r := range rows {
		value := slugChoicePrefix + r.Slug
		if len(value) > maxChoiceLength {
			continue
		}
		label := r.Name
		if r.Stars > 0 {
			label += " · " + strings.Repeat("★", r.Stars)
		}
		out = append(out, &discordgo.ApplicationCommandOptionChoice{Name: truncate(label, maxChoiceLength), Value: value})
	}
	return out
}

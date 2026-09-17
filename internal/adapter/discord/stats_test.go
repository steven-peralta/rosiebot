package discord

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestProfile_Card(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "ranked-000", "ranked-100", "plain")
	f.give(bobID)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil))
	f.bot.WaitBackground()
	f.run(f.click(aliceID, f.waifuMessage("c1", "rem"), favPrefix+"waifu"))

	f.run(f.slash(aliceID, commandProfile, "", nil))
	e := (*f.api.lastEdit().Embeds)[0]
	if e.Title != "alice's profile" || e.Footer == nil {
		t.Fatalf("profile = %+v", e)
	}
	byName := map[string]string{}
	for _, fld := range e.Fields {
		byName[fld.Name] = fld.Value
	}
	if byName["Coins"] != ":coin: 0" || byName["Collection"] != "4 waifus · worth :coin: 1,350" || byName["Favorites"] != "1" {
		t.Errorf("profile fields = %+v", byName)
	}
	if byName["By rating"] != "★★★★★ 1\n★☆☆☆☆ 1\n☆☆☆☆☆ 2 unranked" {
		t.Errorf("rating breakdown = %q", byName["By rating"])
	}
	if byName["Rarest pull"] != "★★★★★ Ranked 000 · Rank #1" || byName["Rolls"] != "1 total\nLast: Name rem (regular)" {
		t.Errorf("rarest/rolls = %q / %q", byName["Rarest pull"], byName["Rolls"])
	}
	if byName["Server rank"] != "#1 by collection value · #1 by collection size · of 2 players" {
		t.Errorf("server rank = %q", byName["Server rank"])
	}

	f.run(f.slash(aliceID, commandProfile, "", resolvedUsers(bobID), userOption(bobID)))
	e = (*f.api.lastEdit().Embeds)[0]
	for _, fld := range e.Fields {
		if fld.Name == "By rating" && fld.Value != "No waifus yet." {
			t.Errorf("bob's rating breakdown = %q", fld.Value)
		}
		if fld.Name == "Rolls" && fld.Value != "0 total" {
			t.Errorf("bob's rolls = %q", fld.Value)
		}
	}
	if e.Title != "bob's profile" {
		t.Errorf("bob's title = %q", e.Title)
	}
	withAvatar := profileEmbed(&discordgo.User{ID: "x", Username: "x", GlobalName: "Xavier", Avatar: "abc"}, app.Profile{})
	if withAvatar.Title != "Xavier's profile" || withAvatar.Thumbnail == nil {
		t.Errorf("avatar embed = %+v", withAvatar)
	}
}

func TestLeaderboard_Rankings(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, commandLeaderboard, "", nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgLeaderboardEmpty {
		t.Errorf("empty = %q", got)
	}

	for i := range 12 {
		id := fmt.Sprintf("p%02d", i)
		f.give(id, fmt.Sprintf("ranked-%03d", i))
		if _, err := f.players.SetCoins(t.Context(), domain.PlayerKey{GuildID: guildID, UserID: id}, int64(1000-i*10)); err != nil {
			t.Fatal(err)
		}
	}
	f.give(aliceID)
	f.run(f.slash(aliceID, commandLeaderboard, "", nil))
	e := (*f.api.lastEdit().Embeds)[0]
	lines := strings.Split(e.Description, "\n")
	if e.Title != "Leaderboard · Collection value" || len(lines) != 12 || lines[0] != "1. <@p00> · :coin: 1,000" || lines[1] != "2. <@p01> · :coin: 1,000" || lines[9] != "10. <@p09> · :coin: 500" || lines[11] != "You're #13 · :coin: 0" {
		t.Errorf("value board = %q", lines)
	}
	f.run(f.slash(aliceID, commandLeaderboard, "", nil, strOpt(optBy, string(domain.MetricCoins))))
	e = (*f.api.lastEdit().Embeds)[0]
	lines = strings.Split(e.Description, "\n")
	if e.Title != "Leaderboard · Coins" || lines[0] != "1. <@p00> · :coin: 1,000" || lines[11] != "You're #13 · :coin: 200" {
		t.Errorf("coins board = %q", lines)
	}
	f.run(f.slash("p03", commandLeaderboard, "", nil, strOpt(optBy, string(domain.MetricStars))))
	e = (*f.api.lastEdit().Embeds)[0]
	lines = strings.Split(e.Description, "\n")
	if e.Title != "Leaderboard · 4-star and 5-star waifus" || lines[0] != "1. <@p00> · 1" || len(lines) != 10 {
		t.Errorf("stars board = %q", lines)
	}
	f.run(f.slash(aliceID, commandLeaderboard, "", nil, strOpt(optBy, "fame")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgNoData {
		t.Errorf("bad metric = %q", got)
	}
	if metricLabel(domain.Metric("odd")) != "odd" {
		t.Error("unknown metric label falls back to its name")
	}
}

func TestHistory_RecentRolls(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, commandWaifu, subHistory, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> Your roll history is empty." {
		t.Errorf("empty = %q", got)
	}
	f.fund(aliceID, 200)
	f.script(d100(50), d100(5), 3)
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil))
	f.bot.WaitBackground()
	f.clock.now = f.clock.now.Add(90 * time.Minute)
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil))
	f.bot.WaitBackground()
	f.clock.now = f.clock.now.Add(5 * time.Minute)

	f.run(f.slash(bobID, commandWaifu, subHistory, resolvedUsers(aliceID), userOption(aliceID)))
	e := f.api.lastEdit()
	if got := editContent(e); got != "<@bob> <@alice>'s last 2 of 2 rolls" {
		t.Errorf("history header = %q", got)
	}
	lines := strings.Split((*e.Embeds)[0].Description, "\n")
	if len(lines) != 2 || lines[0] != "★★★★☆ Ranked 003 · critical · 5m ago" || lines[1] != "☆☆☆☆☆ Name rem · regular · 1h 35m ago" {
		t.Errorf("history lines = %q", lines)
	}
	for _, k := range []domain.RollKind{domain.RollWaifuOfTheDay, domain.RollBanner, domain.RollKind(9)} {
		if rollKindLabel(k) == "" {
			t.Error("labels must not be empty")
		}
	}
}

func TestOwned_SeriesFilterAndCompletion(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem", "ram", "emilia", "other")
	f.source.EXPECT().Work(mock.Anything, "re-zero").Return(reZero, nil).Times(2)
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("rem"), summary("ram"), summary("beatrice"), summary("ranked-000")}}, nil).Times(2)

	f.run(f.slash(aliceID, commandWaifu, subOwned, nil, strOpt(optSeries, slugChoicePrefix+"re-zero"), strOpt(optView, viewCompact)))
	e := f.api.lastEdit()
	if got := editContent(e); !strings.HasPrefix(got, "<@alice> from **Re:Zero**: 2 of 4 characters 2 waifus") {
		t.Errorf("owned by series = %q", got)
	}
	lines := strings.Split((*e.Embeds)[0].Description, "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "Name rem") || !strings.Contains(lines[1], "Name ram") {
		t.Errorf("filtered list = %q", lines)
	}

	f.run(f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, slugChoicePrefix+"re-zero")))
	card := (*f.api.lastEdit().Embeds)[0]
	if len(card.Fields) != 2 || card.Fields[1].Name != "Your collection" || card.Fields[1].Value != "0 of 1 ranked · 2 of 4 characters" {
		t.Errorf("completion field = %+v", card.Fields)
	}
	if got := completionText([]cardCharacter{{summary: domain.WaifuSummary{Slug: "a"}}}, map[string]struct{}{"a": {}}); got != "1 of 1 characters · complete!" {
		t.Errorf("complete text = %q", got)
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "nothing").Return(nil, nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil, strOpt(optSeries, "nothing")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgSeriesNotFound {
		t.Errorf("unknown series = %q", got)
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "re").Return([]domain.Series{reZero}, nil).Once()
	focused := strOpt(optSeries, "re")
	focused.Focused = true
	ic := f.slash(aliceID, commandWaifu, subOwned, nil, focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if r := f.api.lastRespond(); r.Type != discordgo.InteractionApplicationCommandAutocompleteResult || len(r.Data.Choices) != 1 {
		t.Errorf("owned series autocomplete = %+v", r)
	}
}

func TestFavorites_SeriesCardsView(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().Work(mock.Anything, "re-zero").Return(reZero, nil).Once()
	f.source.EXPECT().Work(mock.Anything, "konosuba").Return(domain.Series{Slug: "konosuba", Name: "KonoSuba", URL: "https://mywaifulist.moe/series/konosuba"}, nil).Once()
	f.run(f.click(aliceID, f.seriesMessage("s1", "re-zero", "Re:Zero"), favPrefix+"series"))
	f.clock.now = f.clock.now.Add(time.Second)
	f.run(f.click(aliceID, f.seriesMessage("s2", "konosuba", "KonoSuba"), favPrefix+"series"))

	ic := f.slash(aliceID, commandFavs, subFavSeries, nil, strOpt(optView, viewCards))
	f.run(ic)
	e := f.api.lastEdit()
	if !strings.HasSuffix(editContent(e), "Page 1 out of 2") || (*e.Embeds)[0].Title != "Re:Zero" || (*e.Embeds)[0].Image == nil {
		t.Fatalf("series cards = %q %+v", editContent(e), (*e.Embeds)[0])
	}
	rows := *e.Components
	last := rows[len(rows)-1].(discordgo.ActionsRow).Components
	if len(last) != 1 || last[0].(discordgo.Button).CustomID != favPrefix+"series" {
		t.Errorf("series card actions = %+v", last)
	}
	menu := rows[1].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if len(menu.Options) != 2 || menu.Options[1].Label != "2. KonoSuba" {
		t.Errorf("series select = %+v", menu.Options)
	}
	f.run(f.click(aliceID, f.message("msg-"+ic.ID), pagerPrefix+pagerNext))
	if r := f.api.lastRespond(); r.Data.Embeds[0].Title != "KonoSuba" {
		t.Errorf("second series page = %+v", r.Data.Embeds[0])
	}
}

package discord

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

var reZero = domain.Series{Slug: "re-zero", Name: "Re:Zero", URL: "https://www.mywaifulist.moe/series/re-zero", PictureURL: "https://img/re-zero", Description: "A boy is summoned."}

func TestSeriesSearch_Card(t *testing.T) {
	f := newFixture(t)
	items := []domain.WaifuSummary{summary("ram"), summary("ranked-050"), summary("rem"), summary("ranked-000")}
	f.source.EXPECT().SearchWorks(mock.Anything, "re zero").Return([]domain.Series{reZero}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: items}, nil).Once()

	ic := f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, "re zero"))
	f.run(ic)
	e := f.api.lastEdit()
	if got := editContent(e); got != "<@alice> "+msgSeriesFound {
		t.Errorf("content = %q", got)
	}
	embed := (*e.Embeds)[0]
	if embed.Title != "Re:Zero" || embed.URL != reZero.URL || embed.Image == nil || embed.Description != "A boy is summoned." || embed.Footer == nil {
		t.Errorf("embed = %+v", embed)
	}
	if len(embed.Fields) != 2 || embed.Fields[0].Name != "Characters · 2 ranked of 4" || embed.Fields[1].Name != "Your collection" || embed.Fields[1].Value != "0 of 2 ranked · 0 of 4 characters" {
		t.Fatalf("fields = %+v", embed.Fields)
	}
	lines := strings.Split(embed.Fields[0].Value, "\n")
	want := []string{
		"★★★★★ Name ranked-000 · Rank #1",
		"★★☆☆☆ Name ranked-050 · Rank #51",
		"☆☆☆☆☆ Name ram · unranked",
		"☆☆☆☆☆ Name rem · unranked",
	}
	if strings.Join(lines, "|") != strings.Join(want, "|") {
		t.Errorf("lines = %q", lines)
	}
	if len(*e.Components) != 1 || len((*e.Components)[0].(discordgo.ActionsRow).Components) != 2 {
		t.Errorf("series card should carry browse and favorite buttons in one row, got %+v", *e.Components)
	}
	if fav := (*e.Components)[0].(discordgo.ActionsRow).Components[1].(discordgo.Button); fav.CustomID != favPrefix+"series" {
		t.Errorf("series favorite button = %+v", fav)
	}
	button := (*e.Components)[0].(discordgo.ActionsRow).Components[0].(discordgo.Button)
	if button.CustomID != seriesPrefix+seriesBrowse+":re-zero" || button.Label != "Browse characters" {
		t.Errorf("button = %+v", button)
	}
	card := f.message("msg-" + ic.ID)

	f.source.EXPECT().Work(mock.Anything, "re-zero").Return(reZero, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: items}, nil).Once()
	f.run(f.click(bobID, card, button.CustomID))
	if r := f.api.calls[len(f.api.calls)-2].resp; r == nil || r.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Errorf("browse should reply with a new message, got %+v", r)
	}
	e = f.api.lastEdit()
	if got := editContent(e); got != "<@bob> Showing results for series Re:Zero\nPage 1 out of 4" || (*e.Embeds)[0].Title == "Re:Zero" {
		t.Errorf("browse = %q", got)
	}
	f.run(f.click(bobID, card, seriesPrefix+"bogus:x"))
	f.run(f.click(bobID, card, seriesPrefix+seriesBrowse+":"))

	f.run(f.selectValues(bobID, card, viewWaifuMenu, "rem"))
	if r := f.api.calls[len(f.api.calls)-2].resp; r == nil || r.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource || r.Data == nil || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Errorf("viewing a character should be a deferred ephemeral reply, got %+v", r)
	}
	if e := f.api.lastEdit(); editContent(e) != "" || (*e.Embeds)[0].Title != "Name rem" || (*e.Embeds)[0].Footer == nil {
		t.Errorf("viewed card = %+v", (*e.Embeds)[0])
	}
	f.api.reset()
	f.run(f.selectValues(bobID, card, viewWaifuMenu))
	if len(f.api.calls) != 0 {
		t.Error("an empty selection should be ignored")
	}
	f.source.ExpectedCalls = nil
	f.source.EXPECT().Get(mock.Anything, "ghost").Return(domain.Waifu{}, app.ErrNotFound).Once()
	f.run(f.selectValues(bobID, card, viewWaifuMenu, "ghost"))
	if got := editContent(f.api.lastEdit()); got != "<@bob> "+msgNoData {
		t.Errorf("missing character = %q", got)
	}
}

func TestSeriesAlias_SRoutesLikeSeries(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWorks(mock.Anything, "nothing").Return(nil, nil).Once()
	f.run(f.slash(aliceID, commandSAlias, subSearch, nil, strOpt(optQuery, "nothing")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgSeriesNotFound {
		t.Errorf("/s search = %q", got)
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "re").Return([]domain.Series{reZero}, nil).Once()
	focused := strOpt(optQuery, "re")
	focused.Focused = true
	ic := f.slash(aliceID, commandSAlias, subSearch, nil, focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if r := f.api.lastRespond(); r.Type != discordgo.InteractionApplicationCommandAutocompleteResult || len(r.Data.Choices) != 1 {
		t.Errorf("/s autocomplete = %+v", r)
	}
}

func TestSeriesSearch_DirectSlugNotFoundAndErrors(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().Work(mock.Anything, "re-zero").Return(reZero, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	f.run(f.dm(aliceID, commandSeries, subSearch, strOpt(optQuery, slugChoicePrefix+"re-zero")))
	e := f.api.lastEdit()
	if editContent(e) != "<@alice> "+msgSeriesFound || (*e.Embeds)[0].Title != "Re:Zero" || len((*e.Embeds)[0].Fields) != 0 || len(*e.Components) != 1 || len((*e.Components)[0].(discordgo.ActionsRow).Components) != 1 {
		t.Errorf("empty series card = %q %+v %+v", editContent(e), (*e.Embeds)[0], *e.Components)
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "nothing").Return(nil, nil).Once()
	f.run(f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, "nothing")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgSeriesNotFound {
		t.Errorf("not found = %q", got)
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	f.run(f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, "boom")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgUnexpected {
		t.Errorf("error = %q", got)
	}
	f.run(f.slash(aliceID, commandSeries, "nope", nil))
}

func TestSeriesSearch_Autocomplete(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWorks(mock.Anything, "re").Return([]domain.Series{reZero}, nil).Once()
	focused := strOpt(optQuery, "re")
	focused.Focused = true
	ic := f.slash(aliceID, commandSeries, subSearch, nil, focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	r := f.api.lastRespond()
	if r.Type != discordgo.InteractionApplicationCommandAutocompleteResult || len(r.Data.Choices) != 1 || r.Data.Choices[0].Value != slugChoicePrefix+"re-zero" {
		t.Errorf("autocomplete = %+v", r)
	}
	ic = f.slash(aliceID, commandSeries, "nope", nil, focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
}

func TestStarBar(t *testing.T) {
	cases := map[int]string{0: "☆☆☆☆☆", 1: "★☆☆☆☆", 3: "★★★☆☆", 5: "★★★★★", 9: "★★★★★", -1: "☆☆☆☆☆"}
	for stars, want := range cases {
		if got := starBar(stars); got != want {
			t.Errorf("starBar(%d) = %q, want %q", stars, got, want)
		}
	}
}

func TestSeriesCardEmbed_CapsList(t *testing.T) {
	items := make([]domain.WaifuSummary, 20)
	for i := range items {
		items[i] = summary(fmt.Sprintf("ranked-%03d", i))
	}
	e := seriesCardEmbed(reZero, cardCharacters(items, app.LookupFrom(newFixture(t).ranking)))
	lines := strings.Split(e.Fields[0].Value, "\n")
	if len(lines) != domain.BannerCardLimit+1 || lines[len(lines)-1] != "+5 more" || e.Fields[0].Name != "Characters · 20 ranked of 20" {
		t.Errorf("field = %+v", e.Fields[0])
	}
}

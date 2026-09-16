package discord

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestCommands_Registration(t *testing.T) {
	f := newFixture(t)
	cmds := f.bot.Commands()
	if len(cmds) != 4 {
		t.Fatalf("commands = %d", len(cmds))
	}
	names := map[string][]string{}
	for _, c := range cmds {
		for _, o := range c.Options {
			names[c.Name] = append(names[c.Name], o.Name)
		}
	}
	want := []string{subRoll, subDaily, subCoins, subOwned, subSearch, subRandom, subToday, subTrade}
	if strings.Join(names[commandWaifu], ",") != strings.Join(want, ",") {
		t.Errorf("waifu subcommands = %v", names[commandWaifu])
	}
	if strings.Join(names[commandWAlias], ",") != strings.Join(want, ",") {
		t.Errorf("/w alias subcommands = %v", names[commandWAlias])
	}
	if strings.Join(names[commandSeries], ",") != subSearch {
		t.Errorf("series subcommands = %v", names[commandSeries])
	}
	if cmds[3].Type != discordgo.MessageApplicationCommand || cmds[3].Name != commandSell {
		t.Errorf("context command = %+v", cmds[3])
	}
	if err := f.bot.Register(guildID); err != nil {
		t.Fatal(err)
	}
	if f.api.last().kind != "register" {
		t.Error("Register should bulk overwrite")
	}
}

func TestDMGating_PerSubcommand(t *testing.T) {
	f := newFixture(t)
	for _, sub := range []string{subRoll, subDaily, subCoins, subOwned, subTrade} {
		f.api.reset()
		f.run(f.dm(aliceID, commandWaifu, sub))
		r := f.api.lastRespond()
		if r == nil || r.Data == nil || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 || r.Data.Content != "The "+sub+" command cannot be invoked from the direct messages of the bot." {
			t.Errorf("%s in DM -> %+v", sub, r)
		}
	}
	f.api.reset()
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.dm(aliceID, commandWaifu, subRandom))
	if f.api.lastEdit() == nil || len(*f.api.lastEdit().Embeds) != 1 {
		t.Error("random should work in DMs")
	}
}

func TestRoll_Texts(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil))
	e := f.api.lastEdit()
	if editContent(e) != "<@alice> Here's who you rolled:\n" || len(*e.Embeds) != 1 || (*e.Embeds)[0].Title != "Name rem" {
		t.Errorf("regular roll = %q embeds=%+v", editContent(e), *e.Embeds)
	}
	if f.coins(aliceID) != 0 || !f.owns(aliceID, "rem") {
		t.Error("roll should charge and grant")
	}

	f.run(f.slash(aliceID, commandWaifu, subRoll, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> You don't have enough coins!" {
		t.Errorf("broke roll = %q", got)
	}

	f.give(bobID)
	f.script(d100(5), 3)
	f.run(f.slash(bobID, commandWaifu, subRoll, nil))
	e = f.api.lastEdit()
	if editContent(e) != "<@bob> :sparkles: **CRITICAL ROLL!!** :sparkles: Here's who you rolled:\n" {
		t.Errorf("critical roll = %q", editContent(e))
	}
	if desc := (*e.Embeds)[0].Description; !strings.Contains(desc, "★") {
		t.Errorf("critical roll embed should show stars: %q", desc)
	}

	f.give("carol")
	f.script(d100(1), 7)
	f.run(f.slash("carol", commandWaifu, subRoll, nil))
	if got := editContent(f.api.lastEdit()); got != "<@carol> :star2: **You rolled the Waifu of the Day. Congrats!** Here's who you rolled:\n" {
		t.Errorf("wotd roll = %q", got)
	}
}

func TestAlias_WRoutesLikeWaifu(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWAlias, subRoll, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> Here's who you rolled:\n" {
		t.Errorf("/w roll = %q", got)
	}
	f.give(bobID, "ram")
	focused := strOpt(optGive, "")
	focused.Focused = true
	ic := f.slash(bobID, commandWAlias, subTrade, resolvedUsers(aliceID), userOption(aliceID), focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if r := f.api.lastRespond(); r.Type != discordgo.InteractionApplicationCommandAutocompleteResult || len(r.Data.Choices) != 1 {
		t.Errorf("/w trade autocomplete = %+v", r)
	}
	f.run(f.dm(aliceID, commandWAlias, subDaily))
	if got := f.respondContent(); got != "The daily command cannot be invoked from the direct messages of the bot." {
		t.Errorf("/w in DM = %q", got)
	}
}

func TestRoll_ExhaustedText(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem")
	for range domain.MaxRerollAttempts {
		f.script(d100(50))
	}
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Times(domain.MaxRerollAttempts)
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgRollExhausted {
		t.Errorf("exhausted = %q", got)
	}
}

func TestDaily_Texts(t *testing.T) {
	f := newFixture(t)
	f.script(d100(50))
	f.run(f.slash(aliceID, commandWaifu, subDaily, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> You claimed :coin: 400 coins" {
		t.Errorf("daily = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subDaily, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> You've already claimed your daily for today. You can claim again in 22:00:00" {
		t.Errorf("repeat daily = %q", got)
	}
	f.give(bobID)
	f.script(d100(1))
	f.run(f.slash(bobID, commandWaifu, subDaily, nil))
	if got := editContent(f.api.lastEdit()); got != "<@bob> :sparkles: **CRITICAL ROLL!!** :sparkles: You claimed :coin: 2000 coins!" {
		t.Errorf("critical daily = %q", got)
	}
}

func TestCoins_SingularPlural(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, commandWaifu, subCoins, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> You have :coin: 200 coins" {
		t.Errorf("self = %q", got)
	}
	f.give(bobID)
	if _, ok, _ := f.players.DebitCoins(t.Context(), domain.PlayerKey{GuildID: guildID, UserID: bobID}, 199); !ok {
		t.Fatal("setup")
	}
	f.run(f.slash(aliceID, commandWaifu, subCoins, resolvedUsers(bobID), userOption(bobID)))
	if got := editContent(f.api.lastEdit()); got != "<@alice>: <@bob> has :coin: 1 coin" {
		t.Errorf("target = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subCoins, resolvedUsers(aliceID), userOption(aliceID)))
	if got := editContent(f.api.lastEdit()); got != "<@alice> You have :coin: 200 coins" {
		t.Errorf("self via option = %q", got)
	}
}

func TestOwned_EmptyAndPager(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgOwnsNothing {
		t.Errorf("empty = %q", got)
	}

	f.give(aliceID, "a", "b", "c")
	ic := f.slash(aliceID, commandWaifu, subOwned, nil)
	f.run(ic)
	e := f.api.lastEdit()
	if editContent(e) != "<@alice>\nPage 1 out of 3" || len(*e.Components) != 2 || (*e.Embeds)[0].Title != "Name a" {
		t.Errorf("first page = %q comps=%v title=%q", editContent(e), hasComponents(*e.Components), (*e.Embeds)[0].Title)
	}
	msgID := "msg-" + ic.ID
	if f.bot.Sessions().Len() != 1 {
		t.Fatal("pager session should be stored")
	}

	msg := f.message(msgID)
	f.run(f.click(aliceID, msg, pagerPrefix+pagerNext))
	r := f.api.lastRespond()
	if r.Type != discordgo.InteractionResponseUpdateMessage || r.Data.Content != "<@alice>\nPage 2 out of 3" || r.Data.Embeds[0].Title != "Name b" {
		t.Errorf("next = %+v", r.Data)
	}
	f.run(f.click(aliceID, msg, pagerPrefix+pagerLast))
	if r := f.api.lastRespond(); r.Data.Content != "<@alice>\nPage 3 out of 3" {
		t.Errorf("last = %q", r.Data.Content)
	}
	f.run(f.click(aliceID, msg, pagerPrefix+pagerNext))
	if r := f.api.lastRespond(); r.Data.Content != "<@alice>\nPage 3 out of 3" {
		t.Errorf("next past end should clamp: %q", r.Data.Content)
	}
	f.run(f.click(aliceID, msg, pagerPrefix+pagerPrev))
	if r := f.api.lastRespond(); r.Data.Content != "<@alice>\nPage 2 out of 3" {
		t.Errorf("prev = %q", r.Data.Content)
	}
	f.run(f.click(aliceID, msg, pagerPrefix+pagerFirst))
	if r := f.api.lastRespond(); r.Data.Content != "<@alice>\nPage 1 out of 3" {
		t.Errorf("first = %q", r.Data.Content)
	}

	f.run(f.click(bobID, msg, pagerPrefix+pagerNext))
	if got := f.respondContent(); got != msgNotYourMenu {
		t.Errorf("other user = %q", got)
	}

	f.run(f.click(aliceID, msg, pagerPrefix+pagerJump))
	if r := f.api.lastRespond(); r.Type != discordgo.InteractionResponseModal || r.Data.CustomID != pagerJumpModal {
		t.Errorf("jump should open a modal: %+v", r)
	}
	f.run(f.modal(aliceID, msg, pagerJumpModal, pagerJumpInput, " 3 "))
	if r := f.api.lastRespond(); r.Data.Content != "<@alice>\nPage 3 out of 3" {
		t.Errorf("jump submit = %q", r.Data.Content)
	}
	f.run(f.modal(aliceID, msg, pagerJumpModal, pagerJumpInput, "9"))
	if got := f.respondContent(); got != "Please enter a page number between 1 and 3." {
		t.Errorf("bad jump = %q", got)
	}
	f.run(f.modal(aliceID, msg, pagerJumpModal, "other", "1"))
	if got := f.respondContent(); !strings.HasPrefix(got, "Please enter") {
		t.Errorf("missing input = %q", got)
	}
	f.run(f.click(aliceID, msg, pagerPrefix+"bogus"))

	f.clock.now = f.clock.now.Add(time.Hour)
	f.run(f.click(aliceID, msg, pagerPrefix+pagerNext))
	if got := f.respondContent(); got != msgExpired {
		t.Errorf("expired = %q", got)
	}
	if me := f.api.lastMsgEdit(); me == nil || len(*me.Components) != 0 {
		t.Error("expired menu should have its components stripped")
	}
	f.run(f.modal(aliceID, msg, pagerJumpModal, pagerJumpInput, "1"))
	if got := f.respondContent(); got != msgExpired {
		t.Errorf("expired modal = %q", got)
	}
}

func TestOwned_SingleResultHasNoPager(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "only")
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil))
	e := f.api.lastEdit()
	if editContent(e) != "<@alice>" || len(*e.Components) != 1 || f.bot.Sessions().Len() != 0 {
		t.Errorf("single = %q comps=%d sessions=%d", editContent(e), len(*e.Components), f.bot.Sessions().Len())
	}
}

func TestOwned_TargetUser(t *testing.T) {
	f := newFixture(t)
	f.give(bobID, "x")
	f.run(f.slash(aliceID, commandWaifu, subOwned, resolvedUsers(bobID), userOption(bobID)))
	if (*f.api.lastEdit().Embeds)[0].Title != "Name x" {
		t.Error("should show target's collection")
	}
}

func TestSearch_Texts(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWaifus(mock.Anything, "nobody", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, "nobody")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgWaifuNotFound {
		t.Errorf("no results = %q", got)
	}

	f.source.EXPECT().SearchWaifus(mock.Anything, "rem", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("rem"), summary("ram")}}, nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, "rem")))
	e := f.api.lastEdit()
	if editContent(e) != "<@alice>\nPage 1 out of 2" || !hasComponents(*e.Components) {
		t.Errorf("results = %q", editContent(e))
	}

	f.source.EXPECT().SearchWaifus(mock.Anything, "boom", 1).Return(app.SearchPage{}, errors.New("boom")).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, "boom")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgUnexpected {
		t.Errorf("error = %q", got)
	}
}

func TestRandom_AndToday(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRandom, nil))
	e := f.api.lastEdit()
	if editContent(e) != "" || (*e.Embeds)[0].Title != "Name rem" || !strings.HasPrefix((*e.Embeds)[0].Footer.Text, "rosiebot vtest") {
		t.Errorf("random = %q %+v", editContent(e), (*e.Embeds)[0])
	}

	f.source.EXPECT().Random(mock.Anything).Return(domain.WaifuSummary{}, errors.New("boom")).Once()
	f.run(f.slash(aliceID, commandWaifu, subRandom, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgUnexpected {
		t.Errorf("random error = %q", got)
	}

	f.script(4)
	f.run(f.slash(aliceID, commandWaifu, subToday, nil))
	e = f.api.lastEdit()
	if editContent(e) != "<@alice> Here's the Waifu of the Day:\nRefreshes in 12:00:00" || len(*e.Embeds) != 1 {
		t.Errorf("today = %q", editContent(e))
	}
	first := (*e.Embeds)[0].Title
	f.run(f.slash(bobID, commandWaifu, subToday, nil))
	if (*f.api.lastEdit().Embeds)[0].Title != first {
		t.Error("waifu of the day must be the same for everyone")
	}
}

func TestToday_FallsBackToSummaryWhenDetailFails(t *testing.T) {
	f := newFixture(t)
	f.source.ExpectedCalls = nil
	f.source.EXPECT().Get(mock.Anything, mock.Anything).Return(domain.Waifu{}, errors.New("down")).Maybe()
	f.script(4)
	f.run(f.slash(aliceID, commandWaifu, subToday, nil))
	e := f.api.lastEdit()
	if len(*e.Embeds) != 1 || (*e.Embeds)[0].Title == "" {
		t.Errorf("fallback embed = %+v", *e.Embeds)
	}
}

func TestSeriesSearch_Texts(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWorks(mock.Anything, "nothing").Return(nil, nil).Once()
	f.run(f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, "nothing")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgSeriesNotFound {
		t.Errorf("not found = %q", got)
	}

	series := domain.Series{Slug: "re-zero", Name: "Re:Zero", URL: "https://www.mywaifulist.moe/series/re-zero", Description: "d"}
	f.source.EXPECT().SearchWorks(mock.Anything, "re zero").Return([]domain.Series{series}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("ram"), summary("rem")}}, nil).Once()
	f.run(f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, "re zero")))
	e := f.api.lastEdit()
	if editContent(e) != "<@alice> Showing results for series Re:Zero\nPage 1 out of 2" {
		t.Errorf("results = %q", editContent(e))
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "empty").Return([]domain.Series{series}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	f.run(f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, "empty")))
	e = f.api.lastEdit()
	if editContent(e) != "<@alice> "+msgNoData || (*e.Embeds)[0].Title != "Re:Zero" {
		t.Errorf("no characters = %q", editContent(e))
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	f.run(f.slash(aliceID, commandSeries, subSearch, nil, strOpt(optQuery, "boom")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgUnexpected {
		t.Errorf("error = %q", got)
	}
}

func TestRouter_IgnoresUnknown(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, "nope", "", nil))
	f.run(f.slash(aliceID, commandWaifu, "nope", nil))
	f.run(f.slash(aliceID, commandSeries, "nope", nil))
	f.run(&discordgo.InteractionCreate{Interaction: &discordgo.Interaction{Type: discordgo.InteractionPing}})
	f.run(f.click(aliceID, &discordgo.Message{ID: "x"}, "zz:unknown"))
	f.run(f.modal(aliceID, &discordgo.Message{ID: "x"}, "other-modal", "a", "b"))
	f.run(f.contextMenu(aliceID, nil))
	if len(f.api.calls) != 1 {
		t.Errorf("unknown interactions should be ignored (only the context menu replies), calls=%d", len(f.api.calls))
	}
}

func TestEditFailureIsTolerated(t *testing.T) {
	f := newFixture(t)
	f.api.failEdit = true
	f.give(aliceID, "a", "b")
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil))
	if f.bot.Sessions().Len() != 0 {
		t.Error("no session when the edit failed")
	}
}

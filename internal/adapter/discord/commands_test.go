package discord

import (
	"errors"
	"fmt"
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
	if len(cmds) != 5 {
		t.Fatalf("commands = %d", len(cmds))
	}
	names := map[string][]string{}
	for _, c := range cmds {
		for _, o := range c.Options {
			names[c.Name] = append(names[c.Name], o.Name)
		}
	}
	want := []string{subRoll, subDaily, subCoins, subOwned, subSearch, subList, subRandom, subToday, subBanner, subTrade}
	if strings.Join(names[commandWaifu], ",") != strings.Join(want, ",") {
		t.Errorf("waifu subcommands = %v", names[commandWaifu])
	}
	if roll := cmds[0].Options[0]; len(roll.Options) != 1 || roll.Options[0].Name != optBanner || roll.Options[0].Type != discordgo.ApplicationCommandOptionBoolean {
		t.Errorf("roll options = %+v", roll.Options)
	}
	if strings.Join(names[commandWAlias], ",") != strings.Join(want, ",") {
		t.Errorf("/w alias subcommands = %v", names[commandWAlias])
	}
	if cmds[2].Name != commandSeries || len(cmds[2].Options) != 1 || cmds[2].Options[0].Name != subSearch || !cmds[2].Options[0].Options[0].Required || !cmds[2].Options[0].Options[0].Autocomplete {
		t.Errorf("series command = %+v", cmds[2])
	}
	if cmds[3].Name != commandSAlias || len(cmds[3].Options) != 1 || cmds[3].Options[0].Name != subSearch {
		t.Errorf("/s alias = %+v", cmds[3])
	}
	if cmds[4].Type != discordgo.MessageApplicationCommand || cmds[4].Name != commandSell {
		t.Errorf("context command = %+v", cmds[4])
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
	f.run(f.dm(aliceID, commandWaifu, subRoll, boolOpt(optBanner, true)))
	if got := f.respondContent(); got != "The roll command cannot be invoked from the direct messages of the bot." {
		t.Errorf("banner roll in DM = %q", got)
	}
	f.api.reset()
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.dm(aliceID, commandWaifu, subRandom))
	if f.api.lastEdit() == nil || len(*f.api.lastEdit().Embeds) != 1 {
		t.Error("random should work in DMs")
	}
	f.source.EXPECT().SearchWorks(mock.Anything, "re zero").Return([]domain.Series{{Slug: "re-zero", Name: "Re:Zero"}}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("rem")}}, nil).Once()
	f.run(f.dm(aliceID, commandWaifu, subSearch, strOpt(optSeries, "re zero")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> Showing results for series Re:Zero" {
		t.Errorf("series search should work in DMs: %q", got)
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
	if title := (*e.Embeds)[0].Title; !strings.HasPrefix(title, ":star:") {
		t.Errorf("critical roll embed should show stars: %q", title)
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

func TestRoll_AgainButton(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID)
	if _, ok, err := f.players.ClaimDaily(t.Context(), domain.PlayerKey{GuildID: guildID, UserID: aliceID}, 200, f.clock.now.Add(-time.Hour), f.clock.now); err != nil || !ok {
		t.Fatal("setup coins")
	}
	f.script(d100(50), d100(60))
	f.source.EXPECT().Random(mock.Anything).Return(summary("first"), nil).Once()
	f.source.EXPECT().Random(mock.Anything).Return(summary("second"), nil).Once()

	ic := f.slash(aliceID, commandWaifu, subRoll, nil)
	f.run(ic)
	e := f.api.lastEdit()
	button := (*e.Components)[0].(discordgo.ActionsRow).Components[0].(discordgo.Button)
	if button.CustomID != rollPrefix+rollAgain+":"+aliceID || button.Label != "Roll again · 200 coins" {
		t.Fatalf("roll again button = %+v", button)
	}
	msg := f.message("msg-" + ic.ID)

	f.run(f.click(bobID, msg, button.CustomID))
	if got := f.respondContent(); got != msgNotYourRoll {
		t.Errorf("other user = %q", got)
	}

	f.run(f.click(aliceID, msg, button.CustomID))
	if r := f.api.lastRespond(); r.Type != discordgo.InteractionResponseDeferredMessageUpdate {
		t.Errorf("roll again should defer the message update: %+v", r)
	}
	e = f.api.lastEdit()
	if (*e.Embeds)[0].Title != "Name second" || !hasComponents(*e.Components) {
		t.Errorf("second roll = %+v", (*e.Embeds)[0])
	}
	if f.coins(aliceID) != 0 || !f.owns(aliceID, "first") || !f.owns(aliceID, "second") {
		t.Error("both rolls should have been charged and granted")
	}

	f.api.reset()
	f.run(f.click(aliceID, msg, button.CustomID))
	if f.api.last().kind != "followup" {
		t.Errorf("insufficient coins should be an ephemeral follow-up, got %s", f.api.last().kind)
	}
	if f.api.lastEdit() != nil {
		t.Error("failed roll again must not edit the card")
	}

	dm := f.click(aliceID, msg, button.CustomID)
	dm.GuildID = ""
	f.run(dm)
	if got := f.respondContent(); got != "The roll command cannot be invoked from the direct messages of the bot." {
		t.Errorf("dm = %q", got)
	}
	f.run(f.click(aliceID, msg, rollPrefix+"bogus:"+aliceID))
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
	if editContent(e) != "<@alice>\nPage 1 out of 3" || len(*e.Components) != 3 || (*e.Embeds)[0].Title != "Name a" {
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

func TestOwned_SortAndSelectMenu(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "b-old", "a-new")
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil))
	if (*f.api.lastEdit().Embeds)[0].Title != "Name b-old" {
		t.Error("default order should be acquisition order, oldest first")
	}
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil, strOpt(optSort, "newest")))
	if (*f.api.lastEdit().Embeds)[0].Title != "Name a-new" {
		t.Error("newest first")
	}
	ic := f.slash(aliceID, commandWaifu, subOwned, nil, strOpt(optSort, "name_asc"))
	f.run(ic)
	e := f.api.lastEdit()
	if (*e.Embeds)[0].Title != "Name a-new" || len(*e.Components) != 3 {
		t.Fatalf("name sort = %q rows=%d", (*e.Embeds)[0].Title, len(*e.Components))
	}
	menu := (*e.Components)[1].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if menu.CustomID != pagerPrefix+pagerSelect || len(menu.Options) != 2 || menu.Options[1].Label != "2. Name b-old" || !menu.Options[0].Default {
		t.Errorf("select menu = %+v", menu)
	}
	msg := f.message("msg-" + ic.ID)
	pick := f.click(aliceID, msg, pagerPrefix+pagerSelect)
	pick.Data = discordgo.MessageComponentInteractionData{CustomID: pagerPrefix + pagerSelect, Values: []string{"1"}}
	f.run(pick)
	if r := f.api.lastRespond(); r.Data.Content != "<@alice>\nPage 2 out of 2" || r.Data.Embeds[0].Title != "Name b-old" {
		t.Errorf("select jump = %+v", r.Data)
	}
	bad := f.click(aliceID, msg, pagerPrefix+pagerSelect)
	bad.Data = discordgo.MessageComponentInteractionData{CustomID: pagerPrefix + pagerSelect, Values: []string{"nope"}}
	f.api.reset()
	f.run(bad)
	if len(f.api.calls) != 0 {
		t.Error("malformed select value should be ignored")
	}
	f.give(aliceID, "ranked-001")
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil, strOpt(optSort, "rank_asc")))
	if cardName((*f.api.lastEdit().Embeds)[0]) != "Name ranked-001" {
		t.Error("rank sort should put the ranked waifu first")
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

func TestSearch_TypedOptions(t *testing.T) {
	f := newFixture(t)
	items := []domain.WaifuSummary{summary("low"), summary("ranked-004"), summary("ranked-001")}
	items[0].Likes = 1
	f.source.EXPECT().SearchWaifus(mock.Anything, "r", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: items}, nil).Times(3)

	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, "r"), strOpt(optSort, "rank_asc")))
	if cardName((*f.api.lastEdit().Embeds)[0]) != "Name ranked-001" {
		t.Errorf("rank sort first page = %q", (*f.api.lastEdit().Embeds)[0].Title)
	}
	minStars := &discordgo.ApplicationCommandInteractionDataOption{Name: optMinStars, Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, "r"), minStars))
	if got := editContent(f.api.lastEdit()); got != "<@alice>" {
		t.Errorf("min_stars should leave a single result: %q", got)
	}
	f.source.EXPECT().SearchWaifus(mock.Anything, "low", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: items[:1]}, nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, "low"), minStars))
	if got := editContent(f.api.lastEdit()); got != "<@alice> 1 results matched, but none passed your filters." {
		t.Errorf("filtered out = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, slugChoicePrefix+"ranked-004"), minStars))
	if got := editContent(f.api.lastEdit()); got != "<@alice> Name ranked-004 doesn't pass your filters (⭐⭐⭐⭐, rank #5, 10 likes, 1 trash)." {
		t.Errorf("picked suggestion below filter = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, slugChoicePrefix+"low"), minStars))
	if got := editContent(f.api.lastEdit()); got != "<@alice> Name low doesn't pass your filters (unranked, 10 likes, 1 trash)." {
		t.Errorf("picked unranked below filter = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, slugChoicePrefix+"ranked-001"), minStars))
	if cardName((*f.api.lastEdit().Embeds)[0]) != "Name ranked-001" {
		t.Error("picked suggestion that passes the filter should open the card")
	}
	ranked := &discordgo.ApplicationCommandInteractionDataOption{Name: optRanked, Type: discordgo.ApplicationCommandOptionBoolean, Value: true}
	minLikes := &discordgo.ApplicationCommandInteractionDataOption{Name: optMinLikes, Type: discordgo.ApplicationCommandOptionInteger, Value: float64(5)}
	maxTrash := &discordgo.ApplicationCommandInteractionDataOption{Name: optMaxTrash, Type: discordgo.ApplicationCommandOptionInteger, Value: float64(1)}
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, "r"), ranked, minLikes, maxTrash))
	if got := editContent(f.api.lastEdit()); got != "<@alice>\nPage 1 out of 2" {
		t.Errorf("ranked+likes+trash filters = %q", got)
	}

	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, slugChoicePrefix+"rem")))
	e := f.api.lastEdit()
	if (*e.Embeds)[0].Title != "Name rem" || hasComponents(*e.Components) {
		t.Errorf("direct slug should open one card: %+v", e)
	}
	f.source.ExpectedCalls = nil
	f.source.EXPECT().Get(mock.Anything, "ghost").Return(domain.Waifu{}, app.ErrNotFound).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, slugChoicePrefix+"ghost")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgWaifuNotFound {
		t.Errorf("missing direct slug = %q", got)
	}
	f.source.EXPECT().Get(mock.Anything, "boom").Return(domain.Waifu{}, errors.New("boom")).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optQuery, slugChoicePrefix+"boom")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgUnexpected {
		t.Errorf("direct slug error = %q", got)
	}
}

func TestSelectWindow(t *testing.T) {
	cases := []struct{ page, total, start, end int }{
		{0, 10, 0, 10},
		{0, 100, 0, 25},
		{12, 100, 0, 25},
		{13, 100, 1, 26},
		{50, 100, 38, 63},
		{99, 100, 75, 100},
	}
	for _, c := range cases {
		if s, e := selectWindow(c.page, c.total, 25); s != c.start || e != c.end {
			t.Errorf("selectWindow(%d,%d) = %d..%d, want %d..%d", c.page, c.total, s, e, c.start, c.end)
		}
	}
	f := newFixture(t)
	slugs := make([]string, 60)
	for i := range slugs {
		slugs[i] = fmt.Sprintf("w%02d", i)
	}
	f.give(aliceID, slugs...)
	ic := f.slash(aliceID, commandWaifu, subOwned, nil)
	f.run(ic)
	msg := f.message("msg-" + ic.ID)
	f.run(f.click(aliceID, msg, pagerPrefix+pagerLast))
	menu := f.api.lastRespond().Data.Components[1].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if len(menu.Options) != 25 || menu.Options[0].Label != "36. Name w35" || !menu.Options[24].Default {
		t.Errorf("window on last page = first %q default-last %v", menu.Options[0].Label, menu.Options[24].Default)
	}
}

func TestSearch_Autocomplete(t *testing.T) {
	f := newFixture(t)
	focused := strOpt(optQuery, "ranked-00")
	focused.Focused = true
	ic := f.slash(aliceID, commandWaifu, subSearch, nil, focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	r := f.api.lastRespond()
	if r.Type != discordgo.InteractionApplicationCommandAutocompleteResult || len(r.Data.Choices) != 10 {
		t.Fatalf("choices = %+v", r)
	}
	if r.Data.Choices[0].Value != slugChoicePrefix+"ranked-000" || !strings.HasPrefix(r.Data.Choices[0].Name, "Ranked 000 · ⭐") {
		t.Errorf("first choice = %+v", r.Data.Choices[0])
	}
	f.ranking.Set(nil)
	f.run(ic)
	if len(f.api.lastRespond().Data.Choices) != 0 {
		t.Error("no ranking means no suggestions")
	}
	searchOpts := f.bot.Commands()[0].Options[4].Options
	if len(searchOpts) != 7 || !searchOpts[0].Required || searchOpts[0].Name != optQuery {
		t.Errorf("search should expose seven options with a required query: %+v", searchOpts)
	}
	listOpts := f.bot.Commands()[0].Options[5].Options
	if len(listOpts) != 6 || listOpts[0].Name != optSeries {
		t.Errorf("list should expose the six non-query options: %+v", listOpts)
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

func TestSearch_SeriesOption(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().SearchWorks(mock.Anything, "nothing").Return(nil, nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optSeries, "nothing")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgSeriesNotFound {
		t.Errorf("not found = %q", got)
	}

	series := domain.Series{Slug: "re-zero", Name: "Re:Zero", URL: "https://www.mywaifulist.moe/series/re-zero", Description: "d"}
	f.source.EXPECT().SearchWorks(mock.Anything, "re zero").Return([]domain.Series{series}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("ram"), summary("rem")}}, nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optSeries, "re zero")))
	e := f.api.lastEdit()
	if editContent(e) != "<@alice> Showing results for series Re:Zero\nPage 1 out of 2" {
		t.Errorf("results = %q", editContent(e))
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "empty").Return([]domain.Series{series}, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1}, nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optSeries, "empty")))
	e = f.api.lastEdit()
	if editContent(e) != "<@alice> "+msgNoData || (*e.Embeds)[0].Title != "Re:Zero" {
		t.Errorf("no characters = %q", editContent(e))
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optSeries, "boom")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgUnexpected {
		t.Errorf("error = %q", got)
	}

	f.source.EXPECT().Work(mock.Anything, "re-zero").Return(series, nil).Times(3)
	f.source.EXPECT().WorkCharacters(mock.Anything, "re-zero", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("ram"), summary("rem")}}, nil).Times(3)
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optSeries, slugChoicePrefix+"re-zero"), strOpt(optSort, "name_desc")))
	e = f.api.lastEdit()
	if editContent(e) != "<@alice> Showing results for series Re:Zero\nPage 1 out of 2" || (*e.Embeds)[0].Title != "Name rem" {
		t.Errorf("direct series with sort = %q %q", editContent(e), (*e.Embeds)[0].Title)
	}
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optSeries, slugChoicePrefix+"re-zero"), strOpt(optQuery, "RAM")))
	e = f.api.lastEdit()
	if editContent(e) != "<@alice> Showing results for series Re:Zero" || (*e.Embeds)[0].Title != "Name ram" {
		t.Errorf("series plus query narrows by name = %q %q", editContent(e), (*e.Embeds)[0].Title)
	}
	minLikes := &discordgo.ApplicationCommandInteractionDataOption{Name: optMinLikes, Type: discordgo.ApplicationCommandOptionInteger, Value: float64(999)}
	f.run(f.slash(aliceID, commandWaifu, subSearch, nil, strOpt(optSeries, slugChoicePrefix+"re-zero"), minLikes))
	if got := editContent(f.api.lastEdit()); got != "<@alice> Showing results for series Re:Zero: 2 results matched, but none passed your filters." {
		t.Errorf("series filtered out = %q", got)
	}
}

func TestSearch_SeriesAutocomplete(t *testing.T) {
	f := newFixture(t)
	focused := strOpt(optSeries, "re ze")
	focused.Focused = true
	f.source.EXPECT().SearchWorks(mock.Anything, "re ze").Return([]domain.Series{{Slug: "re-zero", Name: "Re:Zero"}, {Slug: "", Name: ""}}, nil).Once()
	ic := f.slash(aliceID, commandWaifu, subSearch, nil, focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	r := f.api.lastRespond()
	if r.Type != discordgo.InteractionApplicationCommandAutocompleteResult || len(r.Data.Choices) != 1 || r.Data.Choices[0].Value != slugChoicePrefix+"re-zero" || r.Data.Choices[0].Name != "Re:Zero" {
		t.Errorf("choices = %+v", r.Data)
	}

	short := strOpt(optSeries, "r")
	short.Focused = true
	ic = f.slash(aliceID, commandWaifu, subSearch, nil, short)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if len(f.api.lastRespond().Data.Choices) != 0 {
		t.Error("one character should not query the API")
	}

	f.source.EXPECT().SearchWorks(mock.Anything, "boom").Return(nil, errors.New("boom")).Once()
	failing := strOpt(optSeries, "boom")
	failing.Focused = true
	ic = f.slash(aliceID, commandWaifu, subSearch, nil, failing)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if len(f.api.lastRespond().Data.Choices) != 0 {
		t.Error("lookup failure should yield no choices")
	}
}

func TestRouter_IgnoresUnknown(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, "nope", "", nil))
	f.run(f.slash(aliceID, commandWaifu, "nope", nil))
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

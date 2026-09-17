package discord

import (
	"errors"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func TestHelp_AllCommands(t *testing.T) {
	f := newFixture(t)
	check := func(name string, ic *discordgo.InteractionCreate, title string) {
		t.Helper()
		f.api.reset()
		f.run(ic)
		r := f.api.lastRespond()
		if r == nil || r.Type != discordgo.InteractionResponseChannelMessageWithSource || r.Data == nil || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 || len(r.Data.Embeds) != 1 {
			t.Fatalf("%s help = %+v", name, r)
		}
		e := r.Data.Embeds[0]
		if e.Title != title || len(e.Fields) == 0 {
			t.Errorf("%s help embed = %+v", name, e)
		}
		total := len(e.Title) + len(e.Description)
		for _, fld := range e.Fields {
			if len(fld.Value) > limitFieldValue || len(fld.Name) > limitFieldName || fld.Value == "" {
				t.Errorf("%s help field %q is %d characters", name, fld.Name, len(fld.Value))
			}
			total += len(fld.Name) + len(fld.Value)
		}
		if total > limitEmbedTotal || oversized("", []*discordgo.MessageEmbed{e}) {
			t.Errorf("%s help embed totals %d characters", name, total)
		}
	}
	check("waifu", f.slash(aliceID, commandWaifu, subHelp, nil), "/waifu · how it works")
	check("w", f.dm(aliceID, commandWAlias, subHelp), "/waifu · how it works")
	check("series", f.slash(aliceID, commandSeries, subHelp, nil), "/series · how it works")
	check("s", f.dm(aliceID, commandSAlias, subHelp), "/series · how it works")
	admin := f.slash(aliceID, commandAdmin, subHelp, nil)
	admin.Member.Permissions = discordgo.PermissionAdministrator
	check("admin", admin, "/admin · how it works")

	plain := f.slash(bobID, commandAdmin, subHelp, nil)
	f.run(plain)
	if got := f.respondContent(); got != msgAdminOnly {
		t.Errorf("admin help for non-admins = %q", got)
	}
	if got := waifuHelpEmbed().Fields[0].Name; !strings.Contains(got, "200") {
		t.Errorf("roll cost missing from help: %q", got)
	}
}

func TestAdmin_BannerReroll(t *testing.T) {
	f := newFixture(t)
	f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	f.script(0, 1)
	series := domain.Series{Slug: "fresh", Name: "Fresh", URL: "https://www.mywaifulist.moe/series/fresh", PictureURL: "https://img/fresh"}
	f.source.ExpectedCalls = nil
	current := detail("ranked-000")
	current.Appearances = []domain.Series{{Slug: "re-zero", Name: "Re:Zero"}}
	next := detail("ranked-001")
	next.Appearances = []domain.Series{series}
	f.source.EXPECT().Get(mock.Anything, "ranked-000").Return(current, nil).Once()
	f.source.EXPECT().Get(mock.Anything, "ranked-001").Return(next, nil).Once()
	f.source.EXPECT().WorkCharacters(mock.Anything, "fresh", 1).Return(app.SearchPage{Page: 1, LastPage: 1, Items: []domain.WaifuSummary{summary("ranked-001"), summary("ranked-010"), summary("ranked-011"), summary("ranked-012"), summary("ranked-013")}}, nil).Once()

	ic := f.adminCmd(aliceID, groupBanner, adminReroll, nil)
	f.run(ic)
	if r := f.api.calls[0].resp; r.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource || r.Data == nil || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Errorf("reroll should be deferred ephemeral, got %+v", r)
	}
	e := f.api.lastEdit()
	if got := editContent(e); got != "This week's banner is now **Fresh**." || (*e.Embeds)[0].Title != "Fresh" || len((*e.Embeds)[0].Fields) != 1 {
		t.Errorf("reroll = %q %+v", got, (*e.Embeds)[0])
	}
	f.run(f.slash(bobID, commandWaifu, subBanner, nil))
	if title := (*f.api.lastEdit().Embeds)[0].Title; title != "Fresh" {
		t.Errorf("/waifu banner after reroll shows %q", title)
	}

	f.run(f.adminCmd(aliceID, groupBanner, "nope", nil))
	if got := f.respondContent(); got != msgUnexpected {
		t.Errorf("unknown banner sub = %q", got)
	}

	f.ranking.Set(nil)
	f.run(f.adminCmd(aliceID, groupBanner, adminReroll, nil))
	if got := editContent(f.api.lastEdit()); got != msgAdminNoRanking {
		t.Errorf("no ranking = %q", got)
	}

	g := newFixture(t)
	g.source.ExpectedCalls = nil
	g.source.EXPECT().Get(mock.Anything, mock.Anything).Return(domain.Waifu{}, errors.New("down")).Maybe()
	for range 12 {
		g.script(0)
	}
	g.run(g.adminCmd(aliceID, groupBanner, adminReroll, nil))
	if got := editContent(g.api.lastEdit()); got != msgAdminNoSeries {
		t.Errorf("no eligible series = %q", got)
	}
}

func TestAdmin_Coins(t *testing.T) {
	f := newFixture(t)
	ic := f.adminCmd(aliceID, groupCoins, adminSet, resolvedUsers(bobID), userOption(bobID), intOpt(optAmount, 1000))
	f.run(ic)
	if r := f.api.calls[0].resp; r.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource || r.Data == nil || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Errorf("admin replies should be deferred ephemeral, got %+v", r)
	}
	if got := editContent(f.api.lastEdit()); got != "Set <@bob>'s balance to :coin: 1000 coins." || f.coins(bobID) != 1000 {
		t.Errorf("set = %q coins=%d", got, f.coins(bobID))
	}
	f.run(f.adminCmd(aliceID, groupCoins, adminIncrement, resolvedUsers(bobID), userOption(bobID), intOpt(optAmount, 250)))
	if got := editContent(f.api.lastEdit()); got != "Added :coin: 250 coins to <@bob>. New balance: 1250 coins." {
		t.Errorf("increment = %q", got)
	}
	f.run(f.adminCmd(aliceID, groupCoins, adminDecrement, resolvedUsers(bobID), userOption(bobID), intOpt(optAmount, 1250)))
	if got := editContent(f.api.lastEdit()); got != "Removed :coin: 1250 coins from <@bob>. New balance: 0 coins." {
		t.Errorf("decrement = %q", got)
	}
	f.run(f.adminCmd(aliceID, groupCoins, adminDecrement, resolvedUsers(bobID), userOption(bobID), intOpt(optAmount, 1)))
	if got := editContent(f.api.lastEdit()); got != "<@bob> doesn't have that many coins." || f.coins(bobID) != 0 {
		t.Errorf("decrement below zero = %q coins=%d", got, f.coins(bobID))
	}
	f.run(f.adminCmd(aliceID, groupCoins, adminSet, resolvedUsers(bobID), userOption(bobID), intOpt(optAmount, -5)))
	if got := editContent(f.api.lastEdit()); got != msgUnexpected {
		t.Errorf("negative set = %q", got)
	}
}

func TestAdmin_Waifus(t *testing.T) {
	f := newFixture(t)
	f.run(f.adminCmd(aliceID, groupWaifu, adminAdd, resolvedUsers(bobID), userOption(bobID), strOpt(optWaifu, slugChoicePrefix+"rem")))
	if got := editContent(f.api.lastEdit()); got != "Gave Name rem to <@bob>." || !f.owns(bobID, "rem") || f.coins(bobID) != domain.StartingCoins {
		t.Errorf("add = %q owns=%v coins=%d", got, f.owns(bobID, "rem"), f.coins(bobID))
	}
	f.run(f.adminCmd(aliceID, groupWaifu, adminAdd, resolvedUsers(bobID), userOption(bobID), strOpt(optWaifu, "rem")))
	if got := editContent(f.api.lastEdit()); got != "<@bob> already owns that waifu." {
		t.Errorf("duplicate add = %q", got)
	}
	f.source.ExpectedCalls = nil
	f.source.EXPECT().Get(mock.Anything, "ghost").Return(domain.Waifu{}, app.ErrNotFound).Once()
	f.run(f.adminCmd(aliceID, groupWaifu, adminAdd, resolvedUsers(bobID), userOption(bobID), strOpt(optWaifu, "ghost")))
	if got := editContent(f.api.lastEdit()); got != msgWaifuNotFound {
		t.Errorf("unknown waifu = %q", got)
	}
	f.source.EXPECT().Get(mock.Anything, "down").Return(domain.Waifu{}, errors.New("boom")).Once()
	f.run(f.adminCmd(aliceID, groupWaifu, adminAdd, resolvedUsers(bobID), userOption(bobID), strOpt(optWaifu, "down")))
	if got := editContent(f.api.lastEdit()); got != msgUnexpected {
		t.Errorf("source failure = %q", got)
	}

	f.run(f.adminCmd(aliceID, groupWaifu, adminRemove, resolvedUsers(bobID), userOption(bobID), strOpt(optWaifu, "rem")))
	if got := editContent(f.api.lastEdit()); got != "Took Name rem from <@bob>." || f.owns(bobID, "rem") || f.coins(bobID) != domain.StartingCoins {
		t.Errorf("remove = %q owns=%v coins=%d", got, f.owns(bobID, "rem"), f.coins(bobID))
	}
	f.run(f.adminCmd(aliceID, groupWaifu, adminRemove, resolvedUsers(bobID), userOption(bobID), strOpt(optWaifu, "rem")))
	if got := editContent(f.api.lastEdit()); got != "<@bob> doesn't own that waifu." {
		t.Errorf("remove again = %q", got)
	}
}

func TestAdmin_GuardsAndAutocomplete(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID)
	plain := f.adminCmd(bobID, groupCoins, adminSet, resolvedUsers(aliceID), userOption(aliceID), intOpt(optAmount, 1))
	plain.Member.Permissions = 0
	f.run(plain)
	if got := f.respondContent(); got != msgAdminOnly {
		t.Errorf("non-admin = %q", got)
	}
	if f.coins(aliceID) != domain.StartingCoins {
		t.Error("non-admin must not change coins")
	}

	dm := f.adminCmd(aliceID, groupCoins, adminSet, resolvedUsers(bobID), userOption(bobID), intOpt(optAmount, 1))
	dm.GuildID = ""
	f.run(dm)
	if got := f.respondContent(); got != "The admin command cannot be invoked from the direct messages of the bot." {
		t.Errorf("dm = %q", got)
	}

	f.run(f.adminCmd(aliceID, groupCoins, adminSet, nil, intOpt(optAmount, 1)))
	if got := f.respondContent(); got != msgUserNotFound {
		t.Errorf("missing user = %q", got)
	}
	f.run(f.adminCmd(aliceID, "nope", "what", resolvedUsers(bobID), userOption(bobID)))
	if got := editContent(f.api.lastEdit()); got != msgUnexpected {
		t.Errorf("unknown subcommand = %q", got)
	}
	bare := f.slash(aliceID, commandAdmin, "", nil)
	bare.Member.Permissions = discordgo.PermissionAdministrator
	f.run(bare)

	f.give(bobID, "rem", "ram")
	focused := strOpt(optWaifu, "ranked-00")
	focused.Focused = true
	ic := f.adminCmd(aliceID, groupWaifu, adminAdd, resolvedUsers(bobID), userOption(bobID), focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if r := f.api.lastRespond(); r.Type != discordgo.InteractionApplicationCommandAutocompleteResult || len(r.Data.Choices) != 10 || r.Data.Choices[0].Value != slugChoicePrefix+"ranked-000" {
		t.Errorf("add autocomplete = %+v", r)
	}
	focused = strOpt(optWaifu, "Name r")
	focused.Focused = true
	ic = f.adminCmd(aliceID, groupWaifu, adminRemove, resolvedUsers(bobID), userOption(bobID), focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if r := f.api.lastRespond(); len(r.Data.Choices) != 2 || r.Data.Choices[0].Value != "rem" {
		t.Errorf("remove autocomplete = %+v", r)
	}
	ic = f.adminCmd(aliceID, groupWaifu, adminRemove, nil, focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if r := f.api.lastRespond(); len(r.Data.Choices) != 0 {
		t.Errorf("remove autocomplete without a user = %+v", r)
	}
	ic = f.adminCmd(aliceID, groupCoins, adminSet, resolvedUsers(bobID), userOption(bobID), focused)
	ic.Type = discordgo.InteractionApplicationCommandAutocomplete
	f.run(ic)
	if r := f.api.lastRespond(); len(r.Data.Choices) != 0 {
		t.Errorf("coins autocomplete = %+v", r)
	}
}

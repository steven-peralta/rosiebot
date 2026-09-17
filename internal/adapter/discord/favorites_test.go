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

func (f *fixture) seriesMessage(id, slug, name string) *discordgo.Message {
	msg := &discordgo.Message{ID: id, ChannelID: channelID, Author: &discordgo.User{ID: botID}, Embeds: []*discordgo.MessageEmbed{{URL: "https://mywaifulist.moe/series/" + slug, Title: name}}}
	f.api.mu.Lock()
	f.api.messages[id] = msg
	f.api.mu.Unlock()
	return msg
}

func TestFavorites_ButtonToggleAndLists(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem")
	card := f.waifuMessage("c1", "rem")

	f.run(f.click(aliceID, card, favPrefix+"waifu"))
	if got := f.respondContent(); got != "Added **Name rem** to your favorites." {
		t.Fatalf("add = %q", got)
	}
	f.run(f.click(aliceID, card, favPrefix+"waifu"))
	if got := f.respondContent(); got != "Removed **Name rem** from your favorites." {
		t.Errorf("remove = %q", got)
	}
	f.run(f.click(aliceID, card, favPrefix+"waifu"))
	f.clock.now = f.clock.now.Add(time.Second)
	f.run(f.click(aliceID, f.waifuMessage("c2", "ranked-000"), favPrefix+"waifu"))
	f.run(f.click(bobID, card, favPrefix+"waifu"))
	if got := f.respondContent(); got != "Added **Name rem** to your favorites." {
		t.Errorf("bob's own favorite = %q", got)
	}

	f.source.EXPECT().Work(mock.Anything, "re-zero").Return(reZero, nil).Once()
	series := f.seriesMessage("s1", "re-zero", "Re:Zero")
	f.run(f.click(aliceID, series, favPrefix+"series"))
	if got := f.respondContent(); got != "Added **Re:Zero** to your favorites." {
		t.Errorf("series add = %q", got)
	}
	f.clock.now = f.clock.now.Add(time.Second)
	f.source.EXPECT().Work(mock.Anything, "konosuba").Return(domain.Series{}, fmt.Errorf("down")).Once()
	f.run(f.click(aliceID, f.seriesMessage("s2", "konosuba", "KonoSuba"), favPrefix+"series"))
	if got := f.respondContent(); got != "Added **KonoSuba** to your favorites." {
		t.Errorf("series add without detail should use the embed title: %q", got)
	}

	f.run(f.slash(aliceID, commandFavs, subFavWaifus, nil))
	e := f.api.lastEdit()
	if got := editContent(e); got != "<@alice> Your favorite waifus · 2" {
		t.Errorf("favs waifus = %q", got)
	}
	lines := strings.Split((*e.Embeds)[0].Description, "\n")
	if (*e.Embeds)[0].Title != "Favorite waifus" || len(lines) != 2 || lines[0] != "1. ☆☆☆☆☆ Name rem · unranked" || lines[1] != "2. ★★★★★ Name ranked-000 · Rank #1" {
		t.Errorf("favorite list = %+v", (*e.Embeds)[0])
	}
	if hasComponents(*e.Components) {
		t.Error("a one-page compact list has no rows")
	}

	f.run(f.slash(aliceID, commandFavs, subFavWaifus, nil, strOpt(optView, viewCards)))
	e = f.api.lastEdit()
	if !strings.HasSuffix(editContent(e), "Page 1 out of 2") || !strings.HasSuffix((*e.Embeds)[0].Title, "Name rem") {
		t.Errorf("cards view = %q %q", editContent(e), (*e.Embeds)[0].Title)
	}
	rows := *e.Components
	last := rows[len(rows)-1].(discordgo.ActionsRow).Components
	if len(last) != 1 || last[0].(discordgo.Button).CustomID != favPrefix+"waifu" {
		t.Errorf("card view actions = %+v", last)
	}

	f.run(f.slash(aliceID, commandFavs, subFavSeries, nil))
	e = f.api.lastEdit()
	if editContent(e) != "<@alice> Your favorite series · 2" || (*e.Embeds)[0].Title != "Favorite series" || (*e.Embeds)[0].Description != "1. Re:Zero\n2. KonoSuba" {
		t.Errorf("favs series = %q %+v", editContent(e), (*e.Embeds)[0])
	}

	f.run(f.slash(aliceID, commandFavs, subFavWaifus, resolvedUsers(bobID), userOption(bobID)))
	e = f.api.lastEdit()
	if editContent(e) != "<@alice> <@bob>'s favorite waifus · 1" || !strings.Contains((*e.Embeds)[0].Description, "Name rem") {
		t.Errorf("bob's favorites = %q", editContent(e))
	}
	f.run(f.slash(aliceID, commandFavs, subFavSeries, resolvedUsers(bobID), userOption(bobID)))
	if got := editContent(f.api.lastEdit()); got != "<@alice> <@bob>'s favorites don't have any series yet." {
		t.Errorf("bob's empty series = %q", got)
	}
	f.run(f.slash("carol", commandFavs, subFavWaifus, nil))
	if got := editContent(f.api.lastEdit()); got != "<@carol> Your favorites don't have any waifus yet." {
		t.Errorf("empty = %q", got)
	}
}

func TestFavorites_Guards(t *testing.T) {
	f := newFixture(t)
	card := f.waifuMessage("c1", "rem")

	dm := f.click(aliceID, card, favPrefix+"waifu")
	dm.GuildID = ""
	f.run(dm)
	if got := f.respondContent(); got != "The favs command cannot be invoked from the direct messages of the bot." {
		t.Errorf("dm = %q", got)
	}
	f.run(f.dm(aliceID, commandFavs, subFavWaifus))
	if got := f.respondContent(); got != "The favs command cannot be invoked from the direct messages of the bot." {
		t.Errorf("dm command = %q", got)
	}

	f.run(f.click(aliceID, &discordgo.Message{ID: "x", Author: &discordgo.User{ID: "someone"}}, favPrefix+"waifu"))
	if got := f.respondContent(); got != msgNotAWaifu {
		t.Errorf("foreign message = %q", got)
	}
	f.run(f.click(aliceID, &discordgo.Message{ID: "y", Author: &discordgo.User{ID: botID}, Content: "plain"}, favPrefix+"waifu"))
	if got := f.respondContent(); got != msgNotAWaifu {
		t.Errorf("no embed = %q", got)
	}
	f.run(f.click(aliceID, card, favPrefix+"series"))
	if got := f.respondContent(); got != msgNotAWaifu {
		t.Errorf("series button on a waifu card = %q", got)
	}
	f.api.reset()
	f.run(f.click(aliceID, card, favPrefix+"studio"))
	if len(f.api.calls) != 0 {
		t.Error("unknown favorite kind should be ignored")
	}
	f.run(f.slash(aliceID, commandFavs, "nope", nil))

	if slug, ok := SeriesSlugFromEmbeds([]*discordgo.MessageEmbed{nil, {URL: "https://example.com/series/x"}, {URL: "https://mywaifulist.moe/series/re-zero/extra"}, {URL: "https://www.mywaifulist.moe/series/re-zero"}}); !ok || slug != "re-zero" {
		t.Errorf("SeriesSlugFromEmbeds = %q %v", slug, ok)
	}
	if name := embedTitleName([]*discordgo.MessageEmbed{nil, {Title: ":star::star:\n🔞 Rem"}}); name != "Rem" {
		t.Errorf("embedTitleName = %q", name)
	}
}

func TestFavorites_RolledAndRandomCardsCarryTheButton(t *testing.T) {
	f := newFixture(t)
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRandom, nil))
	rows := *f.api.lastEdit().Components
	if len(rows) != 1 || rows[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID != favPrefix+"waifu" {
		t.Errorf("random card actions = %+v", rows)
	}
	f.script(4)
	f.run(f.slash(aliceID, commandWaifu, subToday, nil))
	rows = *f.api.lastEdit().Components
	if len(rows) != 1 || rows[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID != favPrefix+"waifu" {
		t.Errorf("today card actions = %+v", rows)
	}
}

func TestFavorites_AlertsOnRoll(t *testing.T) {
	f := newFixture(t)
	f.run(f.click(aliceID, f.waifuMessage("c1", "rem"), favPrefix+"waifu"))
	f.run(f.click("carol", f.waifuMessage("c2", "rem"), favPrefix+"waifu"))
	f.run(f.click(aliceID, f.waifuMessage("c4", "ram"), favPrefix+"waifu"))
	f.run(f.click("carol", f.waifuMessage("c5", "ram"), favPrefix+"waifu"))
	f.fund(bobID, 200)

	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(bobID, commandWaifu, subRoll, nil))
	f.bot.WaitBackground()
	got := f.api.dmsTo(aliceID)
	if len(got) != 1 || got[0] != "<@bob> just rolled **Name rem**, one of your favorites in Test Guild. Ask them for a trade with `/waifu trade`.\n\nTurn these off with /favs alerts off." {
		t.Errorf("alice's DM = %q", got)
	}
	if len(f.api.dmsTo("carol")) != 1 || len(f.api.dmsTo(bobID)) != 0 {
		t.Errorf("carol should be told, bob should not: carol=%d bob=%d", len(f.api.dmsTo("carol")), len(f.api.dmsTo(bobID)))
	}

	f.run(f.slash(aliceID, commandFavs, subFavAlerts, nil, strOpt(optState, "off")))
	if got := f.respondContent(); got != msgAlertsOff {
		t.Errorf("alerts off = %q", got)
	}
	f.api.closedDMs["carol"] = true
	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("ram"), nil).Once()
	f.run(f.slash(bobID, commandWaifu, subRoll, nil))
	f.bot.WaitBackground()
	if len(f.api.dmsTo(aliceID)) != 1 {
		t.Error("alice turned alerts off and must not get another DM")
	}
	if st, _ := f.alerts.Setting(t.Context(), domain.PlayerKey{GuildID: guildID, UserID: "carol"}); !st.DMClosed {
		t.Error("carol's bounced DM should mark her DMs closed")
	}

	f.run(f.slash(aliceID, commandFavs, subFavAlerts, nil, strOpt(optState, "on")))
	if got := f.respondContent(); got != msgAlertsOn {
		t.Errorf("alerts on = %q", got)
	}
	f.run(f.slash(aliceID, commandFavs, subFavAlerts, nil, strOpt(optState, "maybe")))
	if got := f.respondContent(); got != msgUnexpected {
		t.Errorf("bad state = %q", got)
	}
	f.run(f.dm(aliceID, commandFavs, subFavAlerts, strOpt(optState, "on")))
	if got := f.respondContent(); got != "The favs command cannot be invoked from the direct messages of the bot." {
		t.Errorf("alerts in DM = %q", got)
	}

	f.seedBanner("ranked-000", "ranked-001", "ranked-002", "ranked-003", "ranked-004")
	f.run(f.click(aliceID, f.waifuMessage("c3", "ranked-000"), favPrefix+"waifu"))
	f.bot.svc.Notify.BannerPicked(t.Context(), mustBanner(t, f))
	if got := f.api.dmsTo(aliceID); len(got) != 2 || !strings.Contains(got[1], "This week's banner is **Re:Zero** and it features your favorites: Name ranked-000.") {
		t.Errorf("banner DM = %q", got)
	}
	if err := f.bot.DirectMessage(t.Context(), "carol", "hi"); !errors.Is(err, app.ErrDMClosed) {
		t.Errorf("closed DM error = %v", err)
	}
}

func mustBanner(t *testing.T, f *fixture) domain.Banner {
	t.Helper()
	res, err := f.bot.svc.Banner.Current(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return res.Banner
}

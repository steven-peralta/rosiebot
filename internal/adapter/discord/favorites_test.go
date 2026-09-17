package discord

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"

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

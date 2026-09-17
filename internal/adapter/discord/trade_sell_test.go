package discord

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

func (f *fixture) tradeCmd(sender string, target string, give, receive string) *discordgo.InteractionCreate {
	opts := []*discordgo.ApplicationCommandInteractionDataOption{userOption(target)}
	if give != "" {
		opts = append(opts, strOpt(optGive, give))
	}
	if receive != "" {
		opts = append(opts, strOpt(optReceive, receive))
	}
	return f.slash(sender, commandWaifu, subTrade, resolvedUsers(target), opts...)
}

func TestTrade_ViolationTexts(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem", "shared")
	f.give(bobID, "ram", "shared")
	cases := map[string]struct {
		give, receive, want string
	}{
		"sender lacks":   {give: "ghost", want: "<@alice>, you don't own ghost."},
		"target has":     {give: "shared", want: "<@alice>, <@bob> already owns shared."},
		"target lacks":   {receive: "ghost", want: "<@alice>, <@bob> doesn't own ghost."},
		"sender has":     {receive: "shared", want: "<@alice>, you already own shared."},
		"overlap":        {give: "rem", receive: "rem", want: "<@alice> A waifu can't be on both sides of a trade."},
		"dedupe + valid": {give: "rem, rem", receive: "ram", want: "<@bob>: <@alice> is offering the following trade request:"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f.run(f.tradeCmd(aliceID, bobID, c.give, c.receive))
			if got := editContent(f.api.lastEdit()); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
	f.run(f.slash(aliceID, commandWaifu, subTrade, resolvedUsers(aliceID), userOption(aliceID), strOpt(optGive, "rem")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> You can't trade with yourself." {
		t.Errorf("self trade = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subTrade, nil, strOpt(optGive, "rem")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgUserNotFound {
		t.Errorf("missing user = %q", got)
	}
	botUser := resolvedUsers("robot")
	botUser.Users["robot"].Bot = true
	f.run(f.slash(aliceID, commandWaifu, subTrade, botUser, userOption("robot"), strOpt(optGive, "rem")))
	if got := editContent(f.api.lastEdit()); got != "<@alice> You can't trade with a bot." {
		t.Errorf("bot target = %q", got)
	}
}

func TestTrade_AcceptDeclineFlow(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem")
	f.give(bobID, "ram")

	ic := f.tradeCmd(aliceID, bobID, "rem", "ram")
	f.run(ic)
	e := f.api.lastEdit()
	embed := (*e.Embeds)[0]
	if embed.Title != "Trade Request" || embed.Fields[0].Value != "• Name rem" || embed.Fields[1].Value != "• Name ram" || !hasComponents(*e.Components) {
		t.Errorf("offer embed = %+v", embed)
	}
	msg := f.message("msg-" + ic.ID)

	f.run(f.click("carol", msg, tradePrefix+tradeAccept))
	if got := f.respondContent(); got != msgNotYourMenu {
		t.Errorf("stranger accept = %q", got)
	}
	f.run(f.click(aliceID, msg, tradePrefix+tradeAccept))
	if got := f.respondContent(); got != msgNotYourMenu {
		t.Errorf("sender accept = %q", got)
	}
	f.run(f.click(bobID, msg, tradePrefix+"bogus"))

	f.run(f.click(bobID, msg, tradePrefix+tradeAccept))
	r := f.api.lastRespond()
	if r.Type != discordgo.InteractionResponseUpdateMessage || r.Data.Content != "<@alice> <@bob> "+msgTradeAccepted || len(r.Data.Components) != 0 {
		t.Errorf("accept = %+v", r.Data)
	}
	if !f.owns(bobID, "rem") || !f.owns(aliceID, "ram") {
		t.Error("trade not applied")
	}
	f.run(f.click(bobID, msg, tradePrefix+tradeAccept))
	if got := f.respondContent(); got != msgTradeExpired {
		t.Errorf("second accept = %q", got)
	}

	f.give(aliceID, "gift")
	ic = f.tradeCmd(aliceID, bobID, "gift", "")
	f.run(ic)
	if got := (*f.api.lastEdit().Embeds)[0].Fields[1].Value; got != "*nothing*" {
		t.Errorf("gift wants = %q", got)
	}
	msg = f.message("msg-" + ic.ID)
	f.run(f.click(aliceID, msg, tradePrefix+tradeDecline))
	if got := f.api.lastRespond().Data.Content; got != "<@alice> <@bob> "+msgTradeDenied {
		t.Errorf("sender decline = %q", got)
	}
	if !f.owns(aliceID, "gift") {
		t.Error("declined gift must stay")
	}
}

func TestTrade_ConflictAndExpiry(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem")
	f.give(bobID, "ram")
	ic := f.tradeCmd(aliceID, bobID, "rem", "ram")
	f.run(ic)
	msg := f.message("msg-" + ic.ID)

	key := domain.PlayerKey{GuildID: guildID, UserID: bobID}
	if _, ok, _ := f.players.SellOwned(t.Context(), key, "ram", domain.SellPrice); !ok {
		t.Fatal("setup sale")
	}
	f.run(f.click(bobID, msg, tradePrefix+tradeAccept))
	if got := f.api.lastRespond().Data.Content; got != "<@alice> <@bob> "+msgTradeConflict+" <@bob> doesn't own ram." {
		t.Errorf("conflict = %q", got)
	}
	if f.bot.Sessions().Len() != 0 {
		t.Error("conflicting trade should be closed")
	}

	ic = f.tradeCmd(aliceID, bobID, "rem", "")
	f.run(ic)
	msg = f.message("msg-" + ic.ID)
	f.clock.now = f.clock.now.Add(time.Hour)
	if n := f.bot.SweepExpired(); n != 1 {
		t.Errorf("swept %d", n)
	}
	if me := f.api.lastMsgEdit(); me == nil || *me.Content != msgTradeExpired {
		t.Errorf("expired edit = %+v", me)
	}
	f.run(f.click(bobID, msg, tradePrefix+tradeAccept))
	if got := f.respondContent(); got != msgTradeExpired {
		t.Errorf("click after expiry = %q", got)
	}
	f.run(f.click(bobID, nil, tradePrefix+tradeAccept))
}

func TestTrade_Autocomplete(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem", "ram", "emilia")
	f.give(bobID, "beatrice")

	choices := func() []*discordgo.ApplicationCommandOptionChoice {
		r := f.api.lastRespond()
		if r.Type != discordgo.InteractionApplicationCommandAutocompleteResult {
			t.Fatalf("not an autocomplete response: %+v", r)
		}
		return r.Data.Choices
	}

	focused := strOpt(optGive, "Name r")
	focused.Focused = true
	f.run(f.autocomplete(aliceID, resolvedUsers(bobID), userOption(bobID), focused))
	got := choices()
	if len(got) != 2 || got[0].Value != "rem" || got[1].Value != "ram" || got[0].Name != "Name rem" {
		t.Errorf("give choices = %+v", got)
	}

	focused = strOpt(optGive, "rem, ram, name e")
	focused.Focused = true
	f.run(f.autocomplete(aliceID, resolvedUsers(bobID), userOption(bobID), focused))
	got = choices()
	if len(got) != 1 || got[0].Value != "rem,ram,emilia" {
		t.Errorf("multi choices = %+v", got)
	}

	focused = strOpt(optGive, "rem,")
	focused.Focused = true
	f.run(f.autocomplete(aliceID, resolvedUsers(bobID), userOption(bobID), focused))
	if got = choices(); len(got) != 3 {
		t.Errorf("trailing comma should list everything: %+v", got)
	}

	focused = strOpt(optReceive, "")
	focused.Focused = true
	f.run(f.autocomplete(aliceID, resolvedUsers(bobID), userOption(bobID), focused))
	got = choices()
	if len(got) != 1 || got[0].Value != "beatrice" {
		t.Errorf("receive choices = %+v", got)
	}

	f.run(f.autocomplete(aliceID, nil, focused))
	if got = choices(); len(got) != 0 {
		t.Errorf("receive without a target should be empty: %+v", got)
	}

	plain := strOpt(optGive, "x")
	f.run(f.autocomplete(aliceID, nil, plain))
	if got = choices(); len(got) != 0 {
		t.Errorf("no focused option should be empty: %+v", got)
	}

	dm := f.autocomplete(aliceID, resolvedUsers(bobID), userOption(bobID), focused)
	dm.GuildID = ""
	f.run(dm)
	if got = choices(); len(got) != 0 {
		t.Errorf("DM autocomplete should be empty: %+v", got)
	}

	long := strings.Repeat("x", 95)
	f.give("dave", long)
	focused = strOpt(optGive, "abcdef,")
	focused.Focused = true
	f.run(f.autocomplete("dave", resolvedUsers(bobID), userOption(bobID), focused))
	if got = choices(); len(got) != 0 {
		t.Errorf("over-long values must be dropped: %+v", got)
	}
}

func (f *fixture) waifuMessage(id, slug string) *discordgo.Message {
	msg := &discordgo.Message{ID: id, ChannelID: channelID, Author: &discordgo.User{ID: botID}, Embeds: []*discordgo.MessageEmbed{{URL: waifuURL(slug), Title: "Name " + slug}}}
	f.api.mu.Lock()
	f.api.messages[id] = msg
	f.api.mu.Unlock()
	return msg
}

func TestSell_RankedPrice(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "ranked-000", "ranked-100", "plain")
	cases := []struct {
		slug  string
		price int64
	}{{"ranked-000", 1000}, {"ranked-100", 150}, {"plain", 100}}
	balance := int64(domain.StartingCoins)
	for i, c := range cases {
		msg := f.waifuMessage(fmt.Sprintf("s%d", i), c.slug)
		f.run(f.contextMenu(aliceID, msg))
		r := f.api.lastRespond()
		if want := fmt.Sprintf("Are you sure you want to sell your Name %s for %s coins?", c.slug, thousands(int(c.price))); r.Data.Content != want {
			t.Errorf("confirm = %q, want %q", r.Data.Content, want)
		}
		okID := r.Data.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID
		f.run(f.click(aliceID, nil, okID))
		balance += c.price
		if want := fmt.Sprintf("Sold Name %s for %s coins. You now have %s coins.", c.slug, thousands(int(c.price)), thousands(int(balance))); f.api.lastRespond().Data.Content != want {
			t.Errorf("sold = %q, want %q", f.api.lastRespond().Data.Content, want)
		}
	}
	if f.coins(aliceID) != balance {
		t.Errorf("coins = %d, want %d", f.coins(aliceID), balance)
	}
}

func TestSellAll_Flow(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "ranked-000", "ranked-005", "ranked-020", "ranked-040", "ranked-100", "plain-a", "plain-b")

	ic := f.slash(aliceID, commandWaifu, subSellAll, nil, intOpt(optMaxStars, 2))
	f.run(ic)
	if r := f.api.calls[0].resp; r.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource || r.Data == nil || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatalf("sellall should defer an ephemeral reply, got %+v", r)
	}
	e := f.api.lastEdit()
	if editContent(e) != "Sell **4** waifus (2 unranked, 1 ⭐, 1 ⭐⭐) for :coin: 550 coins? This can't be undone." {
		t.Fatalf("confirm = %q", editContent(e))
	}
	embed := (*e.Embeds)[0]
	lines := strings.Split(embed.Description, "\n")
	if embed.Title != "Waifus to sell" || len(lines) != 4 || lines[0] != "1. ★★☆☆☆ Name ranked-040 · Rank #41" || lines[3] != "4. ☆☆☆☆☆ Name plain-b · unranked" {
		t.Errorf("confirm list = %+v", embed)
	}
	rows := *e.Components
	if len(rows) != 1 {
		t.Fatalf("a four-item confirmation should only have the action row, got %d rows", len(rows))
	}
	buttons := rows[0].(discordgo.ActionsRow).Components
	okID := buttons[0].(discordgo.Button).CustomID
	viewID := buttons[1].(discordgo.Button).CustomID
	if okID != sellPrefix+sellAll+":2" || viewID != sellPrefix+sellView+":2" || buttons[2].(discordgo.Button).CustomID != sellPrefix+sellNo {
		t.Errorf("buttons = %+v", buttons)
	}
	confirmMsg := f.message("msg-" + ic.ID)

	view := f.click(aliceID, confirmMsg, viewID)
	f.run(view)
	ve := f.api.lastEdit()
	if editContent(ve) != msgSellAllViewing+"\nPage 1 out of 4" || !strings.HasSuffix((*ve.Embeds)[0].Title, "Name ranked-040") || !hasComponents(*ve.Components) {
		t.Errorf("view characters = %q %q", editContent(ve), (*ve.Embeds)[0].Title)
	}
	for _, c := range (*ve.Components)[0].(discordgo.ActionsRow).Components {
		if c.(discordgo.Button).CustomID == sellPrefix+sellAsk {
			t.Error("the preview pager must not offer single sells")
		}
	}

	f.run(f.click(aliceID, confirmMsg, sellPrefix+sellNo))
	if got := f.api.lastRespond().Data.Content; got != msgSellCancelled {
		t.Errorf("cancel = %q", got)
	}
	if f.coins(aliceID) != domain.StartingCoins {
		t.Error("cancel must not sell")
	}

	f.run(f.click(aliceID, confirmMsg, okID))
	if got := f.api.lastRespond().Data.Content; got != "Sold 4 waifus for :coin: 550 coins. You now have 750 coins." {
		t.Errorf("sold = %q", got)
	}
	if f.owns(aliceID, "plain-a") || f.owns(aliceID, "ranked-040") || !f.owns(aliceID, "ranked-020") {
		t.Error("wrong waifus sold")
	}

	f.run(f.click(aliceID, nil, okID))
	if got := f.api.lastRespond().Data.Content; got != msgSellAllNone {
		t.Errorf("second confirm = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subSellAll, nil, intOpt(optMaxStars, 0)))
	if got := editContent(f.api.lastEdit()); got != msgSellAllNone {
		t.Errorf("nothing left unranked = %q", got)
	}
	f.run(f.click(aliceID, confirmMsg, viewID))
	if got := editContent(f.api.lastEdit()); got != msgSellAllNone {
		t.Errorf("view after selling = %q", got)
	}

	f.run(f.slash(aliceID, commandWaifu, subSellAll, nil, intOpt(optMaxStars, 5)))
	if got := editContent(f.api.lastEdit()); got != "Sell **3** waifus (1 ⭐⭐⭐, 1 ⭐⭐⭐⭐, 1 ⭐⭐⭐⭐⭐) for :coin: 1,800 coins? This can't be undone." {
		t.Errorf("everything = %q", got)
	}
	f.run(f.click(aliceID, confirmMsg, sellPrefix+sellView+":x"))
	if got := f.respondContent(); got != msgUnexpected {
		t.Errorf("bad view id = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subSellAll, nil, intOpt(optMaxStars, 9)))
	if got := f.respondContent(); got != msgUnexpected {
		t.Errorf("bad threshold = %q", got)
	}
	f.run(f.click(aliceID, nil, sellPrefix+sellAll+":x"))
	if got := f.api.lastRespond().Data.Content; got != msgUnexpected {
		t.Errorf("bad confirm id = %q", got)
	}
	if choices := sellAllChoices(); len(choices) != 6 || choices[0].Name != "Unranked only" || choices[5].Name != "Everything" {
		t.Errorf("choices = %+v", choices)
	}
}

func TestSellAll_PagedConfirmation(t *testing.T) {
	f := newFixture(t)
	slugs := make([]string, 0, 45)
	for i := range 45 {
		slugs = append(slugs, fmt.Sprintf("plain-%02d", i))
	}
	f.give(aliceID, slugs...)
	ic := f.slash(aliceID, commandWaifu, subSellAll, nil, intOpt(optMaxStars, 0))
	f.run(ic)
	e := f.api.lastEdit()
	if !strings.HasSuffix(editContent(e), "Page 1 out of 3") {
		t.Fatalf("content = %q", editContent(e))
	}
	rows := *e.Components
	if len(rows) != 3 {
		t.Fatalf("paged confirmation should have nav, select and action rows, got %d", len(rows))
	}
	if btn := rows[2].(discordgo.ActionsRow).Components[0].(discordgo.Button); btn.CustomID != sellPrefix+sellAll+":0" {
		t.Errorf("action row = %+v", rows[2])
	}
	msg := f.message("msg-" + ic.ID)
	f.run(f.click(aliceID, msg, pagerPrefix+pagerLast))
	r := f.api.lastRespond()
	if !strings.HasSuffix(r.Data.Content, "Page 3 out of 3") || len(strings.Split(r.Data.Embeds[0].Description, "\n")) != 5 || len(r.Data.Components) != 3 {
		t.Errorf("last page = %+v", r.Data)
	}
	f.run(f.click(aliceID, msg, sellPrefix+sellAll+":0"))
	if got := f.api.lastRespond().Data.Content; got != "Sold 45 waifus for :coin: 4,500 coins. You now have 4,700 coins." {
		t.Errorf("sold = %q", got)
	}
	if _, ok := f.bot.Sessions().Get(msg.ID); ok {
		t.Error("confirmation pager session should be closed after selling")
	}
}

func TestSell_ContextMenuFlow(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem")

	f.run(f.contextMenu(aliceID, &discordgo.Message{ID: "m0", Author: &discordgo.User{ID: "someone"}, Content: "hi"}))
	if got := f.respondContent(); got != msgNotAWaifu {
		t.Errorf("foreign message = %q", got)
	}
	f.run(f.contextMenu(aliceID, &discordgo.Message{ID: "m1", Author: &discordgo.User{ID: botID}, Content: "no embed"}))
	if got := f.respondContent(); got != msgNotAWaifu {
		t.Errorf("no embed = %q", got)
	}

	other := f.waifuMessage("m2", "ram")
	f.run(f.contextMenu(aliceID, other))
	if got := f.respondContent(); got != "You don't own ram." {
		t.Errorf("not owned = %q", got)
	}

	mine := f.waifuMessage("m3", "rem")
	f.run(f.contextMenu(aliceID, mine))
	r := f.api.lastRespond()
	if r.Data.Content != "Are you sure you want to sell your Name rem for 100 coins?" || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 || !hasComponents(r.Data.Components) {
		t.Fatalf("confirm = %+v", r.Data)
	}
	okID := r.Data.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID
	if okID != sellPrefix+"ok:"+channelID+":m3" {
		t.Errorf("confirm custom id = %q", okID)
	}

	f.run(f.click(aliceID, nil, sellPrefix+sellNo))
	if got := f.api.lastRespond().Data.Content; got != msgSellCancelled {
		t.Errorf("cancel = %q", got)
	}

	f.run(f.click(aliceID, nil, okID))
	if got := f.api.lastRespond().Data.Content; got != "Sold Name rem for 100 coins. You now have 300 coins." {
		t.Errorf("sold = %q", got)
	}
	if f.owns(aliceID, "rem") || f.coins(aliceID) != 300 {
		t.Error("sale not applied")
	}

	f.run(f.click(aliceID, nil, okID))
	if got := f.api.lastRespond().Data.Content; got != "You don't own rem." {
		t.Errorf("double sell = %q", got)
	}

	f.run(f.click(aliceID, nil, sellPrefix+"ok:"+channelID+":missing"))
	if got := f.api.lastRespond().Data.Content; got != msgNotAWaifu {
		t.Errorf("missing parent = %q", got)
	}
	f.run(f.click(aliceID, nil, sellPrefix+"weird"))

	dm := f.contextMenu(aliceID, mine)
	dm.GuildID = ""
	f.run(dm)
	if got := f.respondContent(); got != "The sell command cannot be invoked from the direct messages of the bot." {
		t.Errorf("dm = %q", got)
	}
}

func TestSell_RerendersLivePager(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "a", "b", "c")
	ic := f.slash(aliceID, commandWaifu, subOwned, nil)
	f.run(ic)
	msgID := "msg-" + ic.ID
	msg := f.message(msgID)
	f.run(f.click(aliceID, msg, pagerPrefix+pagerNext))

	f.run(f.click(aliceID, nil, sellPrefix+"ok:"+channelID+":"+msgID))
	if got := f.api.lastRespond().Data.Content; !strings.HasPrefix(got, "Sold Name b") {
		t.Errorf("sold the displayed page = %q", got)
	}
	me := f.api.lastMsgEdit()
	if me == nil || *me.Content != "<@alice>\nPage 2 out of 2" || (*me.Embeds)[0].Title != "Name c" {
		t.Errorf("re-render = %q %+v", *me.Content, (*me.Embeds)[0].Title)
	}

	f.run(f.click(aliceID, nil, sellPrefix+"ok:"+channelID+":"+msgID))
	me = f.api.lastMsgEdit()
	if me == nil || *me.Content != "<@alice>" || len(*me.Components) != 1 || (*me.Embeds)[0].Title != "Name a" {
		t.Errorf("single page should keep only the sell row: %+v", me)
	}

	f.run(f.click(aliceID, nil, sellPrefix+"ok:"+channelID+":"+msgID))
	me = f.api.lastMsgEdit()
	if me == nil || !strings.HasSuffix(*me.Content, msgOwnsNothing) || f.bot.Sessions().Len() != 0 {
		t.Errorf("selling the final waifu = %+v sessions=%d", me, f.bot.Sessions().Len())
	}
}

func TestSell_ButtonOnOwnedCard(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "solo")
	ic := f.slash(aliceID, commandWaifu, subOwned, nil)
	f.run(ic)
	e := f.api.lastEdit()
	if len(*e.Components) != 1 || (*e.Components)[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID != sellPrefix+sellAsk {
		t.Fatalf("single owned card should carry only the sell row: %+v", *e.Components)
	}
	msgID := "msg-" + ic.ID
	msg := f.message(msgID)

	f.run(f.click(bobID, msg, sellPrefix+sellAsk))
	if got := f.respondContent(); got != "You don't own solo." {
		t.Errorf("non-owner pressing sell = %q", got)
	}

	f.run(f.click(aliceID, msg, sellPrefix+sellAsk))
	r := f.api.lastRespond()
	if r.Data.Content != "Are you sure you want to sell your Name solo for 100 coins?" || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatalf("confirm = %+v", r.Data)
	}
	okID := r.Data.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID
	f.run(f.click(aliceID, nil, okID))
	if got := f.api.lastRespond().Data.Content; got != "Sold Name solo for 100 coins. You now have 300 coins." {
		t.Errorf("sold = %q", got)
	}
	me := f.api.lastMsgEdit()
	if me == nil || me.ID != msgID || hasSellButton(*me.Components) || !strings.HasSuffix(*me.Content, "(sold)") {
		t.Errorf("card without a session should lose its sell button: %+v", me)
	}

	f.give(bobID, "x")
	f.run(f.slash(aliceID, commandWaifu, subOwned, resolvedUsers(bobID), userOption(bobID)))
	if hasSellButton(*f.api.lastEdit().Components) {
		t.Error("viewing someone else's collection must not offer a sell button")
	}
}

func TestSell_RemoveFromPagerEdgeCases(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "a", "b")
	ic := f.slash(aliceID, commandWaifu, subOwned, nil)
	f.run(ic)
	msgID := "msg-" + ic.ID
	f.api.reset()
	f.bot.removeFromPager(t.Context(), channelID, msgID, "not-on-page")
	f.bot.removeFromPager(t.Context(), channelID, "unknown", "a")
	if len(f.api.calls) != 0 {
		t.Error("no edits expected")
	}
	f.api.mu.Lock()
	delete(f.api.messages, msgID)
	f.api.mu.Unlock()
	f.bot.removeFromPager(t.Context(), channelID, msgID, "a")
	if me := f.api.lastMsgEdit(); me == nil {
		t.Error("removal should still re-render")
	}
	f.run(f.slash(aliceID, commandWaifu, subOwned, nil))
	f.api.mu.Lock()
	f.api.failFetch = true
	f.api.mu.Unlock()
	f.run(f.click(aliceID, nil, sellPrefix+"ok:"+channelID+":"+msgID))
	if got := f.api.lastRespond().Data.Content; got != msgNotAWaifu {
		t.Errorf("fetch failure = %q", got)
	}
}

func TestSessions_SweepStripsComponents(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "a", "b")
	ic := f.slash(aliceID, commandWaifu, subOwned, nil)
	f.run(ic)
	msgID := "msg-" + ic.ID
	if f.bot.SweepExpired() != 0 {
		t.Error("nothing should expire yet")
	}
	f.clock.now = f.clock.now.Add(time.Hour)
	if n := f.bot.SweepExpired(); n != 1 {
		t.Errorf("swept %d", n)
	}
	me := f.api.lastMsgEdit()
	if me == nil || me.ID != msgID || len(*me.Components) != 0 || len(*me.Embeds) != 1 {
		t.Errorf("sweep edit = %+v", me)
	}

	f.run(f.slash(aliceID, commandWaifu, subOwned, nil))
	f.api.mu.Lock()
	f.api.failFetch = true
	f.api.mu.Unlock()
	f.clock.now = f.clock.now.Add(time.Hour)
	if n := f.bot.SweepExpired(); n != 1 {
		t.Errorf("swept %d", n)
	}
}

func TestSessionStore_Basics(t *testing.T) {
	clock := &fakeClock{now: time.Now()}
	st := NewSessionStore(clock)
	s := &Session{Kind: SessionPager, OwnerID: aliceID}
	st.Put("m", s, time.Minute)
	if got, ok := st.Get("m"); !ok || got != s || st.Len() != 1 {
		t.Fatal("put/get")
	}
	clock.now = clock.now.Add(59 * time.Second)
	if _, ok := st.Get("m"); !ok {
		t.Fatal("sliding TTL should keep a touched pager alive")
	}
	clock.now = clock.now.Add(59 * time.Second)
	if _, ok := st.Get("m"); !ok {
		t.Fatal("still alive after touch")
	}
	clock.now = clock.now.Add(2 * time.Minute)
	if _, ok := st.Get("m"); ok || st.Len() != 0 {
		t.Fatal("expired session should be gone")
	}

	tr := &Session{Kind: SessionTrade}
	st.Put("t", tr, time.Minute)
	clock.now = clock.now.Add(30 * time.Second)
	st.Get("t")
	clock.now = clock.now.Add(31 * time.Second)
	if _, ok := st.Get("t"); ok {
		t.Fatal("trade sessions must not slide")
	}
	st.Put("x", &Session{}, time.Minute)
	st.Delete("x")
	if st.Len() != 0 {
		t.Fatal("delete")
	}
	if !tr.beginAccept() || tr.beginAccept() {
		t.Fatal("beginAccept should succeed once")
	}
	tr.reopen()
	if !tr.beginAccept() {
		t.Fatal("reopen should allow accepting again")
	}
	tr.finish()
	if tr.beginAccept() {
		t.Fatal("finished trade cannot be accepted")
	}
}

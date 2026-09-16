package discord

import (
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
	if me == nil || me.ID != msgID || len(*me.Components) != 0 || !strings.HasSuffix(*me.Content, "(sold)") {
		t.Errorf("card without a session should lose its sell button: %+v", me)
	}

	f.give(bobID, "x")
	f.run(f.slash(aliceID, commandWaifu, subOwned, resolvedUsers(bobID), userOption(bobID)))
	if len(*f.api.lastEdit().Components) != 0 {
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

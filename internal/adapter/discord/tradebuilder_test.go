package discord

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

func (f *fixture) selectValues(userID string, msg *discordgo.Message, customID string, values ...string) *discordgo.InteractionCreate {
	ic := f.click(userID, msg, customID)
	ic.Data = discordgo.MessageComponentInteractionData{CustomID: customID, Values: values}
	return ic
}

func builderRows(t *testing.T, e *discordgo.WebhookEdit) (give, get *discordgo.SelectMenu, rows []discordgo.MessageComponent) {
	t.Helper()
	rows = *e.Components
	for _, row := range rows {
		for _, c := range row.(discordgo.ActionsRow).Components {
			if menu, ok := c.(discordgo.SelectMenu); ok {
				switch menu.CustomID {
				case builderPrefix + builderGive:
					m := menu
					give = &m
				case builderPrefix + builderGet:
					m := menu
					get = &m
				}
			}
		}
	}
	return give, get, rows
}

func TestTradeBuilder_Flow(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem", "ram")
	f.give(bobID, "emilia")

	ic := f.slash(aliceID, commandWaifu, subTrade, resolvedUsers(bobID), userOption(bobID))
	f.run(ic)
	if r := f.api.calls[0].resp; r.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource || r.Data == nil || r.Data.Flags&discordgo.MessageFlagsEphemeral == 0 {
		t.Fatalf("builder should be a deferred ephemeral response: %+v", r)
	}
	e := f.api.lastEdit()
	give, get, rows := builderRows(t, e)
	if editContent(e) != "Trade with <@bob>" || give == nil || get == nil || len(rows) != 4 {
		t.Fatalf("builder layout = %q give=%v get=%v rows=%d", editContent(e), give != nil, get != nil, len(rows))
	}
	if len(give.Options) != 2 || len(get.Options) != 1 || *give.MinValues != 0 || give.MaxValues != 2 {
		t.Errorf("menus = give %d (min %d max %d) get %d", len(give.Options), *give.MinValues, give.MaxValues, len(get.Options))
	}
	if btn := rows[3].(discordgo.ActionsRow).Components[0].(discordgo.Button); !btn.Disabled {
		t.Error("send should start disabled")
	}
	msgID := "msg-" + ic.ID
	msg := f.message(msgID)

	f.run(f.selectValues(bobID, msg, builderPrefix+builderGive, "rem"))
	if got := f.respondContent(); got != msgNotYourMenu {
		t.Errorf("stranger = %q", got)
	}

	f.run(f.selectValues(aliceID, msg, builderPrefix+builderGive, "rem"))
	r := f.api.lastRespond()
	if r.Type != discordgo.InteractionResponseUpdateMessage || !strings.Contains(r.Data.Embeds[0].Fields[0].Value, "Name rem") {
		t.Errorf("give selection = %+v", r.Data.Embeds[0].Fields[0])
	}
	f.run(f.selectValues(aliceID, msg, builderPrefix+builderGet, "emilia"))
	r = f.api.lastRespond()
	if !strings.Contains(r.Data.Embeds[0].Fields[1].Value, "Name emilia") {
		t.Errorf("get selection = %+v", r.Data.Embeds[0].Fields[1])
	}
	if btn := r.Data.Components[3].(discordgo.ActionsRow).Components[0].(discordgo.Button); btn.Disabled {
		t.Error("send should be enabled once something is picked")
	}

	f.run(f.selectValues(aliceID, msg, builderPrefix+builderGive))
	if strings.Contains(f.api.lastRespond().Data.Embeds[0].Fields[0].Value, "Name rem") {
		t.Error("clearing the menu should unselect")
	}
	f.run(f.selectValues(aliceID, msg, builderPrefix+builderGive, "ram"))

	send := f.click(aliceID, msg, builderPrefix+builderSend)
	f.run(send)
	if got := f.api.lastRespond().Data.Content; got != msgBuilderSent {
		t.Fatalf("send = %q", got)
	}
	fu := f.api.lastFollowup()
	if fu == nil || fu.Content != "<@bob>: <@alice> is offering the following trade request:" || len(fu.Components) != 1 {
		t.Fatalf("offer followup = %+v", fu)
	}
	if fu.Embeds[0].Fields[0].Value != "• Name ram" || fu.Embeds[0].Fields[1].Value != "• Name emilia" {
		t.Errorf("offer embed = %+v", fu.Embeds[0].Fields)
	}
	buttons := fu.Components[0].(discordgo.ActionsRow).Components
	if len(buttons) != 3 || buttons[1].(discordgo.Button).CustomID != tradePrefix+tradeCounter {
		t.Errorf("offer buttons = %+v", buttons)
	}
	if _, ok := f.bot.Sessions().Get(msgID); ok {
		t.Error("builder session should be gone after sending")
	}

	offer := f.message("fu-" + send.ID)
	f.run(f.click(bobID, offer, tradePrefix+tradeAccept))
	if got := f.api.lastRespond().Data.Content; got != "<@alice> <@bob> "+msgTradeAccepted {
		t.Errorf("accept = %q", got)
	}
	if !f.owns(bobID, "ram") || !f.owns(aliceID, "emilia") || !f.owns(aliceID, "rem") {
		t.Error("trade from builder not applied")
	}
}

func TestTradeBuilder_ValidationCancelAndExpiry(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem")
	f.give(bobID, "emilia")
	ic := f.slash(aliceID, commandWaifu, subTrade, resolvedUsers(bobID), userOption(bobID))
	f.run(ic)
	msg := f.message("msg-" + ic.ID)

	f.run(f.click(aliceID, msg, builderPrefix+builderSend))
	if !strings.Contains(f.api.lastRespond().Data.Embeds[0].Description, msgBuilderNothing) {
		t.Errorf("empty send = %q", f.api.lastRespond().Data.Embeds[0].Description)
	}

	f.run(f.selectValues(aliceID, msg, builderPrefix+builderGet, "emilia"))
	if _, ok, _ := f.players.SellOwned(t.Context(), domain.PlayerKey{GuildID: guildID, UserID: bobID}, "emilia", domain.SellPrice); !ok {
		t.Fatal("setup sale")
	}
	f.run(f.click(aliceID, msg, builderPrefix+builderSend))
	if !strings.Contains(f.api.lastRespond().Data.Embeds[0].Description, "<@bob> doesn't own emilia.") {
		t.Errorf("violation notice = %q", f.api.lastRespond().Data.Embeds[0].Description)
	}
	if f.api.lastFollowup() != nil {
		t.Error("no offer should be posted on a violation")
	}
	f.run(f.click(aliceID, msg, builderPrefix+"bogus"))

	f.run(f.click(aliceID, msg, builderPrefix+builderCancel))
	if got := f.api.lastRespond().Data.Content; got != msgBuilderCancelled {
		t.Errorf("cancel = %q", got)
	}
	f.run(f.click(aliceID, msg, builderPrefix+builderSend))
	if got := f.respondContent(); got != msgExpired {
		t.Errorf("after cancel = %q", got)
	}
	f.run(f.click(aliceID, nil, builderPrefix+builderSend))

	ic = f.slash(aliceID, commandWaifu, subTrade, resolvedUsers(bobID), userOption(bobID))
	f.run(ic)
	msg = f.message("msg-" + ic.ID)
	f.clock.now = f.clock.now.Add(time.Hour)
	f.run(f.click(aliceID, msg, builderPrefix+builderGiveNext))
	if got := f.respondContent(); got != msgExpired {
		t.Errorf("expired builder = %q", got)
	}
}

func TestTradeBuilder_PagingAndFilter(t *testing.T) {
	f := newFixture(t)
	slugs := make([]string, 30)
	for i := range slugs {
		slugs[i] = fmt.Sprintf("w%02d", i)
	}
	f.give(aliceID, slugs...)
	f.give(bobID, "rem")
	ic := f.slash(aliceID, commandWaifu, subTrade, resolvedUsers(bobID), userOption(bobID))
	f.run(ic)
	msg := f.message("msg-" + ic.ID)
	give, _, _ := builderRows(t, f.api.lastEdit())
	if len(give.Options) != 25 || give.Options[0].Value != "w00" {
		t.Fatalf("first page = %d options, first %s", len(give.Options), give.Options[0].Value)
	}
	if !strings.Contains((*f.api.lastEdit().Embeds)[0].Footer.Text, "Yours: page 1/2") {
		t.Errorf("footer = %q", (*f.api.lastEdit().Embeds)[0].Footer.Text)
	}

	f.run(f.selectValues(aliceID, msg, builderPrefix+builderGive, "w03"))
	f.run(f.click(aliceID, msg, builderPrefix+builderGiveNext))
	r := f.api.lastRespond()
	menu := r.Data.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if len(menu.Options) != 5 || menu.Options[0].Value != "w25" {
		t.Errorf("second page = %+v", menu.Options)
	}
	f.run(f.selectValues(aliceID, msg, builderPrefix+builderGive, "w27"))
	if v := f.api.lastRespond().Data.Embeds[0].Fields[0].Value; !strings.Contains(v, "Name w03") || !strings.Contains(v, "Name w27") {
		t.Errorf("selections should persist across pages: %q", v)
	}
	f.run(f.click(aliceID, msg, builderPrefix+builderGivePrev))
	menu = f.api.lastRespond().Data.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if !menu.Options[3].Default {
		t.Error("returning to page 1 should show w03 preselected")
	}
	f.run(f.click(aliceID, msg, builderPrefix+builderGetNext))
	f.run(f.click(aliceID, msg, builderPrefix+builderGetPrev))

	f.run(f.click(aliceID, msg, builderPrefix+builderFilter))
	if r := f.api.lastRespond(); r.Type != discordgo.InteractionResponseModal || r.Data.CustomID != builderFilterModal {
		t.Fatalf("filter should open a modal: %+v", r)
	}
	f.run(f.modal(aliceID, msg, builderFilterModal, builderFilterInput, "w2"))
	r = f.api.lastRespond()
	menu = r.Data.Components[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if len(menu.Options) != 10 || !strings.Contains(r.Data.Embeds[0].Description, "Filter: `w2`") {
		t.Errorf("filtered menu = %d options, desc %q", len(menu.Options), r.Data.Embeds[0].Description)
	}
	if len(r.Data.Components) != 3 {
		t.Errorf("their side has no match for the filter, so its menu should be hidden: %d rows", len(r.Data.Components))
	}
	f.run(f.modal(aliceID, msg, builderFilterModal, builderFilterInput, "zzz"))
	if !strings.Contains(f.api.lastRespond().Data.Embeds[0].Description, msgBuilderNoMatches) {
		t.Error("no-match notice expected")
	}
	f.run(f.modal(aliceID, msg, builderFilterModal, builderFilterInput, ""))
	if strings.Contains(f.api.lastRespond().Data.Embeds[0].Description, "Filter:") {
		t.Error("empty filter should clear")
	}
	f.run(f.modal(bobID, msg, builderFilterModal, builderFilterInput, "x"))
	if got := f.respondContent(); got != msgNotYourMenu {
		t.Errorf("stranger modal = %q", got)
	}
}

func TestTradeBuilder_Counter(t *testing.T) {
	f := newFixture(t)
	f.give(aliceID, "rem")
	f.give(bobID, "emilia", "ram")
	ic := f.tradeCmd(aliceID, bobID, "rem", "emilia")
	f.run(ic)
	offer := f.message("msg-" + ic.ID)

	f.run(f.click(aliceID, offer, tradePrefix+tradeCounter))
	if got := f.respondContent(); got != msgNotYourMenu {
		t.Errorf("sender cannot counter their own offer: %q", got)
	}

	counter := f.click(bobID, offer, tradePrefix+tradeCounter)
	f.run(counter)
	e := f.api.lastEdit()
	if editContent(e) != "Trade with <@alice>" {
		t.Fatalf("counter builder = %q", editContent(e))
	}
	if v := (*e.Embeds)[0].Fields[0].Value; !strings.Contains(v, "Name emilia") {
		t.Errorf("counter should be prefilled with what bob was asked for: %q", v)
	}
	if v := (*e.Embeds)[0].Fields[1].Value; !strings.Contains(v, "Name rem") {
		t.Errorf("counter should be prefilled with what alice offered: %q", v)
	}
	builder := f.message("msg-" + counter.ID)
	f.run(f.selectValues(bobID, builder, builderPrefix+builderGive, "ram"))
	send := f.click(bobID, builder, builderPrefix+builderSend)
	f.run(send)
	fu := f.api.lastFollowup()
	if fu == nil || fu.Content != "<@alice>: <@bob> is offering the following trade request:" || fu.Embeds[0].Fields[0].Value != "• Name ram" {
		t.Fatalf("counter offer = %+v", fu)
	}
	me := f.api.lastMsgEdit()
	if me == nil || me.ID != offer.ID || *me.Content != "<@bob> countered this offer; see the new request below." {
		t.Errorf("original offer should be marked countered: %+v", me)
	}
	if _, ok := f.bot.Sessions().Get(offer.ID); ok {
		t.Error("original offer session should be closed")
	}
	newOffer := f.message("fu-" + send.ID)
	f.run(f.click(aliceID, newOffer, tradePrefix+tradeAccept))
	if got := f.api.lastRespond().Data.Content; got != "<@bob> <@alice> "+msgTradeAccepted {
		t.Errorf("accepting the counter = %q", got)
	}
	if !f.owns(aliceID, "ram") || !f.owns(bobID, "rem") {
		t.Error("counter trade not applied")
	}
}

func TestTrade_BuilderPathValidation(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, commandWaifu, subTrade, nil))
	if got := f.respondContent(); got != "<@alice> "+msgUserNotFound {
		t.Errorf("missing user = %q", got)
	}
	f.run(f.slash(aliceID, commandWaifu, subTrade, resolvedUsers(aliceID), userOption(aliceID)))
	if got := f.respondContent(); got != "<@alice> You can't trade with yourself." {
		t.Errorf("self = %q", got)
	}
	robot := resolvedUsers("robot")
	robot.Users["robot"].Bot = true
	f.run(f.slash(aliceID, commandWaifu, subTrade, robot, userOption("robot")))
	if got := f.respondContent(); got != "<@alice> You can't trade with a bot." {
		t.Errorf("bot = %q", got)
	}
	f.api.failEdit = true
	f.run(f.slash(aliceID, commandWaifu, subTrade, resolvedUsers(bobID), userOption(bobID)))
	if f.bot.Sessions().Len() != 0 {
		t.Error("no session when the builder could not be sent")
	}
}

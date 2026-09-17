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

func TestBanner_CardShowsSeriesFeaturedAndCountdown(t *testing.T) {
	f := newFixture(t)
	f.seedBanner("ranked-050", "ranked-000", "ranked-020")
	ic := f.slash(aliceID, commandWaifu, subBanner, nil)
	f.run(ic)
	e := f.api.lastEdit()
	if got := editContent(e); got != "<@alice> Here's this week's banner:\nRefreshes in 118:00:00" {
		t.Errorf("content = %q", got)
	}
	embed := (*e.Embeds)[0]
	if embed.Title != "Re:Zero" || embed.Image == nil || embed.Image.URL != "https://img/re-zero" || embed.Footer == nil {
		t.Errorf("embed header = %+v", embed)
	}
	if len(embed.Fields) != 1 || embed.Fields[0].Name != "Featured this week" {
		t.Fatalf("fields = %+v", embed.Fields)
	}
	lines := strings.Split(embed.Fields[0].Value, "\n")
	if len(lines) != 3 || lines[0] != "★★★★★ Ranked 000 · Rank #1" || lines[2] != "★★☆☆☆ Ranked 050 · Rank #51" {
		t.Errorf("featured lines = %q", lines)
	}
	menu := (*e.Components)[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu)
	if menu.CustomID != viewWaifuMenu || menu.Placeholder != viewPlaceholder || len(menu.Options) != 3 || menu.Options[0].Value != "ranked-000" || menu.Options[0].Description != "⭐⭐⭐⭐⭐ · rank #1" {
		t.Errorf("view menu = %+v", menu)
	}
	button := (*e.Components)[1].(discordgo.ActionsRow).Components[0].(discordgo.Button)
	if button.CustomID != bannerPrefix+bannerRoll || button.Label != "Roll on banner · 400 coins" {
		t.Errorf("button = %+v", button)
	}

	f.run(f.dm(aliceID, commandWaifu, subBanner))
	if got := editContent(f.api.lastEdit()); !strings.HasPrefix(got, "<@alice> Here's this week's banner:") {
		t.Errorf("banner card should work in DMs: %q", got)
	}
}

func TestBanner_NoBannerText(t *testing.T) {
	f := newFixture(t)
	f.run(f.slash(aliceID, commandWaifu, subBanner, nil))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgNoBanner {
		t.Errorf("no banner = %q", got)
	}
	f.fund(aliceID, 400)
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil, boolOpt(optBanner, true)))
	if got := editContent(f.api.lastEdit()); got != "<@alice> "+msgNoBanner {
		t.Errorf("banner roll without banner = %q", got)
	}
	if f.coins(aliceID) != 600 {
		t.Errorf("player was charged: %d", f.coins(aliceID))
	}
}

func TestBanner_ButtonRollsForAnyClickerWithoutTouchingCard(t *testing.T) {
	f := newFixture(t)
	f.seedBanner("ranked-000", "ranked-005", "ranked-020", "ranked-050", "ranked-100")
	card := f.slash(aliceID, commandWaifu, subBanner, nil)
	f.run(card)
	cardMsg := f.message("msg-" + card.ID)
	before := cardMsg.Content

	f.fund(bobID, 600)
	f.script(d100(5), 2)
	click := f.click(bobID, cardMsg, bannerPrefix+bannerRoll)
	f.run(click)
	if r := f.api.calls[len(f.api.calls)-2].resp; r == nil || r.Type != discordgo.InteractionResponseDeferredChannelMessageWithSource {
		t.Errorf("banner button should reply with a new message, got %+v", r)
	}
	e := f.api.lastEdit()
	if got := editContent(e); got != "<@bob> :confetti_ball: **BANNER ROLL!!** :confetti_ball: Here's who you rolled:\nBalance: :coin: 400 coins" {
		t.Errorf("banner roll result = %q", got)
	}
	if title := (*e.Embeds)[0].Title; !strings.Contains(title, "Name ranked-020") {
		t.Errorf("rolled card = %q", title)
	}
	again := (*e.Components)[0].(discordgo.ActionsRow).Components[0].(discordgo.Button)
	if again.CustomID != rollPrefix+rollBanner+":"+bobID || again.Label != "Roll on banner again · 400 coins" {
		t.Errorf("again button = %+v", again)
	}
	if f.coins(bobID) != 400 || !f.owns(bobID, "ranked-020") {
		t.Errorf("bob state: coins=%d owns=%v", f.coins(bobID), f.owns(bobID, "ranked-020"))
	}
	if f.message(cardMsg.ID).Content != before {
		t.Error("the shared banner card must not be edited by a roll")
	}

	result := f.message("msg-" + click.ID)
	f.run(f.click(aliceID, result, again.CustomID))
	if got := f.respondContent(); got != msgNotYourRoll {
		t.Errorf("other user on again button = %q", got)
	}
	f.script(d100(15), 9)
	f.run(f.click(bobID, result, again.CustomID))
	e = f.api.lastEdit()
	if got := editContent(e); got != "<@bob> :sparkles: **CRITICAL ROLL!!** :sparkles: Here's who you rolled:\nBalance: :coin: 0 coins" {
		t.Errorf("banner again result = %q", got)
	}
	if f.coins(bobID) != 0 || !f.owns(bobID, "ranked-009") {
		t.Errorf("second banner roll: coins=%d", f.coins(bobID))
	}
	if btn := (*e.Components)[0].(discordgo.ActionsRow).Components[0].(discordgo.Button); btn.CustomID != rollPrefix+rollBanner+":"+bobID {
		t.Errorf("a critical on a banner roll should keep the banner again button: %+v", btn)
	}

	f.api.reset()
	f.run(f.click(bobID, result, again.CustomID))
	if f.api.last().kind != "followup" || f.api.lastEdit() != nil {
		t.Error("broke banner again should be an ephemeral follow-up without editing the card")
	}
}

func TestBanner_ButtonInDMAndInsufficientCoins(t *testing.T) {
	f := newFixture(t)
	f.seedBanner("ranked-000", "ranked-005", "ranked-020", "ranked-050", "ranked-100")
	card := f.slash(aliceID, commandWaifu, subBanner, nil)
	f.run(card)
	cardMsg := f.message("msg-" + card.ID)

	dm := f.click(bobID, cardMsg, bannerPrefix+bannerRoll)
	dm.GuildID = ""
	f.run(dm)
	if got := f.respondContent(); got != "The roll command cannot be invoked from the direct messages of the bot." {
		t.Errorf("dm = %q", got)
	}

	f.give(bobID)
	f.run(f.click(bobID, cardMsg, bannerPrefix+bannerRoll))
	if got := editContent(f.api.lastEdit()); got != "<@bob> "+msgInsufficientCoins {
		t.Errorf("broke click = %q", got)
	}
	f.run(f.click(bobID, cardMsg, bannerPrefix+"bogus"))
}

func TestRoll_BannerOptionTexts(t *testing.T) {
	f := newFixture(t)
	f.seedBanner("ranked-000", "ranked-005", "ranked-020", "ranked-050", "ranked-100")
	f.fund(aliceID, 1000)

	f.script(d100(1), 0)
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil, boolOpt(optBanner, true)))
	if got := editContent(f.api.lastEdit()); got != "<@alice> :confetti_ball: **BANNER ROLL!!** :confetti_ball: Here's who you rolled:\nBalance: :coin: 800 coins" {
		t.Errorf("banner hit = %q", got)
	}
	if f.coins(aliceID) != 800 || !f.owns(aliceID, "ranked-000") {
		t.Errorf("after banner hit: coins=%d", f.coins(aliceID))
	}

	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil, boolOpt(optBanner, true)))
	if got := editContent(f.api.lastEdit()); got != "<@alice> Here's who you rolled:\nBalance: :coin: 400 coins" || f.coins(aliceID) != 400 {
		t.Errorf("regular on banner = %q coins=%d", got, f.coins(aliceID))
	}

	f.script(d100(50))
	f.source.EXPECT().Random(mock.Anything).Return(summary("ram"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRoll, nil, boolOpt(optBanner, false)))
	if f.coins(aliceID) != 200 {
		t.Errorf("banner:false should be a normal 200 coin roll, coins=%d", f.coins(aliceID))
	}

	f.give(bobID, "ranked-000", "ranked-005", "ranked-020", "ranked-050", "ranked-100")
	f.fund(bobID, 200)
	f.script(d100(3), 11)
	f.run(f.slash(bobID, commandWaifu, subRoll, nil, boolOpt(optBanner, true)))
	if got := editContent(f.api.lastEdit()); got != "<@bob> :sparkles: **CRITICAL ROLL!!** :sparkles: Here's who you rolled:\nBalance: :coin: 0 coins" {
		t.Errorf("degraded banner roll = %q", got)
	}
	if f.coins(bobID) != 0 || !f.owns(bobID, "ranked-011") {
		t.Errorf("degraded roll state: coins=%d", f.coins(bobID))
	}
}

func TestBannerEmbed_CapsFeaturedList(t *testing.T) {
	chars := make([]domain.RankedWaifu, 20)
	for i := range chars {
		chars[i] = domain.RankedWaifu{WaifuSummary: domain.WaifuSummary{Slug: fmt.Sprintf("c%02d", i), Name: fmt.Sprintf("C%02d", i)}, Position: i + 1, Stars: 1}
	}
	e := bannerEmbed(domain.NewBanner(time.Time{}, domain.Series{Slug: "s", Name: "S", Description: "d"}, chars))
	lines := strings.Split(e.Fields[0].Value, "\n")
	if len(lines) != domain.BannerCardLimit+1 || lines[len(lines)-1] != "+5 more" {
		t.Errorf("lines = %d, last %q", len(lines), lines[len(lines)-1])
	}
	if empty := bannerEmbed(domain.Banner{Series: domain.Series{Name: "S"}}); len(empty.Fields) != 0 {
		t.Errorf("empty banner should have no field: %+v", empty.Fields)
	}
	if rows := bannerCardComponents(domain.Banner{}); len(rows) != 1 {
		t.Errorf("empty banner should only have the roll button: %d rows", len(rows))
	}
	many := make([]domain.RankedWaifu, 30)
	for i := range many {
		many[i] = domain.RankedWaifu{WaifuSummary: domain.WaifuSummary{Slug: fmt.Sprintf("m%02d", i), Name: fmt.Sprintf("M%02d", i)}, Position: i + 1, Stars: 1}
	}
	rows := bannerCardComponents(domain.Banner{Characters: many})
	if menu := rows[0].(discordgo.ActionsRow).Components[0].(discordgo.SelectMenu); len(menu.Options) != viewMenuLimit {
		t.Errorf("view menu should cap at %d options, got %d", viewMenuLimit, len(menu.Options))
	}
}

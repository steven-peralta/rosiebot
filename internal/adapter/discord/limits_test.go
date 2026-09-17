package discord

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"
)

func TestOversized(t *testing.T) {
	big := strings.Repeat("あ", 1025)
	ok := &discordgo.MessageEmbed{Title: "t", Description: strings.Repeat("d", 4096), Fields: []*discordgo.MessageEmbedField{{Name: "n", Value: strings.Repeat("v", 1024)}}}
	cases := map[string]struct {
		content string
		embeds  []*discordgo.MessageEmbed
		want    bool
	}{
		"fits":              {"hi", []*discordgo.MessageEmbed{ok}, false},
		"content":           {strings.Repeat("x", 2001), nil, true},
		"content multibyte": {strings.Repeat("あ", 2000), nil, false},
		"too many embeds":   {"", make([]*discordgo.MessageEmbed, 11), true},
		"title":             {"", []*discordgo.MessageEmbed{{Title: strings.Repeat("t", 257)}}, true},
		"description":       {"", []*discordgo.MessageEmbed{{Description: strings.Repeat("d", 4097)}}, true},
		"field value":       {"", []*discordgo.MessageEmbed{{Fields: []*discordgo.MessageEmbedField{{Name: "n", Value: big}}}}, true},
		"field name":        {"", []*discordgo.MessageEmbed{{Fields: []*discordgo.MessageEmbedField{{Name: strings.Repeat("n", 257), Value: "v"}}}}, true},
		"too many fields":   {"", []*discordgo.MessageEmbed{{Fields: make([]*discordgo.MessageEmbedField, 26)}}, true},
		"footer":            {"", []*discordgo.MessageEmbed{{Footer: &discordgo.MessageEmbedFooter{Text: strings.Repeat("f", 2049)}}}, true},
		"author":            {"", []*discordgo.MessageEmbed{{Author: &discordgo.MessageEmbedAuthor{Name: strings.Repeat("a", 257)}}}, true},
		"embed total": {"", []*discordgo.MessageEmbed{{Description: strings.Repeat("d", 4000), Fields: []*discordgo.MessageEmbedField{
			{Name: "a", Value: strings.Repeat("v", 1000)}, {Name: "b", Value: strings.Repeat("v", 1000)}, {Name: "c", Value: strings.Repeat("v", 100)},
		}}}, true},
		"nil entries": {"", []*discordgo.MessageEmbed{nil, {Fields: []*discordgo.MessageEmbedField{nil}}}, false},
	}
	for name, c := range cases {
		if got := oversized(c.content, c.embeds); got != c.want {
			t.Errorf("%s: oversized = %v, want %v", name, got, c.want)
		}
	}
}

func TestOversizedReplyIsRefused(t *testing.T) {
	f := newFixture(t)
	huge := detail("rem")
	huge.Description = strings.Repeat("x", 300)
	huge.Appearances = nil
	huge.Name = strings.Repeat("N", 300)
	f.source.ExpectedCalls = nil
	f.source.EXPECT().Get(mock.Anything, "rem").Return(huge, nil).Once()
	f.source.EXPECT().Random(mock.Anything).Return(summary("rem"), nil).Once()
	f.run(f.slash(aliceID, commandWaifu, subRandom, nil))
	e := f.api.lastEdit()
	if editContent(e) != "<@alice> "+msgResponseTooLarge || len(*e.Embeds) != 0 || len(*e.Components) != 0 {
		t.Errorf("oversized reply = %q %+v", editContent(e), e)
	}
	edits := 0
	for _, c := range f.api.calls {
		if c.kind == "edit" {
			edits++
		}
	}
	if edits != 1 {
		t.Errorf("an oversized reply must never be sent to Discord, got %d edits", edits)
	}

	f.give(aliceID, "a", "b")
	f.source.ExpectedCalls = nil
	f.source.EXPECT().Get(mock.Anything, mock.Anything).Return(huge, nil).Maybe()
	owned := f.slash(aliceID, commandWaifu, subOwned, nil)
	f.run(owned)
	msg := f.message("msg-" + owned.ID)
	f.run(f.click(aliceID, msg, pagerPrefix+pagerNext))
	if got := f.respondContent(); got != msgResponseTooLarge {
		t.Errorf("oversized page update = %q", got)
	}
}

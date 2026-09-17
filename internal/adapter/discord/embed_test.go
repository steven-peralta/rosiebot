package discord

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func ptrF(v float64) *float64 { return &v }
func ptrI(v int) *int         { return &v }

func field(e *discordgo.MessageEmbed, name string) *discordgo.MessageEmbedField {
	for _, f := range e.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

func fullWaifu() domain.Waifu {
	return domain.Waifu{
		WaifuSummary: domain.WaifuSummary{Slug: "rem", Name: "Rem", OriginalName: "レム", RomajiName: "Remu", PictureURL: "https://img/rem", Likes: 16199, Trash: 3203},
		URL:          "https://www.mywaifulist.moe/waifu/rem",
		Description:  strings.Repeat("a", 300),
		NSFW:         true,
		Weight:       ptrF(45),
		Height:       ptrF(154),
		Bust:         ptrF(81),
		Hip:          ptrF(83),
		Waist:        ptrF(56),
		BloodType:    "A",
		Origin:       "Lugnica",
		Age:          ptrI(0),
		Appearances:  []domain.Series{{Name: "Re:Zero", URL: "https://www.mywaifulist.moe/series/re-zero"}, {Name: "Isekai Quartet"}},
	}
}

func TestWaifuEmbed_FullCard(t *testing.T) {
	e := waifuEmbed(fullWaifu(), &domain.RankedWaifu{Position: 3, Stars: 5})

	if e.Title != ":star::star::star::star::star:\n🔞 Rem" || e.URL != "https://www.mywaifulist.moe/waifu/rem" {
		t.Errorf("title/url = %q %q", e.Title, e.URL)
	}
	if e.Author == nil || e.Author.Name != "Re:Zero" || e.Author.URL != "https://www.mywaifulist.moe/series/re-zero" {
		t.Errorf("author = %+v", e.Author)
	}
	if e.Image == nil || e.Image.URL != "https://img/rem" || e.Color != starColors[5] {
		t.Errorf("image/color = %+v %x", e.Image, e.Color)
	}
	lines := strings.Split(e.Description, "\n")
	if lines[0] != "*レム · Remu*" {
		t.Errorf("alt names line = %q", lines[0])
	}
	if lines[1] != "Rank #3" {
		t.Errorf("rating line = %q", lines[1])
	}
	if lines[2] != "❤️ 16,199 · 🗑️ 3,203 · 83% liked" {
		t.Errorf("votes line = %q", lines[2])
	}
	if lines[3] != "" || !strings.HasPrefix(lines[4], "||aaaa") || !strings.HasSuffix(lines[4], "...||") || len([]rune(lines[4])) != 256+3+4 {
		t.Errorf("description = %q", lines[4])
	}

	vitals := field(e, "Vitals")
	if vitals == nil || !vitals.Inline || vitals.Value != "Height 154 cm (5′0″)\nWeight 45 kg (99 lb)\nB·W·H 81/56/83" {
		t.Errorf("vitals = %+v", vitals)
	}
	details := field(e, "Details")
	if details == nil || !details.Inline || details.Value != "Age 0\nBlood type A\nOrigin ||Lugnica||" {
		t.Errorf("details = %+v", details)
	}
	appears := field(e, "Appears in")
	if appears == nil || appears.Inline || appears.Value != "Re:Zero · Isekai Quartet" {
		t.Errorf("appears = %+v", appears)
	}
	if len(e.Fields) != 3 {
		t.Errorf("fields = %d", len(e.Fields))
	}
}

func TestWaifuEmbed_SparseAndUnranked(t *testing.T) {
	w := domain.Waifu{WaifuSummary: domain.WaifuSummary{Slug: "x", Name: "X", Likes: 1, Trash: 2}}
	e := waifuEmbed(w, nil)
	if e.Title != "X" || e.Author != nil || e.Image != nil || e.Color != 0 || len(e.Fields) != 0 {
		t.Errorf("sparse = %+v", e)
	}
	if zeroStars := waifuEmbed(w, &domain.RankedWaifu{Stars: 0}); zeroStars.Color != 0 {
		t.Errorf("zero stars should have no accent colour: %x", zeroStars.Color)
	}
	if e.Description != "Unranked\n❤️ 1 · 🗑️ 2 · 33% liked" {
		t.Errorf("sparse description = %q", e.Description)
	}
	zero := waifuEmbed(domain.Waifu{WaifuSummary: domain.WaifuSummary{Name: "Z"}}, &domain.RankedWaifu{Stars: 0})
	if !strings.HasPrefix(zero.Description, "Unranked\n❤️ 0 · 🗑️ 0") || strings.Contains(zero.Description, "liked") {
		t.Errorf("zero votes = %q", zero.Description)
	}
	partial := waifuEmbed(domain.Waifu{WaifuSummary: domain.WaifuSummary{Name: "P", OriginalName: "P", RomajiName: "Pee"}, Bust: ptrF(80)}, &domain.RankedWaifu{Position: 1200, Stars: 2})
	if !strings.HasPrefix(partial.Description, "*Pee*\nRank #1,200") || !strings.HasPrefix(partial.Title, ":star::star:\nP") {
		t.Errorf("partial card = %q / %q", partial.Title, partial.Description)
	}
	if zero.Title != "Z" {
		t.Errorf("zero stars should not add a star line: %q", zero.Title)
	}
	if v := field(partial, "Vitals"); v == nil || v.Value != "B·W·H 80/?/?" {
		t.Errorf("partial vitals = %+v", v)
	}
	if partial.Color != starColors[2] {
		t.Errorf("two-star colour = %x", partial.Color)
	}
	brief := waifuEmbed(domain.Waifu{WaifuSummary: domain.WaifuSummary{Name: "B"}, Description: "  brief  "}, nil)
	if !strings.HasSuffix(brief.Description, "\n\n||brief||") {
		t.Errorf("brief description = %q", brief.Description)
	}
}

func TestWaifuEmbed_AppearancesCapped(t *testing.T) {
	w := domain.Waifu{WaifuSummary: domain.WaifuSummary{Name: "Many"}}
	for i := range 9 {
		w.Appearances = append(w.Appearances, domain.Series{Name: string(rune('A' + i))})
	}
	e := waifuEmbed(w, nil)
	if got := field(e, "Appears in").Value; got != "A · B · C · D · E · F · +3 more" {
		t.Errorf("appearances = %q", got)
	}
}

func TestHeightAndWeightConversions(t *testing.T) {
	cases := []struct {
		height, weight float64
		wantH, wantW   string
	}{
		{height: 170, weight: 60, wantH: "Height 170 cm (5′6″)", wantW: "Weight 60 kg (132 lb)"},
		{height: 182.5, weight: 70.4, wantH: "Height 182.5 cm (5′11″)", wantW: "Weight 70.4 kg (155 lb)"},
		{height: 30.48, weight: 0.5, wantH: "Height 30.48 cm (1′0″)", wantW: "Weight 0.5 kg (1 lb)"},
	}
	for _, c := range cases {
		lines := vitalsLines(domain.Waifu{Height: ptrF(c.height), Weight: ptrF(c.weight)})
		if lines[0] != c.wantH || lines[1] != c.wantW {
			t.Errorf("%v/%v = %v", c.height, c.weight, lines)
		}
	}
}

func TestThousands(t *testing.T) {
	cases := map[int]string{0: "0", 999: "999", 1000: "1,000", 16199: "16,199", 1234567: "1,234,567"}
	for n, want := range cases {
		if got := thousands(n); got != want {
			t.Errorf("thousands(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestSeriesEmbedAndFooter(t *testing.T) {
	f := newFixture(t)
	e := seriesEmbed(domain.Series{Name: "S", URL: "u", Description: strings.Repeat("b", 300), PictureURL: "p"})
	if e.Title != "S" || e.URL != "u" || !strings.HasSuffix(e.Description, "...") || e.Image.URL != "p" {
		t.Errorf("series = %+v", e)
	}
	if got := f.bot.footer(0).Text; got != "rosiebot vtest" {
		t.Errorf("footer = %q", got)
	}
	if got := f.bot.footer(1500 * time.Millisecond).Text; got != "rosiebot vtest · 1500ms" {
		t.Errorf("footer with time = %q", got)
	}
	withRank := f.bot.waifuEmbed(detail("ranked-000"), 0)
	if !strings.HasPrefix(withRank.Title, ":star::star::star::star::star:\n") || !strings.Contains(withRank.Description, "Rank #1") || withRank.Color != starColors[5] {
		t.Errorf("ranking lookup not applied: %+v", withRank)
	}
}

func TestSlugFromEmbeds(t *testing.T) {
	cases := []struct {
		url  string
		want string
		ok   bool
	}{
		{url: "https://www.mywaifulist.moe/waifu/rem", want: "rem", ok: true},
		{url: "https://mywaifulist.moe/waifu/rem-re-zero/", want: "rem-re-zero", ok: true},
		{url: "https://www.mywaifulist.moe/series/re-zero"},
		{url: "https://evil.example/waifu/rem"},
		{url: "https://www.mywaifulist.moe/waifu/"},
		{url: "https://www.mywaifulist.moe/waifu/a/b"},
		{url: "::not a url"},
		{url: ""},
	}
	for _, c := range cases {
		got, ok := SlugFromEmbeds([]*discordgo.MessageEmbed{nil, {URL: c.url}})
		if ok != c.ok || got != c.want {
			t.Errorf("SlugFromEmbeds(%q) = %q, %v", c.url, got, ok)
		}
	}
	if _, ok := SlugFromEmbeds(nil); ok {
		t.Error("no embeds")
	}
}

func TestErrorText_V1Strings(t *testing.T) {
	cases := map[string]struct {
		err  error
		want string
	}{
		"coins":      {app.ErrInsufficientCoins, msgInsufficientCoins},
		"exhausted":  {app.ErrRollExhausted, msgRollExhausted},
		"daily":      {&app.DailyAlreadyClaimedError{RefreshIn: 3661 * time.Second}, "You've already claimed your daily for today. You can claim again in 01:01:01"},
		"conflict":   {app.ErrTradeConflict, msgTradeConflict},
		"self":       {domain.ErrTradeWithSelf, "You can't trade with yourself."},
		"empty":      {domain.ErrTradeEmpty, "A trade needs at least one waifu on either side."},
		"overlap":    {domain.ErrTradeOverlap, "A waifu can't be on both sides of a trade."},
		"not found":  {app.ErrNotFound, msgNoData},
		"ratelimit":  {app.ErrRateLimited, msgRateLimited},
		"no banner":  {app.ErrNoBanner, msgNoBanner},
		"unexpected": {errors.New("boom"), msgUnexpected},
		"violation":  {&domain.TradeViolation{Slug: "rem", Side: domain.TradeSideTarget, Err: domain.ErrTradeAlreadyOwn}, "they already owns rem"},
	}
	for name, c := range cases {
		if got := errorText(c.err); got != c.want {
			t.Errorf("%s: errorText = %q, want %q", name, got, c.want)
		}
	}
	v := &domain.TradeViolation{Slug: "rem", Side: domain.TradeSideTarget, Err: domain.ErrTradeNotOwned}
	if got := violationText(v, "<@bob>", "Rem"); got != "<@bob> doesn't own Rem" {
		t.Errorf("violationText = %q", got)
	}
	if coinWord(1) != "1 coin" || coinWord(2) != "2 coins" || coinWord(0) != "0 coins" {
		t.Error("coinWord")
	}
	if truncate("héllo", 3) != "hél..." || truncate("hi", 5) != "hi" {
		t.Error("truncate")
	}
}

func TestPagerComponents(t *testing.T) {
	rows := pagerComponents(0, 3, false)
	if len(rows) != 1 {
		t.Fatalf("rows = %d", len(rows))
	}
	row := rows[0].(discordgo.ActionsRow)
	first := row.Components[0].(discordgo.Button)
	next := row.Components[3].(discordgo.Button)
	if !first.Disabled || next.Disabled || first.CustomID != pagerPrefix+pagerFirst {
		t.Errorf("page 0 = first:%v next:%v", first.Disabled, next.Disabled)
	}
	row = pagerComponents(2, 3, false)[0].(discordgo.ActionsRow)
	if !row.Components[3].(discordgo.Button).Disabled || row.Components[0].(discordgo.Button).Disabled {
		t.Error("last page should disable forward buttons only")
	}
	if len(pagerComponents(0, 1, false)) != 0 {
		t.Error("single page without sell has no rows")
	}
	rows = pagerComponents(0, 1, true)
	if len(rows) != 1 || rows[0].(discordgo.ActionsRow).Components[0].(discordgo.Button).CustomID != sellPrefix+sellAsk {
		t.Errorf("single sellable page = %+v", rows)
	}
	if len(pagerComponents(1, 3, true)) != 2 {
		t.Error("multi-page sellable should have pager and sell rows")
	}
}

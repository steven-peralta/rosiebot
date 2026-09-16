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

func TestWaifuEmbed_AllFields(t *testing.T) {
	w := domain.Waifu{
		WaifuSummary: domain.WaifuSummary{Slug: "rem", Name: "Rem", OriginalName: "レム", PictureURL: "https://img/rem", Likes: 16199, Trash: 3203},
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
		Appearances:  []domain.Series{{Name: "Re:Zero"}, {Name: "Isekai Quartet"}},
	}
	ranked := &domain.RankedWaifu{Position: 3, Stars: 5}
	e := waifuEmbed(w, ranked)

	if e.Title != ":star::star::star::star::star:\n:underage: Rem - レム" {
		t.Errorf("title = %q", e.Title)
	}
	if e.URL != w.URL || e.Image == nil || e.Image.URL != w.PictureURL || e.Color != brandingColor {
		t.Errorf("url/image/color = %q %+v %x", e.URL, e.Image, e.Color)
	}
	if !strings.HasPrefix(e.Description, "**Description** (may have spoilers):\n||") || !strings.HasSuffix(e.Description, "...||") || len([]rune(e.Description)) != len([]rune("**Description** (may have spoilers):\n||||"))+256+3 {
		t.Errorf("description = %q", e.Description)
	}
	checks := map[string]string{
		":heart: Likes":              "16199",
		":wastebasket: Trash":        "3203",
		":trophy: Rank":              "#3",
		":scales: Weight":            "45 kg (99 lbs)",
		":straight_ruler: Height":    "154 cm (5 ft 0 in)",
		":bikini: Bust":              "81 cm",
		":pear: Hip":                 "83 cm",
		":jeans: Waist":              "56 cm",
		":earth_americas: Origin":    "||Lugnica||",
		":calendar_spiral: Age":      "0",
		":drop_of_blood: Blood Type": "A",
		":book: Series":              "Re:Zero",
	}
	for name, want := range checks {
		f := field(e, name)
		if f == nil || f.Value != want || !f.Inline {
			t.Errorf("field %q = %+v, want %q inline", name, f, want)
		}
	}
	appears := e.Fields[len(e.Fields)-1]
	if appears.Name != ":camera_with_flash: Appears In" || appears.Value != "Re:Zero, Isekai Quartet" || appears.Inline {
		t.Errorf("appears = %+v", appears)
	}
	inline := 0
	for _, f := range e.Fields {
		if f.Inline {
			inline++
		}
	}
	if inline%3 != 0 {
		t.Errorf("inline fields should be padded to a multiple of 3, got %d", inline)
	}
}

func TestWaifuEmbed_Sparse(t *testing.T) {
	w := domain.Waifu{WaifuSummary: domain.WaifuSummary{Slug: "x", Name: "X", Likes: 1, Trash: 2}}
	e := waifuEmbed(w, nil)
	if e.Title != "X" || e.Description != "" || e.Image != nil {
		t.Errorf("sparse = %+v", e)
	}
	if field(e, ":trophy: Rank") != nil || field(e, ":scales: Weight") != nil || field(e, ":book: Series") != nil {
		t.Error("absent data must not produce fields")
	}
	if e.Fields[len(e.Fields)-1].Name == ":camera_with_flash: Appears In" {
		t.Error("no appearances field without appearances")
	}
	unranked := waifuEmbed(w, &domain.RankedWaifu{Position: 0, Stars: 0})
	if strings.Contains(unranked.Title, ":star:") {
		t.Error("zero stars should not render a star line")
	}
	short := waifuEmbed(domain.Waifu{WaifuSummary: domain.WaifuSummary{Name: "S"}, Description: "brief"}, nil)
	if short.Description != "**Description** (may have spoilers):\n||brief||" {
		t.Errorf("short description = %q", short.Description)
	}
}

func TestHeightAndWeightConversions(t *testing.T) {
	cases := []struct {
		height, weight float64
		wantH, wantW   string
	}{
		{height: 170, weight: 60, wantH: "170 cm (5 ft 6 in)", wantW: "60 kg (132 lbs)"},
		{height: 182.5, weight: 70.4, wantH: "182.5 cm (5 ft 11 in)", wantW: "70.4 kg (155 lbs)"},
		{height: 30.48, weight: 0.5, wantH: "30.48 cm (1 ft 0 in)", wantW: "0.5 kg (1 lbs)"},
	}
	for _, c := range cases {
		e := waifuEmbed(domain.Waifu{Height: ptrF(c.height), Weight: ptrF(c.weight)}, nil)
		if got := field(e, ":straight_ruler: Height").Value; got != c.wantH {
			t.Errorf("height %v = %q, want %q", c.height, got, c.wantH)
		}
		if got := field(e, ":scales: Weight").Value; got != c.wantW {
			t.Errorf("weight %v = %q, want %q", c.weight, got, c.wantW)
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
	if got := f.bot.footer(1500 * time.Millisecond).Text; got != "rosiebot vtest (1500ms)" {
		t.Errorf("footer with time = %q", got)
	}
	withRank := f.bot.waifuEmbed(detail("ranked-000"), 0)
	if !strings.HasPrefix(withRank.Title, ":star:") || field(withRank, ":trophy: Rank").Value != "#1" {
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
	row := pagerComponents(0, 3)[0].(discordgo.ActionsRow)
	first := row.Components[0].(discordgo.Button)
	next := row.Components[3].(discordgo.Button)
	if !first.Disabled || next.Disabled || first.CustomID != pagerPrefix+pagerFirst {
		t.Errorf("page 0 = first:%v next:%v", first.Disabled, next.Disabled)
	}
	row = pagerComponents(2, 3)[0].(discordgo.ActionsRow)
	if row.Components[3].(discordgo.Button).Disabled != true || row.Components[0].(discordgo.Button).Disabled {
		t.Error("last page should disable forward buttons only")
	}
	row = pagerComponents(0, 1)[0].(discordgo.ActionsRow)
	for _, c := range row.Components {
		if !c.(discordgo.Button).Disabled {
			t.Error("single page should disable everything")
		}
	}
}

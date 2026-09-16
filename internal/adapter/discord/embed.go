package discord

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/steven-peralta/rosiebot/internal/domain"
)

const (
	brandingColor      = 0x7752a0
	descriptionLimit   = 256
	mwlWaifuPathPrefix = "/waifu/"
	blank              = "\u200b"
)

func truncate(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "..."
}

func (b *Bot) footer(elapsed time.Duration) *discordgo.MessageEmbedFooter {
	text := "rosiebot v" + b.cfg.Version
	if elapsed > 0 {
		text += fmt.Sprintf(" (%dms)", elapsed.Milliseconds())
	}
	return &discordgo.MessageEmbedFooter{Text: text}
}

func (b *Bot) waifuEmbed(w domain.Waifu, elapsed time.Duration) *discordgo.MessageEmbed {
	var ranked *domain.RankedWaifu
	if r, ok := b.svc.Ranking.Current().Lookup(w.Slug); ok {
		ranked = &r
	}
	e := waifuEmbed(w, ranked)
	e.Footer = b.footer(elapsed)
	return e
}

func waifuEmbed(w domain.Waifu, ranked *domain.RankedWaifu) *discordgo.MessageEmbed {
	var title strings.Builder
	if ranked != nil && ranked.Stars > 0 {
		title.WriteString(strings.Repeat(":star:", ranked.Stars))
		title.WriteString("\n")
	}
	if w.NSFW {
		title.WriteString(":underage: ")
	}
	title.WriteString(w.Name)
	if w.OriginalName != "" {
		title.WriteString(" - " + w.OriginalName)
	}

	e := &discordgo.MessageEmbed{Title: title.String(), URL: w.URL, Color: brandingColor}
	if w.PictureURL != "" {
		e.Image = &discordgo.MessageEmbedImage{URL: w.PictureURL}
	}
	if w.Description != "" {
		e.Description = "**Description** (may have spoilers):\n||" + truncate(w.Description, descriptionLimit) + "||"
	}

	add := func(name, value string) {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: name, Value: value, Inline: true})
	}
	add(":heart: Likes", strconv.Itoa(w.Likes))
	add(":wastebasket: Trash", strconv.Itoa(w.Trash))
	if ranked != nil && ranked.Position > 0 {
		add(":trophy: Rank", "#"+strconv.Itoa(ranked.Position))
	}
	if w.Weight != nil {
		add(":scales: Weight", fmt.Sprintf("%s kg (%d lbs)", num(*w.Weight), int(math.Round(*w.Weight*2.20462))))
	}
	if w.Height != nil {
		inches := *w.Height * 0.393701
		add(":straight_ruler: Height", fmt.Sprintf("%s cm (%d ft %d in)", num(*w.Height), int(math.Floor(inches/12)), int(math.Floor(math.Mod(inches, 12)))))
	}
	if w.Bust != nil {
		add(":bikini: Bust", num(*w.Bust)+" cm")
	}
	if w.Hip != nil {
		add(":pear: Hip", num(*w.Hip)+" cm")
	}
	if w.Waist != nil {
		add(":jeans: Waist", num(*w.Waist)+" cm")
	}
	if w.Origin != "" {
		add(":earth_americas: Origin", "||"+w.Origin+"||")
	}
	if w.Age != nil {
		add(":calendar_spiral: Age", strconv.Itoa(*w.Age))
	}
	if w.BloodType != "" {
		add(":drop_of_blood: Blood Type", w.BloodType)
	}
	if series, ok := w.FirstSeries(); ok && series.Name != "" {
		add(":book: Series", series.Name)
	}
	for pad := (3 - len(e.Fields)%3) % 3; pad > 0; pad-- {
		add(blank, blank)
	}

	names := make([]string, 0, len(w.Appearances))
	for _, a := range w.Appearances {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	if len(names) > 0 {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: ":camera_with_flash: Appears In", Value: strings.Join(names, ", ")})
	}
	return e
}

func summaryEmbed(s domain.WaifuSummary, ranked *domain.RankedWaifu) *discordgo.MessageEmbed {
	return waifuEmbed(domain.Waifu{WaifuSummary: s, URL: waifuURL(s.Slug)}, ranked)
}

func waifuURL(slug string) string {
	return "https://www.mywaifulist.moe" + mwlWaifuPathPrefix + slug
}

func seriesEmbed(s domain.Series) *discordgo.MessageEmbed {
	e := &discordgo.MessageEmbed{Title: s.Name, URL: s.URL, Color: brandingColor}
	if s.Description != "" {
		e.Description = truncate(s.Description, descriptionLimit)
	}
	if s.PictureURL != "" {
		e.Image = &discordgo.MessageEmbedImage{URL: s.PictureURL}
	}
	return e
}

func num(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func SlugFromEmbeds(embeds []*discordgo.MessageEmbed) (string, bool) {
	for _, e := range embeds {
		if e == nil || e.URL == "" {
			continue
		}
		u, err := url.Parse(e.URL)
		if err != nil {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(u.Hostname()), "mywaifulist.moe") {
			continue
		}
		if !strings.HasPrefix(u.Path, mwlWaifuPathPrefix) {
			continue
		}
		slug := strings.Trim(strings.TrimPrefix(u.Path, mwlWaifuPathPrefix), "/")
		if slug != "" && !strings.Contains(slug, "/") {
			return slug, true
		}
	}
	return "", false
}

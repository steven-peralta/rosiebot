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
	maxAppearances     = 6
	mwlWaifuPathPrefix = "/waifu/"
)

var starColors = map[int]int{
	5: 0xf1c40f,
	4: 0x9b59b6,
	3: 0x3498db,
	2: 0x2ecc71,
	1: 0x95a5a6,
}

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
		text += fmt.Sprintf(" · %dms", elapsed.Milliseconds())
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
	title := w.Name
	if w.NSFW {
		title = "🔞 " + title
	}
	e := &discordgo.MessageEmbed{Title: title, URL: w.URL, Color: cardColor(ranked)}

	if series, ok := w.FirstSeries(); ok && series.Name != "" {
		e.Author = &discordgo.MessageEmbedAuthor{Name: series.Name, URL: series.URL}
	}
	if w.PictureURL != "" {
		e.Image = &discordgo.MessageEmbedImage{URL: w.PictureURL}
	}

	var lines []string
	if names := altNames(w); names != "" {
		lines = append(lines, "*"+names+"*")
	}
	lines = append(lines, statsLine(w, ranked))
	if w.Description != "" {
		lines = append(lines, "", "||"+truncate(strings.TrimSpace(w.Description), descriptionLimit)+"||")
	}
	e.Description = strings.Join(lines, "\n")

	if vitals := vitalsLines(w); len(vitals) > 0 {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Vitals", Value: strings.Join(vitals, "\n"), Inline: true})
	}
	if details := detailLines(w); len(details) > 0 {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Details", Value: strings.Join(details, "\n"), Inline: true})
	}
	if appears := appearancesLine(w); appears != "" {
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Appears in", Value: appears})
	}
	return e
}

func cardColor(ranked *domain.RankedWaifu) int {
	if ranked != nil {
		if c, ok := starColors[ranked.Stars]; ok {
			return c
		}
	}
	return brandingColor
}

func altNames(w domain.Waifu) string {
	var parts []string
	for _, n := range []string{w.OriginalName, w.RomajiName} {
		n = strings.TrimSpace(n)
		if n == "" || strings.EqualFold(n, w.Name) {
			continue
		}
		duplicate := false
		for _, p := range parts {
			if strings.EqualFold(p, n) {
				duplicate = true
			}
		}
		if !duplicate {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, " · ")
}

func statsLine(w domain.Waifu, ranked *domain.RankedWaifu) string {
	rating := "Unranked"
	if ranked != nil && ranked.Stars > 0 {
		rating = strings.Repeat("★", ranked.Stars) + strings.Repeat("☆", domain.MaxStars-ranked.Stars)
		if ranked.Position > 0 {
			rating += fmt.Sprintf(" · #%s", thousands(ranked.Position))
		}
	}
	votes := fmt.Sprintf("❤️ %s · 🗑️ %s", thousands(w.Likes), thousands(w.Trash))
	if total := w.Likes + w.Trash; total > 0 {
		votes += fmt.Sprintf(" · %d%% liked", int(math.Round(100*float64(w.Likes)/float64(total))))
	}
	return rating + "\n" + votes
}

func vitalsLines(w domain.Waifu) []string {
	var lines []string
	if w.Height != nil {
		inches := *w.Height * 0.393701
		lines = append(lines, fmt.Sprintf("Height %s cm (%d′%d″)", num(*w.Height), int(math.Floor(inches/12)), int(math.Floor(math.Mod(inches, 12)))))
	}
	if w.Weight != nil {
		lines = append(lines, fmt.Sprintf("Weight %s kg (%d lb)", num(*w.Weight), int(math.Round(*w.Weight*2.20462))))
	}
	if w.Bust != nil || w.Waist != nil || w.Hip != nil {
		lines = append(lines, "B·W·H "+measure(w.Bust)+"/"+measure(w.Waist)+"/"+measure(w.Hip))
	}
	return lines
}

func detailLines(w domain.Waifu) []string {
	var lines []string
	if w.Age != nil {
		lines = append(lines, "Age "+strconv.Itoa(*w.Age))
	}
	if w.BloodType != "" {
		lines = append(lines, "Blood type "+w.BloodType)
	}
	if w.Origin != "" {
		lines = append(lines, "Origin ||"+w.Origin+"||")
	}
	return lines
}

func appearancesLine(w domain.Waifu) string {
	names := make([]string, 0, len(w.Appearances))
	for _, a := range w.Appearances {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	extra := 0
	if len(names) > maxAppearances {
		extra = len(names) - maxAppearances
		names = names[:maxAppearances]
	}
	line := strings.Join(names, " · ")
	if extra > 0 {
		line += fmt.Sprintf(" · +%d more", extra)
	}
	return line
}

func measure(v *float64) string {
	if v == nil {
		return "?"
	}
	return num(*v)
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

func thousands(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var out strings.Builder
	lead := len(s) % 3
	if lead > 0 {
		out.WriteString(s[:lead])
	}
	for i := lead; i < len(s); i += 3 {
		if out.Len() > 0 {
			out.WriteByte(',')
		}
		out.WriteString(s[i : i+3])
	}
	return out.String()
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

package discord

import (
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"
)

const (
	limitContent     = 2000
	limitEmbeds      = 10
	limitTitle       = 256
	limitDescription = 4096
	limitFields      = 25
	limitFieldName   = 256
	limitFieldValue  = 1024
	limitFooter      = 2048
	limitAuthor      = 256
	limitEmbedTotal  = 6000
)

func runes(s string) int { return utf8.RuneCountInString(s) }

func oversized(content string, embeds []*discordgo.MessageEmbed) bool {
	if runes(content) > limitContent || len(embeds) > limitEmbeds {
		return true
	}
	for _, e := range embeds {
		if e == nil {
			continue
		}
		total := runes(e.Title) + runes(e.Description)
		if runes(e.Title) > limitTitle || runes(e.Description) > limitDescription || len(e.Fields) > limitFields {
			return true
		}
		if e.Footer != nil {
			if runes(e.Footer.Text) > limitFooter {
				return true
			}
			total += runes(e.Footer.Text)
		}
		if e.Author != nil {
			if runes(e.Author.Name) > limitAuthor {
				return true
			}
			total += runes(e.Author.Name)
		}
		for _, f := range e.Fields {
			if f == nil {
				continue
			}
			if runes(f.Name) > limitFieldName || runes(f.Value) > limitFieldValue {
				return true
			}
			total += runes(f.Name) + runes(f.Value)
		}
		if total > limitEmbedTotal {
			return true
		}
	}
	return false
}

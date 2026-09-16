package mwl

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

type pageMeta struct {
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
}

type seriesDTO struct {
	UUID           *string `json:"uuid"`
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	OriginalName   *string `json:"original_name"`
	RomajiName     *string `json:"romaji_name"`
	Description    *string `json:"description"`
	DisplayPicture *string `json:"display_picture"`
	URL            string  `json:"url"`
}

type characterDTO struct {
	UUID           *string     `json:"uuid"`
	Slug           string      `json:"slug"`
	Name           string      `json:"name"`
	OriginalName   *string     `json:"original_name"`
	RomajiName     *string     `json:"romaji_name"`
	DisplayPicture *string     `json:"display_picture"`
	Description    string      `json:"description"`
	Weight         *float64    `json:"weight"`
	Height         *float64    `json:"height"`
	Bust           *float64    `json:"bust"`
	Hip            *float64    `json:"hip"`
	Waist          *float64    `json:"waist"`
	BloodType      *string     `json:"blood_type"`
	Origin         *string     `json:"origin"`
	Age            *int        `json:"age"`
	Likes          int         `json:"likes"`
	Trash          int         `json:"trash"`
	Husbando       bool        `json:"husbando"`
	NSFW           bool        `json:"nsfw"`
	URL            string      `json:"url"`
	Appearances    []seriesDTO `json:"appearances"`
}

type characterEnvelope struct {
	Data characterDTO `json:"data"`
}

type summaryDTO struct {
	UUID           *string `json:"uuid"`
	Slug           string  `json:"slug"`
	Name           string  `json:"name"`
	OriginalName   *string `json:"original_name"`
	RomajiName     *string `json:"romaji_name"`
	DisplayPicture *string `json:"display_picture"`
	Likes          int     `json:"likes"`
	Trash          int     `json:"trash"`
	Rank           *int    `json:"rank"`
}

type summaryListEnvelope struct {
	Data []summaryDTO `json:"data"`
	Meta *pageMeta    `json:"meta"`
}

type seriesListEnvelope struct {
	Data []seriesDTO `json:"data"`
}

type popularEnvelope struct {
	Rows []summaryDTO
	Meta pageMeta
}

func (p *popularEnvelope) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	type indexed struct {
		idx int
		row summaryDTO
	}
	rows := make([]indexed, 0, len(raw))
	for key, val := range raw {
		if key == "meta" {
			if err := json.Unmarshal(val, &p.Meta); err != nil {
				return fmt.Errorf("meta: %w", err)
			}
			continue
		}
		idx, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		var row summaryDTO
		if err := json.Unmarshal(val, &row); err != nil {
			return fmt.Errorf("row %s: %w", key, err)
		}
		rows = append(rows, indexed{idx: idx, row: row})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].idx < rows[j].idx })
	p.Rows = make([]summaryDTO, len(rows))
	for i, r := range rows {
		p.Rows[i] = r.row
	}
	return nil
}

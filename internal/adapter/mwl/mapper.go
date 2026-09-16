package mwl

import (
	"github.com/steven-peralta/rosiebot/internal/adapter/mwl/gen"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nilString(s gen.NilString) string {
	return s.Or("")
}

func summaryFromGen(r gen.AlphaCharacterSummaryResource) domain.WaifuSummary {
	return domain.WaifuSummary{
		Slug:         r.Slug,
		UUID:         nilString(r.UUID),
		Name:         r.Name,
		OriginalName: nilString(r.OriginalName),
		RomajiName:   nilString(r.RomajiName),
		PictureURL:   nilString(r.DisplayPicture),
		Likes:        r.Likes,
		Trash:        r.Trash,
	}
}

func summaryFromDTO(d summaryDTO) domain.WaifuSummary {
	return domain.WaifuSummary{
		Slug:         d.Slug,
		UUID:         deref(d.UUID),
		Name:         d.Name,
		OriginalName: deref(d.OriginalName),
		RomajiName:   deref(d.RomajiName),
		PictureURL:   deref(d.DisplayPicture),
		Likes:        d.Likes,
		Trash:        d.Trash,
	}
}

func summariesFromDTO(rows []summaryDTO) []domain.WaifuSummary {
	out := make([]domain.WaifuSummary, len(rows))
	for i, r := range rows {
		out[i] = summaryFromDTO(r)
	}
	return out
}

func seriesFromDTO(d seriesDTO) domain.Series {
	return domain.Series{
		Slug:        d.Slug,
		UUID:        deref(d.UUID),
		Name:        d.Name,
		URL:         d.URL,
		PictureURL:  deref(d.DisplayPicture),
		Description: deref(d.Description),
	}
}

func seriesListFromDTO(rows []seriesDTO) []domain.Series {
	out := make([]domain.Series, len(rows))
	for i, r := range rows {
		out[i] = seriesFromDTO(r)
	}
	return out
}

func waifuFromDTO(d characterDTO) domain.Waifu {
	return domain.Waifu{
		WaifuSummary: domain.WaifuSummary{
			Slug:         d.Slug,
			UUID:         deref(d.UUID),
			Name:         d.Name,
			OriginalName: deref(d.OriginalName),
			RomajiName:   deref(d.RomajiName),
			PictureURL:   deref(d.DisplayPicture),
			Likes:        d.Likes,
			Trash:        d.Trash,
		},
		URL:         d.URL,
		Description: d.Description,
		Husbando:    d.Husbando,
		NSFW:        d.NSFW,
		Weight:      d.Weight,
		Height:      d.Height,
		Bust:        d.Bust,
		Hip:         d.Hip,
		Waist:       d.Waist,
		BloodType:   deref(d.BloodType),
		Origin:      deref(d.Origin),
		Age:         d.Age,
		Appearances: seriesListFromDTO(d.Appearances),
	}
}

package domain

import "time"

type Series struct {
	Slug        string
	UUID        string
	Name        string
	URL         string
	PictureURL  string
	Description string
}

type WaifuSummary struct {
	Slug         string
	UUID         string
	Name         string
	OriginalName string
	RomajiName   string
	PictureURL   string
	Likes        int
	Trash        int
}

func (s WaifuSummary) TotalVotes() int { return s.Likes + s.Trash }

type Waifu struct {
	WaifuSummary
	URL         string
	Description string
	Husbando    bool
	NSFW        bool
	Weight      *float64
	Height      *float64
	Bust        *float64
	Hip         *float64
	Waist       *float64
	BloodType   string
	Origin      string
	Age         *int
	Appearances []Series
}

func (w Waifu) FirstSeries() (Series, bool) {
	if len(w.Appearances) == 0 {
		return Series{}, false
	}
	return w.Appearances[0], true
}

type OwnedWaifu struct {
	Slug       string
	UUID       string
	Name       string
	PictureURL string
	Likes      int
	Trash      int
	AcquiredAt time.Time
}

func OwnedFromSummary(s WaifuSummary, at time.Time) OwnedWaifu {
	return OwnedWaifu{
		Slug:       s.Slug,
		UUID:       s.UUID,
		Name:       s.Name,
		PictureURL: s.PictureURL,
		Likes:      s.Likes,
		Trash:      s.Trash,
		AcquiredAt: at,
	}
}

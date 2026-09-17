package domain

import "time"

type FavoriteKind string

const (
	FavoriteWaifu  FavoriteKind = "waifu"
	FavoriteSeries FavoriteKind = "series"
)

func (k FavoriteKind) Valid() bool {
	return k == FavoriteWaifu || k == FavoriteSeries
}

type Favorite struct {
	Kind       FavoriteKind
	Slug       string
	Name       string
	URL        string
	PictureURL string
	AddedAt    time.Time
}

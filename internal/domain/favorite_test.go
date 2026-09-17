package domain

import "testing"

func TestFavoriteKind_Valid(t *testing.T) {
	if !FavoriteWaifu.Valid() || !FavoriteSeries.Valid() || FavoriteKind("studio").Valid() || FavoriteKind("").Valid() {
		t.Error("favorite kind validation drifted")
	}
}

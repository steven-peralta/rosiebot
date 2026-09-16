package domain

import (
	"errors"
	"sort"
	"time"
)

const (
	DefaultMinVotes = 100
	MaxStars        = 5
)

var ErrRankingEmpty = errors.New("ranking has no entries")

type IntN interface {
	IntN(n int) int
}

func Score(likes, trash int) float64 {
	return (float64(likes) + 1) / (float64(trash) + 1) * float64(likes+trash)
}

func Stars(position, total int) int {
	if position < 1 || total < 1 || position > total {
		return 0
	}
	share := float64(position) / float64(total)
	switch {
	case share <= 0.01:
		return 5
	case share <= 0.06:
		return 4
	case share <= 0.16:
		return 3
	case share <= 0.26:
		return 2
	default:
		return 1
	}
}

type RankedWaifu struct {
	WaifuSummary
	Position int
	Score    float64
	Stars    int
}

type Ranking struct {
	FetchedAt  time.Time
	CutoffPage int
	rows       []RankedWaifu
	bySlug     map[string]int
}

func BuildRanking(rows []WaifuSummary, minVotes int, fetchedAt time.Time, cutoffPage int) *Ranking {
	seen := make(map[string]struct{}, len(rows))
	ranked := make([]RankedWaifu, 0, len(rows))
	for _, r := range rows {
		if r.TotalVotes() <= minVotes {
			continue
		}
		if _, dup := seen[r.Slug]; dup {
			continue
		}
		seen[r.Slug] = struct{}{}
		ranked = append(ranked, RankedWaifu{WaifuSummary: r, Score: Score(r.Likes, r.Trash)})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score > ranked[j].Score
		}
		return ranked[i].Slug < ranked[j].Slug
	})
	return RankingFromSorted(ranked, fetchedAt, cutoffPage)
}

func RankingFromSorted(sorted []RankedWaifu, fetchedAt time.Time, cutoffPage int) *Ranking {
	rows := make([]RankedWaifu, len(sorted))
	bySlug := make(map[string]int, len(sorted))
	total := len(sorted)
	for i, r := range sorted {
		r.Position = i + 1
		r.Stars = Stars(r.Position, total)
		rows[i] = r
		bySlug[r.Slug] = i
	}
	return &Ranking{FetchedAt: fetchedAt, CutoffPage: cutoffPage, rows: rows, bySlug: bySlug}
}

func (r *Ranking) Len() int {
	if r == nil {
		return 0
	}
	return len(r.rows)
}

func (r *Ranking) Rows() []RankedWaifu {
	if r == nil {
		return nil
	}
	out := make([]RankedWaifu, len(r.rows))
	copy(out, r.rows)
	return out
}

func (r *Ranking) Lookup(slug string) (RankedWaifu, bool) {
	if r == nil {
		return RankedWaifu{}, false
	}
	i, ok := r.bySlug[slug]
	if !ok {
		return RankedWaifu{}, false
	}
	return r.rows[i], true
}

func (r *Ranking) Random(rng IntN) (RankedWaifu, error) {
	return r.Sample(rng, func(RankedWaifu) bool { return true })
}

func (r *Ranking) Sample(rng IntN, keep func(RankedWaifu) bool) (RankedWaifu, error) {
	if r.Len() == 0 {
		return RankedWaifu{}, ErrRankingEmpty
	}
	candidates := make([]int, 0, len(r.rows))
	for i, row := range r.rows {
		if keep(row) {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return RankedWaifu{}, ErrRankingEmpty
	}
	return r.rows[candidates[rng.IntN(len(candidates))]], nil
}

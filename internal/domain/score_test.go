package domain

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

func TestScore_Formula(t *testing.T) {
	cases := []struct {
		likes, trash int
		want         float64
	}{
		{likes: 0, trash: 0, want: 0},
		{likes: 100, trash: 0, want: 101.0 / 1.0 * 100},
		{likes: 99, trash: 1, want: 100.0 / 2.0 * 100},
		{likes: 16199, trash: 3203, want: (16200.0 / 3204.0) * 19402},
	}
	for _, c := range cases {
		got := Score(c.likes, c.trash)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Score(%d,%d) = %v, want %v", c.likes, c.trash, got, c.want)
		}
	}
}

func TestStars_PositionBoundaries(t *testing.T) {
	for _, total := range []int{100, 1000, 10007} {
		t.Run(fmt.Sprintf("total=%d", total), func(t *testing.T) {
			boundary := func(pct float64) int { return int(math.Floor(float64(total) * pct)) }
			check := func(pos, want int) {
				t.Helper()
				if got := Stars(pos, total); got != want {
					t.Errorf("Stars(%d,%d) = %d, want %d", pos, total, got, want)
				}
			}
			check(1, 5)
			check(boundary(0.01), 5)
			check(boundary(0.01)+1, 4)
			check(boundary(0.06), 4)
			check(boundary(0.06)+1, 3)
			check(boundary(0.16), 3)
			check(boundary(0.16)+1, 2)
			check(boundary(0.26), 2)
			check(boundary(0.26)+1, 1)
			check(total, 1)
		})
	}
}

func TestStars_InvalidInputs(t *testing.T) {
	for _, c := range [][2]int{{0, 10}, {11, 10}, {1, 0}, {-1, 5}} {
		if got := Stars(c[0], c[1]); got != 0 {
			t.Errorf("Stars(%d,%d) = %d, want 0", c[0], c[1], got)
		}
	}
}

func TestStars_SmallTotals(t *testing.T) {
	if got := Stars(1, 1); got != 1 {
		t.Errorf("single entry is 100%% of the table and therefore 1 star (v1 getTier), got %d", got)
	}
	if got := Stars(1, 100); got != 5 {
		t.Errorf("first of 100 should be 5 stars, got %d", got)
	}
	if got := Stars(2, 2); got != 1 {
		t.Errorf("last of two should be 1 star, got %d", got)
	}
}

func summary(slug string, likes, trash int) WaifuSummary {
	return WaifuSummary{Slug: slug, Name: slug, Likes: likes, Trash: trash}
}

func TestBuildRanking_FiltersUnderMinAndDedupes(t *testing.T) {
	rows := []WaifuSummary{
		summary("exactly-100", 90, 10),
		summary("low", 1000, 10),
		summary("high", 5000, 100),
		summary("low", 1000, 10),
		summary("mid", 2000, 20),
	}
	fetched := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	r := BuildRanking(rows, DefaultMinVotes, fetched, 42)

	if r.Len() != 3 {
		t.Fatalf("Len = %d, want 3 (100-vote row excluded, duplicate collapsed)", r.Len())
	}
	if r.FetchedAt != fetched || r.CutoffPage != 42 {
		t.Errorf("metadata not preserved: %+v", r)
	}
	got := r.Rows()
	wantOrder := []string{"high", "mid", "low"}
	for i, slug := range wantOrder {
		if got[i].Slug != slug {
			t.Errorf("row %d = %s, want %s", i, got[i].Slug, slug)
		}
		if got[i].Position != i+1 {
			t.Errorf("row %d position = %d", i, got[i].Position)
		}
	}
	if _, ok := r.Lookup("exactly-100"); ok {
		t.Error("row with exactly 100 votes must not be ranked")
	}
	if row, ok := r.Lookup("high"); !ok || row.Position != 1 || row.Score != Score(5000, 100) {
		t.Errorf("Lookup(high) = %+v, %v", row, ok)
	}
	if row, ok := r.Lookup("low"); !ok || row.Position != 3 || row.Stars != 1 {
		t.Errorf("Lookup(low) = %+v, %v", row, ok)
	}
}

func TestBuildRanking_TieBreaksBySlug(t *testing.T) {
	r := BuildRanking([]WaifuSummary{summary("b", 200, 0), summary("a", 200, 0)}, DefaultMinVotes, time.Time{}, 0)
	rows := r.Rows()
	if rows[0].Slug != "a" || rows[1].Slug != "b" {
		t.Errorf("tie order = %s,%s; want a,b", rows[0].Slug, rows[1].Slug)
	}
}

func TestRanking_RowsIsACopy(t *testing.T) {
	r := BuildRanking([]WaifuSummary{summary("a", 200, 0)}, DefaultMinVotes, time.Time{}, 0)
	rows := r.Rows()
	rows[0].Slug = "mutated"
	if row, _ := r.Lookup("a"); row.Slug != "a" {
		t.Error("Rows() must not expose internal storage")
	}
}

func TestRanking_NilSafe(t *testing.T) {
	var r *Ranking
	if r.Len() != 0 || r.Rows() != nil {
		t.Error("nil ranking should be empty")
	}
	if _, ok := r.Lookup("x"); ok {
		t.Error("nil ranking lookup should miss")
	}
	if _, err := r.Random(rand.New(rand.NewPCG(1, 2))); err != ErrRankingEmpty {
		t.Errorf("Random on nil = %v, want ErrRankingEmpty", err)
	}
}

func TestRanking_RandomAndSample(t *testing.T) {
	var rows []WaifuSummary
	for i := range 200 {
		rows = append(rows, summary(fmt.Sprintf("w%03d", i), 10000-i*10, 5))
	}
	r := BuildRanking(rows, DefaultMinVotes, time.Time{}, 0)
	rng := rand.New(rand.NewPCG(7, 11))

	seen := map[string]bool{}
	for range 500 {
		row, err := r.Random(rng)
		if err != nil {
			t.Fatal(err)
		}
		seen[row.Slug] = true
	}
	if len(seen) < 150 {
		t.Errorf("Random should cover the table broadly, saw %d distinct of 200", len(seen))
	}

	for range 200 {
		row, err := r.Sample(rng, func(w RankedWaifu) bool { return w.Stars == 3 })
		if err != nil {
			t.Fatal(err)
		}
		if row.Stars != 3 {
			t.Fatalf("Sample returned stars=%d", row.Stars)
		}
	}

	if _, err := r.Sample(rng, func(RankedWaifu) bool { return false }); err != ErrRankingEmpty {
		t.Errorf("Sample with no candidates = %v, want ErrRankingEmpty", err)
	}
}

func TestRankingFromSorted_AssignsPositionsAndStars(t *testing.T) {
	sorted := make([]RankedWaifu, 100)
	for i := range sorted {
		sorted[i] = RankedWaifu{WaifuSummary: summary(fmt.Sprintf("s%d", i), 1000-i, 1)}
	}
	r := RankingFromSorted(sorted, time.Time{}, 0)
	rows := r.Rows()
	if rows[0].Stars != 5 || rows[5].Stars != 4 || rows[15].Stars != 3 || rows[25].Stars != 2 || rows[26].Stars != 1 {
		t.Errorf("unexpected star assignment: %d %d %d %d %d", rows[0].Stars, rows[5].Stars, rows[15].Stars, rows[25].Stars, rows[26].Stars)
	}
}

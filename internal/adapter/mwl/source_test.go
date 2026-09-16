package mwl

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/steven-peralta/rosiebot/internal/app"
)

func TestSource_Random(t *testing.T) {
	s := newServer(t)
	got, err := newClient(t, s).Random(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "cross-yumikawa-anonymous-code" || got.UUID != "c3418881-6262-4129-904d-7f938b8f33c0" || got.Name != "Cross Yumikawa" || got.OriginalName == "" || got.PictureURL == "" {
		t.Errorf("random = %+v", got)
	}
}

func TestSource_Daily(t *testing.T) {
	s := newServer(t)
	got, err := newClient(t, s).Daily(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "kukuri-yukizome" {
		t.Errorf("daily = %+v", got)
	}
}

func TestSource_GetMapsDetail(t *testing.T) {
	s := newServer(t)
	got, err := newClient(t, s).Get(context.Background(), "rem")
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "rem" || got.Name != "Rem" || got.UUID != "d0ddea7e-4be5-49a5-8f76-ce3136c96cb8" || got.Likes != 8 || got.Trash != 0 {
		t.Errorf("identity = %+v", got.WaifuSummary)
	}
	if got.Weight != nil || got.Height != nil || got.Age != nil || got.BloodType != "" || got.Origin != "" {
		t.Errorf("null measurements should stay unset: %+v", got)
	}
	if got.NSFW || got.Husbando || got.Description == "" || !strings.HasPrefix(got.URL, "https://") {
		t.Errorf("flags = nsfw:%v husbando:%v desc:%q url:%q", got.NSFW, got.Husbando, got.Description, got.URL)
	}
	if len(got.Appearances) != 1 || got.Appearances[0].Slug != "tokyo-xanadu" || got.Appearances[0].Name != "Tokyo Xanadu" {
		t.Errorf("appearances = %+v", got.Appearances)
	}
	if first, ok := got.FirstSeries(); !ok || first.Slug != "tokyo-xanadu" {
		t.Error("FirstSeries should return the only appearance")
	}
}

func TestSource_GetNotFound(t *testing.T) {
	s := newServer(t)
	if _, err := newClient(t, s).Get(context.Background(), "nobody"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestSource_GetEscapesSlug(t *testing.T) {
	s := newServer(t)
	_, _ = newClient(t, s).Get(context.Background(), "a b/c")
	if got := s.last().path; got != "/api/v1/character/a%20b%2Fc" && got != "/api/v1/character/a b/c" {
		t.Errorf("path = %q", got)
	}
}

func TestSource_SearchWaifus(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	got, err := c.SearchWaifus(context.Background(), "rem", 1)
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(fixture(t, "search_waifus.json"), &raw); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != len(raw.Data) || got.Items[0].Slug != "trigun" || got.Items[0].Name != "Rem Saverem" {
		t.Errorf("items = %d first=%+v (series rows must be dropped)", len(got.Items), got.Items[0])
	}
	for _, it := range got.Items {
		if it.Slug == "trigun-series" {
			t.Error("series entry leaked into character results")
		}
	}
	if got.Page != 1 || got.LastPage != 1 {
		t.Errorf("unpaginated search should report a single page: %+v", got)
	}
	if q := s.last().query; !strings.Contains(q, "term=rem") || strings.Contains(q, "page=") {
		t.Errorf("page 1 query = %q", q)
	}

	if _, err := c.SearchWaifus(context.Background(), "rem", 2); err != nil {
		t.Fatal(err)
	}
	if q := s.last().query; !strings.Contains(q, "page=2") {
		t.Errorf("page 2 query = %q", q)
	}
}

func TestSource_ListCharacters(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	p1, err := c.ListCharacters(context.Background(), 1)
	if err != nil || p1.Page != 1 || p1.LastPage != 5113 || len(p1.Items) != 1 || p1.Items[0].Slug != "list-1" {
		t.Fatalf("page 1 = %+v %v", p1, err)
	}
	p3, err := c.ListCharacters(context.Background(), 3)
	if err != nil || p3.Page != 3 || p3.Items[0].Slug != "list-3" {
		t.Fatalf("page 3 = %+v %v", p3, err)
	}
	if !strings.Contains(s.last().query, "page=3") {
		t.Errorf("query = %q", s.last().query)
	}
}

func TestSource_SearchWorks(t *testing.T) {
	s := newServer(t)
	got, err := newClient(t, s).SearchWorks(context.Background(), "re zero")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Slug != "re-zero" || got[0].UUID != "w-1" || got[0].Description != "Subaru suffers" || got[1].UUID != "" {
		t.Errorf("works = %+v", got)
	}
	if !strings.Contains(s.last().query, "term=re+zero") {
		t.Errorf("query = %q", s.last().query)
	}
}

func TestSource_WorkCharactersPaginates(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	p1, err := c.WorkCharacters(context.Background(), "re-zero", 1)
	if err != nil {
		t.Fatal(err)
	}
	if p1.Page != 1 || p1.LastPage != 2 || len(p1.Items) != 1 || p1.Items[0].Slug != "emilia" {
		t.Errorf("page 1 = %+v", p1)
	}
	p2, err := c.WorkCharacters(context.Background(), "re-zero", 2)
	if err != nil {
		t.Fatal(err)
	}
	if p2.Page != 2 || len(p2.Items) != 1 || p2.Items[0].Slug != "rem" || p2.Items[0].UUID != "" {
		t.Errorf("page 2 = %+v", p2)
	}
}

func TestSource_PopularPage(t *testing.T) {
	s := newServer(t)
	c := newClient(t, s)
	got, err := c.PopularPage(context.Background(), 1000)
	if err != nil {
		t.Fatal(err)
	}
	if got.Page != 1000 || got.LastPage != 5113 || len(got.Rows) != 10 {
		t.Fatalf("page = %+v", got)
	}
	if got.Rows[0].Slug != "ceri-sakura-games" || got.Rows[0].Likes != 98 || got.Rows[0].Trash != 3 {
		t.Errorf("first row = %+v", got.Rows[0])
	}
	for _, row := range got.Rows {
		if row.TotalVotes() != 101 {
			t.Errorf("row %s total = %d", row.Slug, row.TotalVotes())
		}
	}

	first, err := c.PopularPage(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Page != 1 || first.LastPage != 5113 || len(first.Rows) != 1 || first.Rows[0].Slug != "top" {
		t.Errorf("page 1 = %+v", first)
	}
	if strings.Contains(s.last().query, "page=") {
		t.Errorf("page 1 should not send a page param: %q", s.last().query)
	}
}

func TestPopularDecoder_ObjectWithNumericKeys(t *testing.T) {
	var env popularEnvelope
	err := json.Unmarshal([]byte(`{"2":{"slug":"c","likes":1,"trash":0},"0":{"slug":"a","likes":3,"trash":0},"1":{"slug":"b","likes":2,"trash":0},"meta":{"current_page":7,"last_page":9},"junk":"ignored"}`), &env)
	if err != nil {
		t.Fatal(err)
	}
	if len(env.Rows) != 3 || env.Rows[0].Slug != "a" || env.Rows[1].Slug != "b" || env.Rows[2].Slug != "c" {
		t.Errorf("rows out of order: %+v", env.Rows)
	}
	if env.Meta.CurrentPage != 7 || env.Meta.LastPage != 9 {
		t.Errorf("meta = %+v", env.Meta)
	}

	if err := json.Unmarshal([]byte(`{"0":"not an object"}`), &env); err == nil {
		t.Error("bad row should fail")
	}
	if err := json.Unmarshal([]byte(`{"meta":"not an object"}`), &env); err == nil {
		t.Error("bad meta should fail")
	}
	if err := json.Unmarshal([]byte(`[]`), &env); err == nil {
		t.Error("array should fail")
	}

	var empty popularEnvelope
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil || len(empty.Rows) != 0 {
		t.Errorf("empty object: %v %+v", err, empty)
	}
}

func TestPopularPage_DefaultsWhenMetaMissing(t *testing.T) {
	var env popularEnvelope
	if err := json.Unmarshal([]byte(`{"0":{"slug":"a","likes":3,"trash":0}}`), &env); err != nil {
		t.Fatal(err)
	}
	if env.Meta.CurrentPage != 0 {
		t.Fatal("precondition")
	}
	page := app.PopularPage{Page: env.Meta.CurrentPage, LastPage: env.Meta.LastPage}
	if page.Page != 0 {
		t.Fatal("precondition")
	}
}

func TestSource_GeneratedClientErrorsWrapped(t *testing.T) {
	s := newServer(t)
	c, err := New(Config{BaseURL: s.URL + "/nope/", APIKey: "k", RequestsPerMinute: 6000})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Random(context.Background()); err == nil || !strings.Contains(err.Error(), "random") {
		t.Errorf("random err = %v", err)
	}
	if _, err := c.Daily(context.Background()); err == nil || !strings.Contains(err.Error(), "daily") {
		t.Errorf("daily err = %v", err)
	}
}

func TestLive_Smoke(t *testing.T) {
	key := os.Getenv("MWL_API_KEY")
	if key == "" {
		t.Skip("MWL_API_KEY not set")
	}
	c, err := New(Config{APIKey: key})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if w, err := c.Random(ctx); err != nil || w.Slug == "" {
		t.Errorf("random: %+v %v", w, err)
	}
	if w, err := c.Daily(ctx); err != nil || w.Slug == "" {
		t.Errorf("daily: %+v %v", w, err)
	}
	if w, err := c.Get(ctx, "rem"); err != nil || w.Slug != "rem" {
		t.Errorf("get: %+v %v", w.WaifuSummary, err)
	}
	if p, err := c.SearchWaifus(ctx, "shinji", 1); err != nil || len(p.Items) == 0 {
		t.Errorf("search should include husbandos: %+v %v", p, err)
	}
	if p, err := c.ListCharacters(ctx, 2); err != nil || len(p.Items) == 0 || p.LastPage < 2 {
		t.Errorf("list characters: %+v %v", p, err)
	}
	if w, err := c.SearchWorks(ctx, "re zero"); err != nil || len(w) == 0 {
		t.Errorf("search works: %+v %v", w, err)
	}
	if p, err := c.PopularPage(ctx, 1000); err != nil || len(p.Rows) == 0 || p.LastPage == 0 {
		t.Errorf("popular: %+v %v", p, err)
	}
	if _, err := c.Get(ctx, "definitely-not-a-real-slug-xyz"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("missing slug err = %v", err)
	}
}

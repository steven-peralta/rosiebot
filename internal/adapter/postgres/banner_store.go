package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/steven-peralta/rosiebot/internal/adapter/postgres/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

type BannerStore struct {
	q *gen.Queries
}

var _ app.BannerStore = (*BannerStore)(nil)

func NewBannerStore(pool *pgxpool.Pool) *BannerStore {
	return &BannerStore{q: gen.New(pool)}
}

type bannerCharacter struct {
	Slug         string `json:"slug"`
	UUID         string `json:"uuid,omitempty"`
	Name         string `json:"name"`
	OriginalName string `json:"original_name,omitempty"`
	RomajiName   string `json:"romaji_name,omitempty"`
	PictureURL   string `json:"picture_url,omitempty"`
	Likes        int    `json:"likes"`
	Trash        int    `json:"trash"`
	Position     int    `json:"position"`
	Stars        int    `json:"stars"`
}

func (s *BannerStore) Get(ctx context.Context, weekStart time.Time) (domain.Banner, error) {
	row, err := s.q.GetBanner(ctx, weekStart.UTC())
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Banner{}, app.ErrNotFound
	}
	if err != nil {
		return domain.Banner{}, fmt.Errorf("postgres: get banner: %w", err)
	}
	return toBanner(row)
}

func (s *BannerStore) Put(ctx context.Context, b domain.Banner) (domain.Banner, error) {
	params, err := bannerParams(b)
	if err != nil {
		return domain.Banner{}, err
	}
	if _, err := s.q.PutBanner(ctx, gen.PutBannerParams(params)); err != nil {
		return domain.Banner{}, fmt.Errorf("postgres: put banner: %w", err)
	}
	return s.Get(ctx, b.WeekStart)
}

func (s *BannerStore) Replace(ctx context.Context, b domain.Banner) error {
	params, err := bannerParams(b)
	if err != nil {
		return err
	}
	if err := s.q.ReplaceBanner(ctx, gen.ReplaceBannerParams(params)); err != nil {
		return fmt.Errorf("postgres: replace banner: %w", err)
	}
	return nil
}

func bannerParams(b domain.Banner) (gen.PutBannerParams, error) {
	chars := make([]bannerCharacter, len(b.Characters))
	for i, c := range b.Characters {
		chars[i] = bannerCharacter{
			Slug: c.Slug, UUID: c.UUID, Name: c.Name, OriginalName: c.OriginalName, RomajiName: c.RomajiName,
			PictureURL: c.PictureURL, Likes: c.Likes, Trash: c.Trash, Position: c.Position, Stars: c.Stars,
		}
	}
	payload, err := json.Marshal(chars)
	if err != nil {
		return gen.PutBannerParams{}, fmt.Errorf("postgres: encode banner: %w", err)
	}
	return gen.PutBannerParams{
		WeekStart:   b.WeekStart.UTC(),
		SeriesSlug:  b.Series.Slug,
		SeriesUuid:  b.Series.UUID,
		SeriesName:  b.Series.Name,
		SeriesUrl:   b.Series.URL,
		PictureUrl:  b.Series.PictureURL,
		Description: b.Series.Description,
		Characters:  payload,
	}, nil
}

func toBanner(row gen.Banner) (domain.Banner, error) {
	var chars []bannerCharacter
	if err := json.Unmarshal(row.Characters, &chars); err != nil {
		return domain.Banner{}, fmt.Errorf("postgres: decode banner characters: %w", err)
	}
	out := make([]domain.RankedWaifu, len(chars))
	for i, c := range chars {
		out[i] = domain.RankedWaifu{
			WaifuSummary: domain.WaifuSummary{Slug: c.Slug, UUID: c.UUID, Name: c.Name, OriginalName: c.OriginalName, RomajiName: c.RomajiName, PictureURL: c.PictureURL, Likes: c.Likes, Trash: c.Trash},
			Position:     c.Position,
			Stars:        c.Stars,
		}
	}
	return domain.Banner{
		WeekStart: row.WeekStart,
		Series: domain.Series{
			Slug: row.SeriesSlug, UUID: row.SeriesUuid, Name: row.SeriesName, URL: row.SeriesUrl,
			PictureURL: row.PictureUrl, Description: row.Description,
		},
		Characters: out,
	}, nil
}

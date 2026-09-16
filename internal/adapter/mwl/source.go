package mwl

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/steven-peralta/rosiebot/internal/adapter/mwl/gen"
	"github.com/steven-peralta/rosiebot/internal/app"
	"github.com/steven-peralta/rosiebot/internal/domain"
)

var _ app.WaifuSource = (*Client)(nil)

func (c *Client) Random(ctx context.Context) (domain.WaifuSummary, error) {
	res, err := c.gen.CharacterRandom(ctx)
	if err != nil {
		return domain.WaifuSummary{}, fmt.Errorf("mwl: random: %w", err)
	}
	return summaryFromGen(res.Data), nil
}

func (c *Client) Daily(ctx context.Context) (domain.WaifuSummary, error) {
	res, err := c.gen.CharacterDaily(ctx)
	if err != nil {
		return domain.WaifuSummary{}, fmt.Errorf("mwl: daily: %w", err)
	}
	switch v := res.(type) {
	case *gen.CharacterDailyOK:
		return summaryFromGen(v.Data), nil
	default:
		return domain.WaifuSummary{}, app.ErrNotFound
	}
}

func (c *Client) Get(ctx context.Context, slug string) (domain.Waifu, error) {
	var env characterEnvelope
	if err := c.getJSON(ctx, "character/"+url.PathEscape(slug), nil, &env); err != nil {
		return domain.Waifu{}, err
	}
	return waifuFromDTO(env.Data), nil
}

func (c *Client) SearchWaifus(ctx context.Context, term string, page int) (app.SearchPage, error) {
	var env summaryListEnvelope
	if err := c.getJSON(ctx, "search/waifus", pageQuery(page, url.Values{"term": {term}}), &env); err != nil {
		return app.SearchPage{}, err
	}
	return searchPage(env, page), nil
}

func (c *Client) SearchWorks(ctx context.Context, term string) ([]domain.Series, error) {
	var env seriesListEnvelope
	if err := c.getJSON(ctx, "search/works", url.Values{"term": {term}}, &env); err != nil {
		return nil, err
	}
	return seriesListFromDTO(env.Data), nil
}

func (c *Client) WorkCharacters(ctx context.Context, slug string, page int) (app.SearchPage, error) {
	var env summaryListEnvelope
	if err := c.getJSON(ctx, "work/"+url.PathEscape(slug)+"/characters", pageQuery(page, nil), &env); err != nil {
		return app.SearchPage{}, err
	}
	return searchPage(env, page), nil
}

func (c *Client) PopularPage(ctx context.Context, page int) (app.PopularPage, error) {
	var env popularEnvelope
	if err := c.getJSON(WithBackground(ctx), "ranking/popular", pageQuery(page, nil), &env); err != nil {
		return app.PopularPage{}, err
	}
	current := env.Meta.CurrentPage
	if current == 0 {
		current = page
	}
	last := env.Meta.LastPage
	if last == 0 {
		last = current
	}
	return app.PopularPage{Rows: summariesFromDTO(env.Rows), Page: current, LastPage: last}, nil
}

func pageQuery(page int, q url.Values) url.Values {
	if q == nil {
		q = url.Values{}
	}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	return q
}

func searchPage(env summaryListEnvelope, requested int) app.SearchPage {
	out := app.SearchPage{Items: summariesFromDTO(env.Data), Page: requested, LastPage: requested}
	if env.Meta != nil {
		if env.Meta.CurrentPage > 0 {
			out.Page = env.Meta.CurrentPage
		}
		if env.Meta.LastPage > 0 {
			out.LastPage = env.Meta.LastPage
		}
	}
	return out
}

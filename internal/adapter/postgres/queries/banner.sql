-- name: GetBanner :one
SELECT * FROM banners WHERE week_start = @week_start;

-- name: PutBanner :execrows
INSERT INTO banners (week_start, series_slug, series_uuid, series_name, series_url, picture_url, description, characters)
VALUES (@week_start, @series_slug, @series_uuid, @series_name, @series_url, @picture_url, @description, @characters)
ON CONFLICT (week_start) DO NOTHING;

-- name: ReplaceBanner :exec
INSERT INTO banners (week_start, series_slug, series_uuid, series_name, series_url, picture_url, description, characters)
VALUES (@week_start, @series_slug, @series_uuid, @series_name, @series_url, @picture_url, @description, @characters)
ON CONFLICT (week_start) DO UPDATE SET
    series_slug = EXCLUDED.series_slug,
    series_uuid = EXCLUDED.series_uuid,
    series_name = EXCLUDED.series_name,
    series_url  = EXCLUDED.series_url,
    picture_url = EXCLUDED.picture_url,
    description = EXCLUDED.description,
    characters  = EXCLUDED.characters,
    created_at  = now();

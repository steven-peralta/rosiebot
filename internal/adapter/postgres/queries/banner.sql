-- name: GetBanner :one
SELECT * FROM banners WHERE week_start = @week_start;

-- name: PutBanner :execrows
INSERT INTO banners (week_start, series_slug, series_uuid, series_name, series_url, picture_url, description, characters)
VALUES (@week_start, @series_slug, @series_uuid, @series_name, @series_url, @picture_url, @description, @characters)
ON CONFLICT (week_start) DO NOTHING;

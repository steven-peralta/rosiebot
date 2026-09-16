-- name: GetDailyWaifu :one
SELECT * FROM daily_waifu WHERE day = @day;

-- name: PutDailyWaifu :execrows
INSERT INTO daily_waifu (day, slug, uuid, name, original_name, romaji_name, picture_url, likes, trash)
VALUES (@day, @slug, @uuid, @name, @original_name, @romaji_name, @picture_url, @likes, @trash)
ON CONFLICT (day) DO NOTHING;

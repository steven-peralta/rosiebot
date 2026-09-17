-- name: GetDailyWaifu :one
SELECT * FROM daily_waifu WHERE day = @day;

-- name: PutDailyWaifu :execrows
INSERT INTO daily_waifu (day, slug, uuid, name, original_name, romaji_name, picture_url, likes, trash)
VALUES (@day, @slug, @uuid, @name, @original_name, @romaji_name, @picture_url, @likes, @trash)
ON CONFLICT (day) DO NOTHING;

-- name: ReplaceDailyWaifu :exec
INSERT INTO daily_waifu (day, slug, uuid, name, original_name, romaji_name, picture_url, likes, trash)
VALUES (@day, @slug, @uuid, @name, @original_name, @romaji_name, @picture_url, @likes, @trash)
ON CONFLICT (day) DO UPDATE SET
    slug          = EXCLUDED.slug,
    uuid          = EXCLUDED.uuid,
    name          = EXCLUDED.name,
    original_name = EXCLUDED.original_name,
    romaji_name   = EXCLUDED.romaji_name,
    picture_url   = EXCLUDED.picture_url,
    likes         = EXCLUDED.likes,
    trash         = EXCLUDED.trash,
    created_at    = now();

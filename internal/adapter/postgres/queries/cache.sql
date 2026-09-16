-- name: GetCachedWaifu :one
SELECT payload, fetched_at FROM waifu_cache WHERE slug = @slug;

-- name: TouchCachedWaifu :exec
UPDATE waifu_cache SET last_read_at = @now
WHERE slug = @slug AND last_read_at < @stale_before;

-- name: PutCachedWaifu :exec
INSERT INTO waifu_cache (slug, payload, fetched_at, last_read_at)
VALUES (@slug, @payload, @fetched_at, @fetched_at)
ON CONFLICT (slug) DO UPDATE SET payload = EXCLUDED.payload, fetched_at = EXCLUDED.fetched_at, last_read_at = EXCLUDED.last_read_at;

-- name: GetCachedPage :one
SELECT payload, fetched_at FROM page_cache WHERE key = @key;

-- name: PutCachedPage :exec
INSERT INTO page_cache (key, payload, fetched_at)
VALUES (@key, @payload, @fetched_at)
ON CONFLICT (key) DO UPDATE SET payload = EXCLUDED.payload, fetched_at = EXCLUDED.fetched_at;

-- name: PruneCachedWaifus :execrows
DELETE FROM waifu_cache WHERE last_read_at < @unread_since;

-- name: PruneCachedPages :execrows
DELETE FROM page_cache WHERE fetched_at < @unread_since;

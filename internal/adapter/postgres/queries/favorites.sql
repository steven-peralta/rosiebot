-- name: AddFavorite :execrows
INSERT INTO favorites (guild_id, user_id, kind, slug, name, url, picture_url, added_at)
VALUES (@guild_id, @user_id, @kind, @slug, @name, @url, @picture_url, @added_at)
ON CONFLICT (guild_id, user_id, kind, slug) DO NOTHING;

-- name: RemoveFavorite :execrows
DELETE FROM favorites WHERE guild_id = @guild_id AND user_id = @user_id AND kind = @kind AND slug = @slug;

-- name: ListFavorites :many
SELECT * FROM favorites WHERE guild_id = @guild_id AND user_id = @user_id AND kind = @kind ORDER BY added_at, slug;

-- name: FindFavorites :many
SELECT * FROM favorites
WHERE kind = @kind AND slug = ANY(@slugs::text[]) AND (@guild_id::text = '' OR guild_id = @guild_id::text)
ORDER BY user_id, guild_id, slug;

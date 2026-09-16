-- name: AddOwned :execrows
INSERT INTO inventory (guild_id, user_id, slug, uuid, name, picture_url, likes, trash, acquired_at)
VALUES (@guild_id, @user_id, @slug, @uuid, @name, @picture_url, @likes, @trash, @acquired_at)
ON CONFLICT DO NOTHING;

-- name: GetOwned :one
SELECT * FROM inventory
WHERE guild_id = @guild_id AND user_id = @user_id AND slug = @slug;

-- name: SellOwned :one
WITH removed AS (
    DELETE FROM inventory
    WHERE inventory.guild_id = @guild_id AND inventory.user_id = @user_id AND inventory.slug = @slug
    RETURNING inventory.slug
)
UPDATE players
SET coins = players.coins + @price, updated_at = @now
WHERE players.guild_id = @guild_id AND players.user_id = @user_id AND EXISTS (SELECT 1 FROM removed)
RETURNING players.coins;

-- name: OwnedSlugs :many
SELECT slug FROM inventory
WHERE guild_id = @guild_id AND user_id = @user_id AND slug = ANY(@slugs::text[])
ORDER BY slug;

-- name: TransferOwned :execrows
UPDATE inventory
SET user_id = @to_user_id
WHERE guild_id = @guild_id AND user_id = @from_user_id AND slug = ANY(@slugs::text[]);

-- name: ListOwned :many
SELECT * FROM inventory
WHERE guild_id = @guild_id AND user_id = @user_id
  AND (@prefix::text = '' OR lower(name) LIKE lower(@prefix::text) || '%')
ORDER BY acquired_at, slug
LIMIT NULLIF(@row_limit::int, 0);

-- name: CountOwned :one
SELECT count(*) FROM inventory
WHERE guild_id = @guild_id AND user_id = @user_id;

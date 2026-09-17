-- name: RecordRoll :exec
INSERT INTO rolls (guild_id, user_id, slug, name, kind, cost, rolled_at)
VALUES (@guild_id, @user_id, @slug, @name, @kind, @cost, @rolled_at);

-- name: RecentRolls :many
SELECT * FROM rolls
WHERE guild_id = @guild_id AND user_id = @user_id
ORDER BY rolled_at DESC, id DESC
LIMIT @row_limit;

-- name: CountRolls :one
SELECT count(*) FROM rolls WHERE guild_id = @guild_id AND user_id = @user_id;

-- name: GuildPlayers :many
SELECT * FROM players WHERE guild_id = @guild_id ORDER BY user_id;

-- name: GuildInventory :many
SELECT user_id, slug FROM inventory WHERE guild_id = @guild_id ORDER BY user_id, slug;

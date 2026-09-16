-- name: EnsurePlayer :one
INSERT INTO players (guild_id, user_id, coins, created_at, updated_at)
VALUES (@guild_id, @user_id, @coins, @now, @now)
ON CONFLICT (guild_id, user_id) DO UPDATE SET guild_id = EXCLUDED.guild_id
RETURNING *;

-- name: GetPlayer :one
SELECT * FROM players
WHERE guild_id = @guild_id AND user_id = @user_id;

-- name: LockPlayers :many
SELECT * FROM players
WHERE guild_id = @guild_id AND user_id = ANY(@user_ids::text[])
ORDER BY user_id
FOR UPDATE;

-- name: DebitCoins :one
UPDATE players
SET coins = coins - @amount, updated_at = @now
WHERE guild_id = @guild_id AND user_id = @user_id AND coins >= @amount
RETURNING coins;

-- name: ClaimDaily :one
UPDATE players
SET coins = coins + @amount, daily_claimed_at = @now, updated_at = @now
WHERE guild_id = @guild_id AND user_id = @user_id
  AND (daily_claimed_at IS NULL OR daily_claimed_at < @window_start)
RETURNING coins;

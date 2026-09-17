-- name: GetAlertEnabled :one
SELECT enabled FROM alert_settings WHERE guild_id = @guild_id AND user_id = @user_id;

-- name: SetAlertEnabled :exec
INSERT INTO alert_settings (guild_id, user_id, enabled, updated_at)
VALUES (@guild_id, @user_id, @enabled, @updated_at)
ON CONFLICT (guild_id, user_id) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at;

-- name: IsDMClosed :one
SELECT EXISTS (SELECT 1 FROM alert_dm_closed WHERE user_id = @user_id);

-- name: MarkDMClosed :exec
INSERT INTO alert_dm_closed (user_id, closed_at) VALUES (@user_id, @closed_at)
ON CONFLICT (user_id) DO UPDATE SET closed_at = EXCLUDED.closed_at;

-- name: ClearDMClosed :exec
DELETE FROM alert_dm_closed WHERE user_id = @user_id;

-- name: MarkAlertSent :execrows
INSERT INTO alerts_sent (event_id, user_id, sent_at) VALUES (@event_id, @user_id, @sent_at)
ON CONFLICT (event_id, user_id) DO NOTHING;

-- name: PruneAlertsSent :execrows
DELETE FROM alerts_sent WHERE sent_at < @before;

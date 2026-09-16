-- name: InsertRankingSnapshot :one
INSERT INTO ranking_snapshots (fetched_at, cutoff_page, row_count, complete)
VALUES (@fetched_at, @cutoff_page, @row_count, false)
RETURNING id;

-- name: InsertRankingRows :copyfrom
INSERT INTO ranking_rows (snapshot_id, position, slug, uuid, name, original_name, romaji_name, picture_url, likes, trash, score, stars)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: CompleteRankingSnapshot :exec
UPDATE ranking_snapshots SET complete = true WHERE id = @id;

-- name: LatestCompleteSnapshot :one
SELECT * FROM ranking_snapshots
WHERE complete
ORDER BY fetched_at DESC, id DESC
LIMIT 1;

-- name: RankingRows :many
SELECT * FROM ranking_rows
WHERE snapshot_id = @snapshot_id
ORDER BY position;

-- name: PruneRankingSnapshots :exec
DELETE FROM ranking_snapshots
WHERE id NOT IN (
    SELECT id FROM ranking_snapshots
    WHERE complete
    ORDER BY fetched_at DESC, id DESC
    LIMIT @keep
);

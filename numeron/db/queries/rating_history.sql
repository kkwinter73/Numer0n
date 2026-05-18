-- ============================================================
-- rating_history: レーティング変動履歴
-- ============================================================

-- name: InsertRatingHistory :one
-- 試合終了時に各プレイヤーの履歴を記録。
-- users.UpdateUserRating と同じトランザクション内で呼び出す想定。
INSERT INTO rating_history (
    user_id, match_id,
    rating_before, deviation_before, volatility_before,
    rating_after,  deviation_after,  volatility_after,
    result,
    opponent_rating_before, opponent_deviation_before
) VALUES (
    $1, $2,
    $3, $4, $5,
    $6, $7, $8,
    $9,
    $10, $11
)
RETURNING *;

-- name: ListUserRatingHistory :many
-- ユーザーのレーティング推移 (新しい順)。グラフ表示用。
SELECT * FROM rating_history
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMatchRatingHistory :many
-- 1試合の両プレイヤーのレーティング変動 (試合詳細画面用)
SELECT * FROM rating_history
WHERE match_id = $1
ORDER BY created_at ASC;

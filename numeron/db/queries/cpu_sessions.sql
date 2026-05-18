-- ============================================================
-- cpu_sessions / cpu_session_turns: CPU対戦
-- ============================================================

-- name: CreateCPUSession :one
INSERT INTO cpu_sessions (
    id, user_id, player_secret, cpu_secret, cpu_candidates
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetCPUSessionByID :one
SELECT * FROM cpu_sessions WHERE id = $1;

-- name: UpdateCPUSessionAfterTurn :one
-- 1ターン進行後の状態を更新。終了判定もこのクエリで一発。
-- ended_at は status が変わった瞬間にだけセット (DBレベルで CASE)
UPDATE cpu_sessions
SET
    turn            = $2,
    status          = $3,
    cpu_candidates  = $4,
    updated_at      = NOW(),
    ended_at        = CASE WHEN $3::text != 'playing' AND ended_at IS NULL THEN NOW() ELSE ended_at END
WHERE id = $1
RETURNING *;

-- name: InsertCPUSessionTurn :one
-- ターン記録を追加。
INSERT INTO cpu_session_turns (
    session_id, turn,
    player_guess, player_eat, player_bite,
    cpu_guess,    cpu_eat,    cpu_bite
) VALUES (
    $1, $2,
    $3, $4, $5,
    $6, $7, $8
)
RETURNING *;

-- name: ListCPUSessionTurns :many
-- セッションの全ターン (時系列順)
SELECT * FROM cpu_session_turns
WHERE session_id = $1
ORDER BY turn ASC;

-- name: ListUserCPUSessions :many
-- ユーザーのCPU対戦履歴 (新しい順)
SELECT id, status, turn, created_at, ended_at
FROM cpu_sessions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteOldCPUSessions :exec
-- GC: 終了済みで30日以上更新のないセッションを削除。
-- CASCADE で cpu_session_turns も自動削除される。
DELETE FROM cpu_sessions
WHERE status != 'playing'
  AND updated_at < NOW() - INTERVAL '30 days';

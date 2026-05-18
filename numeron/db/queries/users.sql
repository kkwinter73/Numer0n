-- ============================================================
-- users: 認証・プロフィール・レーティング操作
-- ============================================================

-- name: CreateUser :one
-- 新規ユーザー登録。レーティングは Glicko-2 初期値 (1500, 350, 0.06) でDB側のDEFAULTから入る。
INSERT INTO users (
    id, username, email, password_hash, display_name
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByUsername :one
-- 大文字小文字を区別しない検索 (idx_users_username_lower を活用)
SELECT * FROM users
WHERE LOWER(username) = LOWER($1) AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL;

-- name: UpdateUserProfile :one
-- 表示名のみ変更可。username/email は別 endpoint で慎重に変える想定 (現状は実装しない)
UPDATE users
SET display_name = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2, updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateUserRating :one
-- 試合終了時にレーティング3要素を更新 + 統計をインクリメント。
-- result は 'win' | 'loss' | 'draw'。トランザクション内で rating_history への INSERT と組み合わせて使う想定。
UPDATE users
SET
    rating            = $2,
    rating_deviation  = $3,
    rating_volatility = $4,
    games_played      = games_played + 1,
    games_won         = games_won  + CASE WHEN @result::text = 'win'  THEN 1 ELSE 0 END,
    games_lost        = games_lost + CASE WHEN @result::text = 'loss' THEN 1 ELSE 0 END,
    games_drawn       = games_drawn + CASE WHEN @result::text = 'draw' THEN 1 ELSE 0 END,
    updated_at        = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteUser :exec
-- ソフトデリート (deleted_at をセット)。
-- 履歴は残るが、authログイン・ランキング等からは除外される。
UPDATE users
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListTopRatedUsers :many
-- レーティング上位N人。ランキング画面用。
-- 試合数が極端に少ない人を除外したい場合は WHERE games_played >= ? を加える。
SELECT id, username, display_name, rating, rating_deviation, games_played, games_won
FROM users
WHERE deleted_at IS NULL
ORDER BY rating DESC, id ASC
LIMIT $1 OFFSET $2;

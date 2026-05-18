-- ============================================================
-- online_matches / online_match_turns: オンライン対戦
-- ============================================================

-- name: CreateOnlineMatch :one
-- ホストが新規ルームを作成した時点。phase='lobby' で開始。
INSERT INTO online_matches (
    id, code, host_user_id, host_name, host_token
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: JoinOnlineMatch :one
-- ゲストが参加。phase を 'setup' に進める。
UPDATE online_matches
SET
    guest_user_id    = $2,
    guest_name       = $3,
    guest_token      = $4,
    rated            = (host_user_id IS NOT NULL AND $2 IS NOT NULL),
    phase            = 'setup',
    last_activity_at = NOW()
WHERE id = $1 AND phase = 'lobby' AND guest_user_id IS NULL AND guest_token IS NULL
RETURNING *;

-- name: GetOnlineMatchByID :one
SELECT * FROM online_matches WHERE id = $1;

-- name: GetActiveOnlineMatchByCode :one
-- コードからアクティブなルームを検索。code is NULL (=終了済み) は除外される。
SELECT * FROM online_matches WHERE code = $1;

-- name: SetSecret :one
-- 自分の暗証を設定。slot は 0 (host) or 1 (guest)。
-- token と slot の組み合わせが正しいことは usecase 側で事前検証する想定。
UPDATE online_matches
SET
    host_secret      = CASE WHEN @slot::int = 0 THEN @secret::text ELSE host_secret END,
    guest_secret     = CASE WHEN @slot::int = 1 THEN @secret::text ELSE guest_secret END,
    -- 両者設定完了したら phase='play' と started_at をセット
    phase            = CASE
                          WHEN (CASE WHEN @slot::int = 0 THEN @secret::text ELSE host_secret END) IS NOT NULL
                               AND
                               (CASE WHEN @slot::int = 1 THEN @secret::text ELSE guest_secret END) IS NOT NULL
                          THEN 'play'
                          ELSE phase
                       END,
    started_at       = CASE
                          WHEN started_at IS NULL
                               AND (CASE WHEN @slot::int = 0 THEN @secret::text ELSE host_secret END) IS NOT NULL
                               AND (CASE WHEN @slot::int = 1 THEN @secret::text ELSE guest_secret END) IS NOT NULL
                          THEN NOW()
                          ELSE started_at
                       END,
    last_activity_at = NOW()
WHERE id = $1 AND phase = 'setup'
RETURNING *;

-- name: AdvanceTurn :one
-- ターン進行後の状態更新。両者の予想が揃ったタイミングで usecase が呼ぶ。
-- end_status を NULL でなくセットした場合は phase='ended' に。
UPDATE online_matches
SET
    turn             = $2,
    phase            = $3,
    end_status       = $4,
    winner_user_id   = $5,
    ended_at         = CASE WHEN $3::text = 'ended' AND ended_at IS NULL THEN NOW() ELSE ended_at END,
    -- 試合終了時、code を NULL にして再利用可能にする
    code             = CASE WHEN $3::text = 'ended' THEN NULL ELSE code END,
    last_activity_at = NOW()
WHERE id = $1
RETURNING *;

-- name: TouchOnlineMatch :exec
-- アクティビティの更新 (poll や guess の都度)。GC判定用 last_activity_at を更新。
UPDATE online_matches
SET last_activity_at = NOW()
WHERE id = $1 AND phase != 'ended';

-- name: InsertOnlineMatchTurn :one
INSERT INTO online_match_turns (
    match_id, turn,
    host_guess,  host_eat,  host_bite,
    guest_guess, guest_eat, guest_bite
) VALUES (
    $1, $2,
    $3, $4, $5,
    $6, $7, $8
)
RETURNING *;

-- name: ListOnlineMatchTurns :many
SELECT * FROM online_match_turns
WHERE match_id = $1
ORDER BY turn ASC;

-- name: ListUserOnlineMatches :many
-- ユーザーが host or guest として参加した試合の履歴。
-- レーティング戦のみ表示するなら WHERE rated = TRUE を加える。
SELECT * FROM online_matches
WHERE host_user_id = $1 OR guest_user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteIdleOnlineMatches :exec
-- GC: 30分以上アクティビティのない未終了ルームを削除。
-- 終了済みルームは履歴として残すので消さない。
DELETE FROM online_matches
WHERE phase != 'ended'
  AND last_activity_at < NOW() - INTERVAL '30 minutes';

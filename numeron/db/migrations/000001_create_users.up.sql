-- フェーズ2.3: users テーブル
--
-- 認証情報、プロフィール、現在のレーティングを保持します。
-- レーティングの履歴は別途 rating_history テーブルで管理します。
--
-- 設計判断:
--   - id は UUID v7 (時刻順ソート可能、推測されない、分散DB対応)
--     PostgreSQL 17 には組み込み uuidv7() 関数があるが、まだ widespread でないため
--     アプリ側で生成して INSERT する想定 (Go の uuid ライブラリ or 自前実装)
--   - パスワードは argon2id でハッシュ化したものを保存
--   - 削除はソフトデリート (deleted_at IS NOT NULL なら削除済み)
--   - Glicko-2 レーティング: R=1500, RD=350, σ=0.06 が新規プレイヤーの初期値

CREATE TABLE users (
    id                  UUID            PRIMARY KEY,
    username            TEXT            NOT NULL,
    email               TEXT            NOT NULL,
    password_hash       TEXT            NOT NULL,
    display_name        TEXT            NOT NULL,

    -- Glicko-2 レーティング (現在値)
    rating              INTEGER         NOT NULL DEFAULT 1500,
    rating_deviation    REAL            NOT NULL DEFAULT 350.0,
    rating_volatility   REAL            NOT NULL DEFAULT 0.06,

    -- 統計 (集計クエリを高速化するための非正規化)
    games_played        INTEGER         NOT NULL DEFAULT 0,
    games_won           INTEGER         NOT NULL DEFAULT 0,
    games_lost          INTEGER         NOT NULL DEFAULT 0,
    games_drawn         INTEGER         NOT NULL DEFAULT 0,

    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

-- 認証用の検索高速化 + 重複防止
-- username は大文字小文字を区別しない一意 (例: "Alice" と "alice" は同一視)
CREATE UNIQUE INDEX idx_users_username_lower ON users (LOWER(username)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_email_lower    ON users (LOWER(email))    WHERE deleted_at IS NULL;

-- レーティングランキング用
CREATE INDEX idx_users_rating_desc ON users (rating DESC) WHERE deleted_at IS NULL;

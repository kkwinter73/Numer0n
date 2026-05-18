-- フェーズ2.3: オンライン対戦の試合 + ターン記録
--
-- 設計判断:
--   - code はアクティブ中のみ一意。試合終了後に NULL 化することで code を再利用可能に
--   - host_user_id / guest_user_id は NULL 許容 (未ログイン参加対応)
--   - rated フラグで「レーティング戦か」を区別。両者ログイン済みのみ true
--   - 暗証は平文で保存 (理由: 3桁720通りで暗号化に意味がない、ゲーム終了後は両者に開示済)

CREATE TABLE online_matches (
    id                  UUID            PRIMARY KEY,

    -- ルームコード (アクティブ中のみ。終了後はNULLに更新する)
    -- 部分ユニーク制約: NULL 同士は衝突しないので、複数の終了済み試合が共存できる
    code                VARCHAR(6),

    -- プレイヤー情報 (slot 0 = host, slot 1 = guest)
    host_user_id        UUID            REFERENCES users(id) ON DELETE SET NULL,
    guest_user_id       UUID            REFERENCES users(id) ON DELETE SET NULL,
    host_name           TEXT            NOT NULL,
    guest_name          TEXT,
    host_token          TEXT            NOT NULL,
    guest_token         TEXT,

    -- 暗証 (設定後にセット、終了後もそのまま残す)
    host_secret         VARCHAR(3),
    guest_secret        VARCHAR(3),

    -- 進行状態
    phase               TEXT            NOT NULL DEFAULT 'lobby'
                          CHECK (phase IN ('lobby', 'setup', 'play', 'ended')),
    end_status          TEXT
                          CHECK (end_status IS NULL OR end_status IN ('p0_win', 'p1_win', 'draw')),
    -- 勝者 (引き分けなら NULL、終了していなければ NULL)
    winner_user_id      UUID            REFERENCES users(id) ON DELETE SET NULL,
    turn                INTEGER         NOT NULL DEFAULT 1,

    -- レーティング戦か (両者ログイン済みのみ true)
    rated               BOOLEAN         NOT NULL DEFAULT FALSE,

    -- 時刻
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    started_at          TIMESTAMPTZ,         -- 両者暗証設定完了時に記録
    ended_at            TIMESTAMPTZ,         -- 試合終了時に記録
    last_activity_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW()  -- GC判定用
);

-- アクティブなコードは一意 (部分インデックス)
CREATE UNIQUE INDEX idx_online_matches_code_active ON online_matches (code) WHERE code IS NOT NULL;

-- ユーザー別の試合履歴取得用 (host/guest 両方向)
CREATE INDEX idx_online_matches_host  ON online_matches (host_user_id, created_at DESC)
    WHERE host_user_id IS NOT NULL;
CREATE INDEX idx_online_matches_guest ON online_matches (guest_user_id, created_at DESC)
    WHERE guest_user_id IS NOT NULL;

-- GC判定用 (idle なルームを定期削除)
CREATE INDEX idx_online_matches_active_idle ON online_matches (last_activity_at)
    WHERE phase != 'ended';

-- ランキング/勝率集計の補助
CREATE INDEX idx_online_matches_winner ON online_matches (winner_user_id, ended_at DESC)
    WHERE winner_user_id IS NOT NULL AND rated = TRUE;

-- ----------------------------------------------------------------
-- オンライン対戦の各ターン記録
-- ----------------------------------------------------------------
CREATE TABLE online_match_turns (
    id                  BIGSERIAL       PRIMARY KEY,
    match_id            UUID            NOT NULL REFERENCES online_matches(id) ON DELETE CASCADE,
    turn                INTEGER         NOT NULL,

    -- host 側
    host_guess          VARCHAR(3)      NOT NULL,
    host_eat            SMALLINT        NOT NULL,
    host_bite           SMALLINT        NOT NULL,

    -- guest 側
    guest_guess         VARCHAR(3)      NOT NULL,
    guest_eat           SMALLINT        NOT NULL,
    guest_bite          SMALLINT        NOT NULL,

    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT host_eatbite_valid  CHECK (host_eat + host_bite <= 3   AND (host_eat  != 3 OR host_bite  = 0)),
    CONSTRAINT guest_eatbite_valid CHECK (guest_eat + guest_bite <= 3 AND (guest_eat != 3 OR guest_bite = 0)),
    CONSTRAINT online_match_turn_unique UNIQUE (match_id, turn)
);

CREATE INDEX idx_online_match_turns_match ON online_match_turns (match_id, turn);

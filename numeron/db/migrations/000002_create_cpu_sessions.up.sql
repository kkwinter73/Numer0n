-- フェーズ2.3: CPU対戦セッション + ターンログ
--
-- 現状はメモリ実装 (persistence.MemorySessionStore) で管理しています。
-- フェーズ2.4 でこのテーブルを使う実装に差し替えます。
--
-- 設計判断:
--   - user_id は NULL 許容 (未ログインプレイ対応)
--   - cpu_candidates は JSONB で保持。720要素の配列が徐々に絞り込まれる
--   - 終了済みセッションも一定期間保持 (履歴・統計用)。古いものは別途 GC

CREATE TABLE cpu_sessions (
    id                  UUID            PRIMARY KEY,
    user_id             UUID            REFERENCES users(id) ON DELETE SET NULL,

    -- 暗証番号 (3桁・重複なし)
    player_secret       VARCHAR(3)      NOT NULL,
    cpu_secret          VARCHAR(3)      NOT NULL,

    turn                INTEGER         NOT NULL DEFAULT 1,
    status              TEXT            NOT NULL DEFAULT 'playing'
                          CHECK (status IN ('playing', 'player_win', 'cpu_win', 'draw')),

    -- CPUが推論に使う候補リスト (JSONB配列)
    -- 例: [[1,2,3], [4,5,6], ...] (絞り込まれていく)
    cpu_candidates      JSONB           NOT NULL,

    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    ended_at            TIMESTAMPTZ
);

-- ユーザー別の履歴取得用
CREATE INDEX idx_cpu_sessions_user_id ON cpu_sessions (user_id, created_at DESC)
    WHERE user_id IS NOT NULL;

-- 古いセッションのGC用 (進行中のものは長く残らない想定)
CREATE INDEX idx_cpu_sessions_status_updated ON cpu_sessions (status, updated_at);

-- ----------------------------------------------------------------
-- CPU セッションの各ターン記録 (履歴・統計用)
-- ----------------------------------------------------------------
CREATE TABLE cpu_session_turns (
    id                  BIGSERIAL       PRIMARY KEY,
    session_id          UUID            NOT NULL REFERENCES cpu_sessions(id) ON DELETE CASCADE,
    turn                INTEGER         NOT NULL,

    -- プレイヤー側の予想と結果
    player_guess        VARCHAR(3)      NOT NULL,
    player_eat          SMALLINT        NOT NULL,
    player_bite         SMALLINT        NOT NULL,

    -- CPU側の予想と結果
    cpu_guess           VARCHAR(3)      NOT NULL,
    cpu_eat             SMALLINT        NOT NULL,
    cpu_bite            SMALLINT        NOT NULL,

    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    -- 不変条件: eat + bite <= 3, eat == 3 ⇒ bite == 0
    CONSTRAINT player_eatbite_valid CHECK (player_eat + player_bite <= 3 AND (player_eat != 3 OR player_bite = 0)),
    CONSTRAINT cpu_eatbite_valid    CHECK (cpu_eat + cpu_bite <= 3       AND (cpu_eat    != 3 OR cpu_bite    = 0)),
    -- 同一セッション内でターン番号は一意
    CONSTRAINT cpu_session_turn_unique UNIQUE (session_id, turn)
);

-- セッション別のターン取得用 (順序付き)
CREATE INDEX idx_cpu_session_turns_session ON cpu_session_turns (session_id, turn);

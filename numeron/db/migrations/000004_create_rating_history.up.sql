-- フェーズ2.3: レーティング変動履歴
--
-- 試合ごとのレーティング変動を記録します。
-- 用途:
--   - ユーザーのレーティング推移グラフ表示
--   - 「この試合でレーティングどれくらい動いた?」を表示
--   - レーティング計算のロジック検証 (再現可能)

CREATE TABLE rating_history (
    id                  BIGSERIAL       PRIMARY KEY,
    user_id             UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    match_id            UUID            NOT NULL REFERENCES online_matches(id) ON DELETE CASCADE,

    -- 試合前のレーティング (Glicko-2 の3つの値)
    rating_before       INTEGER         NOT NULL,
    deviation_before    REAL            NOT NULL,
    volatility_before   REAL            NOT NULL,

    -- 試合後のレーティング
    rating_after        INTEGER         NOT NULL,
    deviation_after     REAL            NOT NULL,
    volatility_after    REAL            NOT NULL,

    -- 試合結果 (ユーザーの視点で)
    result              TEXT            NOT NULL CHECK (result IN ('win', 'loss', 'draw')),

    -- 相手のレーティング (計算の検証用)
    opponent_rating_before    INTEGER   NOT NULL,
    opponent_deviation_before REAL      NOT NULL,

    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ユーザー別の履歴表示用 (時系列順)
CREATE INDEX idx_rating_history_user ON rating_history (user_id, created_at DESC);

-- 試合別の参照用 (両プレイヤーの記録が確認できる)
CREATE INDEX idx_rating_history_match ON rating_history (match_id);

-- 1試合 × 1ユーザー で1レコード (重複INSERTを防ぐ)
CREATE UNIQUE INDEX idx_rating_history_unique ON rating_history (user_id, match_id);

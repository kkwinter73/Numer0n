# データベーススキーマ

## ER図 (テキスト表現)

```
┌─────────────────────────────────────┐
│ users                               │
├─────────────────────────────────────┤
│ id (PK, UUID v7)                    │
│ username (UNIQUE, lowercased)       │
│ email (UNIQUE, lowercased)          │
│ password_hash (argon2id)            │
│ display_name                        │
│ rating, rating_deviation,           │
│ rating_volatility (Glicko-2)        │
│ games_played, games_won/lost/drawn  │
│ created_at, updated_at, deleted_at  │
└────────────┬────────────────────────┘
             │
             │ 1
             │
        ┌────┴────┬────────────┐
        │ 0..n    │ 0..n       │ 0..n
        ▼         ▼            ▼
┌──────────────┐ ┌────────────────────────┐ ┌──────────────────┐
│ cpu_sessions │ │ online_matches         │ │ rating_history   │
├──────────────┤ ├────────────────────────┤ ├──────────────────┤
│ id (PK)      │ │ id (PK)                │ │ id (PK)          │
│ user_id (FK) │ │ code (UNIQ アクティブ中)│ │ user_id (FK)     │
│ player_secret│ │ host_user_id (FK)      │ │ match_id (FK)    │
│ cpu_secret   │ │ guest_user_id (FK)     │ │ rating_before/   │
│ status       │ │ host_name, guest_name  │ │   _after         │
│ turn         │ │ host_token, guest_token│ │ deviation_before/│
│ cpu_candidates│ │ host_secret,           │ │   _after         │
│ created_at,  │ │   guest_secret         │ │ volatility_*     │
│ ended_at     │ │ phase, end_status      │ │ result           │
└────────┬─────┘ │ winner_user_id (FK)    │ │ opponent_*_before│
         │ 1    │ turn, rated            │ │ created_at       │
         │      │ created_at, started_at,│ └──────────────────┘
         │ 0..n │   ended_at,            │
         ▼      │   last_activity_at     │
┌──────────────┐└──────┬─────────────────┘
│ cpu_session_ │       │ 1
│ turns        │       │
├──────────────┤       │ 0..n
│ id (BIGSER)  │       ▼
│ session_id   │ ┌──────────────────────┐
│ turn         │ │ online_match_turns   │
│ player_guess │ ├──────────────────────┤
│ player_eat/  │ │ id (BIGSERIAL)       │
│   _bite      │ │ match_id (FK)        │
│ cpu_guess    │ │ turn                 │
│ cpu_eat/_bite│ │ host_guess, host_*   │
│ created_at   │ │ guest_guess, guest_* │
│ UNIQUE       │ │ created_at           │
│ (session_id, │ │ UNIQUE (match_id,    │
│  turn)       │ │         turn)        │
└──────────────┘ └──────────────────────┘
```

## テーブル責務マトリクス

| テーブル | 役割 | 主なクエリ |
|---|---|---|
| `users` | 認証 + プロフィール + 現在レーティング | ログイン認証、ランキング表示 |
| `cpu_sessions` | CPU対戦の進行状態 | 進行中セッションの取得・更新 |
| `cpu_session_turns` | CPU対戦の各ターン記録 | 履歴表示 (replay) |
| `online_matches` | オンライン対戦の試合 | アクティブルーム検索、履歴 |
| `online_match_turns` | オンライン対戦の各ターン記録 | 試合詳細・統計分析 |
| `rating_history` | レーティング変動の履歴 | 推移グラフ、変動量表示 |

## 重要なインデックス

| インデックス | 目的 | クエリ例 |
|---|---|---|
| `idx_users_username_lower` | 大小無視ログイン認証 | `WHERE LOWER(username) = LOWER(?)` |
| `idx_users_rating_desc` | ランキング表示 | `ORDER BY rating DESC LIMIT 100` |
| `idx_online_matches_code_active` | ルームコード検索 (部分UNIQ) | `WHERE code = ?` |
| `idx_online_matches_active_idle` | GC対象ルーム検索 (部分) | `WHERE phase != 'ended' AND last_activity_at < ?` |
| `idx_rating_history_user` | ユーザーのレーティング推移 | `WHERE user_id = ? ORDER BY created_at DESC` |

## トランザクション境界 (重要)

### レーティング更新 (試合終了時)
複数の書き込みを **1つのトランザクション**にまとめる必要があります:

```sql
BEGIN;
  -- 各プレイヤーのレーティング更新
  UPDATE users SET rating=..., games_played=... WHERE id = host_id;
  UPDATE users SET rating=..., games_played=... WHERE id = guest_id;
  -- 履歴記録
  INSERT INTO rating_history (...) VALUES (host_*);
  INSERT INTO rating_history (...) VALUES (guest_*);
  -- 試合状態
  UPDATE online_matches SET phase='ended', ended_at=NOW(), winner_user_id=...;
COMMIT;
```

途中で失敗すれば全部巻き戻る。これは sqlc から直接書けないので、
usecase 層で `*sql.Tx` を使って実装します(フェーズ2.4)。

### オンライン対戦のターン解決
両者の guess が揃った瞬間:

```sql
BEGIN;
  INSERT INTO online_match_turns (...);
  UPDATE online_matches SET turn=..., phase=..., end_status=...;
COMMIT;
```

## 設計判断の記録

### なぜ UUID v7 か
- 推測されない (`/users/1` → `/users/2` で他人を覗けない)
- 時刻ソート可能 (BIGSERIAL に近い特性)
- 分散DBへの拡張余地

### なぜ部分インデックス (`WHERE deleted_at IS NULL`) を多用するか
- 論理削除されたユーザーは認証・ランキング対象外
- 全行をスキャンする代わりにアクティブユーザーだけを高速検索
- PostgreSQL ならではの最適化

### なぜ `online_matches` の暗証は平文で保存するか
- 3桁の数字は720通りしかない (暗号化しても秒で総当たり可能)
- ゲーム終了後は両者に開示される情報なので機密性ゼロ
- 履歴を見返す要件で必要

### なぜ統計カラム (games_played 等) を `users` に持つか
- 集計クエリ (COUNT(*)) を毎回走らせると重い
- レーティング更新のトランザクションと一緒に書けばコストほぼゼロ
- 非正規化のトレードオフを意識的に選択

# NUMERON — 開発ロードマップ

このドキュメントは、Numeronプロジェクトの開発を**複数セッションにまたがって継続する**ための作業記録です。
新しいセッションを始めたら、まずこのファイルを読んでください。

---

## 🎯 プロジェクト目標

- **ユーザー2万人規模のWebサービス**として運用可能なNumeron対戦サイト
- **学習目的** (Go + モダンスタック) **かつ本気運用**を両立
- 拡張性 (ランキング、ログイン、マッチング等の機能追加)
- 保守性・可読性 (1人で開発を続けられる構造)

## 🏗 技術スタック (確定)

| カテゴリ | 採用技術 | 理由 |
|---|---|---|
| バックエンド言語 | Go 1.22+ | 要件 |
| HTTPルータ | `net/http` (標準) → 必要に応じて `chi` | 標準で十分。複雑化したら chi |
| データベース | PostgreSQL | リレーショナル要件 (ユーザー、戦績、ランキング) |
| ORM | sqlc (SQL記述+型生成) | 型安全、SQL力が付く、生成コードが素直 |
| マイグレーション | golang-migrate | デファクト |
| キャッシュ/セッション | Redis | セッション + ランキング Sorted Set |
| リアルタイム通信 | WebSocket (`coder/websocket`) | long-poll より効率的、機能拡張しやすい |
| 認証 | セッションCookie + bcrypt | Webサービス標準、JWTより安全 |
| レーティング | Glicko-2 | ELOより精緻、現代のオンラインゲーム標準 |
| ログ | `log/slog` (標準) | Go 1.21+ の構造化ログ標準 |
| 設定管理 | `envconfig` | 環境変数の構造体マッピング |
| フロント言語 | TypeScript | 型安全、API契約 |
| フロントFW | React + Vite | エコシステム最大、求人最多 |
| サーバ状態管理 | TanStack Query | キャッシュ/再フェッチ自動化 |
| クライアント状態 | Zustand | 軽量、Redux回避 |
| インフラ (開発) | Docker Compose | DB+Redis を一発起動 |
| インフラ (本番) | Cloudflare Pages (front) + Fly.io等 (Go API) | 検討中 |

## 📐 アーキテクチャ

レイヤード+ヘキサゴナル風 (DDD軽量版):

```
cmd/server/main.go                エントリーポイント (依存組み立て)
    │
    ├─ internal/adapter/          外界とのIO層
    │   ├─ httphandler/           HTTPプロトコル変換
    │   └─ persistence/           ストレージ実装 (現在メモリ、将来DB)
    │
    ├─ internal/usecase/          アプリケーション固有のロジック
    │
    ├─ internal/port/             インターフェース定義 (Repository等)
    │
    └─ internal/domain/           純粋なドメインモデル (依存なし)
```

**依存方向**: `cmd` → `adapter` → `usecase` → `port` → `domain`
(下位層は上位層を一切知らない)

---

## 📅 フェーズ計画

各フェーズはさらに「1セッションに収まる単位」に分割しています。
✅ = 完了、🚧 = 作業中、⬜ = 未着手

### フェーズ1: Goアーキテクチャ整理 (DB無し) ✅ 完了
- ✅ **1.1** ディレクトリ構造再編 + DEVELOPMENT.md作成
- ✅ **1.2** domain層のユニットテスト (カバレッジ 99.4%, テスト数 60件, `-race` クリーン)
- ✅ **1.3** port (interface) 定義 + handler の依存をインターフェースに切替
- ✅ **1.4** usecase 層実装 + 業務エラー型定義 + handler を薄く
- ✅ **1.5** 構造化エラー (`{error: {code, message, details}}`) + フロント対応
- ✅ **1.6** slog 構造化ログ + リクエストID + ミドルウェア (RequestID/Logger/AccessLog/Recover)
- ✅ **1.7** 統合テスト (httphandler 95.0%, 全体 121件パス)

### フェーズ2: DB + 設定管理 + フロント基盤
- ✅ **2.1** envconfig による設定管理 + Docker Compose (PostgreSQL + Redis) + ヘルスチェック
- ✅ **2.2** sqlc + golang-migrate セットアップ + DB接続ヘルパー + DBHealthChecker
- ✅ **2.3** スキーマ設計 (users, cpu_sessions, online_matches, rating_history)
- ⬜ **2.4** リポジトリのDB実装 + testcontainers
- ⬜ **2.5** React + Vite + TS の足場構築 (既存フロントはそのまま)

### フェーズ3: WebSocket化 + 切断ハンドリング
- ⬜ **3.1** WebSocket hub の設計と実装 (`coder/websocket`)
- ⬜ **3.2** 既存 long-poll を WS に置き換え
- ⬜ **3.3** 切断検知 + 再接続 + 対戦放棄判定
- ⬜ **3.4** 既存フロントを WS クライアントに更新

### フェーズ4: 認証 + ユーザー機能
- ⬜ **4.1** ユーザー登録/ログインAPI (bcrypt)
- ⬜ **4.2** セッション管理 (Redis)
- ⬜ **4.3** 認証ミドルウェア + WS 認証
- ⬜ **4.4** React でログイン/登録画面
- ⬜ **4.5** 既存対戦をユーザーIDに紐付け
- ⬜ **4.6** レート制限 (ログインAPI 優先)

### フェーズ5: マッチング + Glicko-2レーティング + ランキング
- ⬜ **5.1** ランダムマッチング (待機キュー)
- ⬜ **5.2** Glicko-2レーティング実装
- ⬜ **5.3** ランキング集計 (Redis Sorted Set + 定期バッチ)
- ⬜ **5.4** React でマッチング/ランキング/マイページ画面

### フェーズ6: 既存画面のReact移行 + UI仕上げ
- ⬜ **6.1** ゲーム画面・対戦画面の React 化
- ⬜ **6.2** デザインシステムのコンポーネント整理
- ⬜ **6.3** アクセシビリティ対応

### フェーズ7: 本番化
- ⬜ **7.1** Prometheus メトリクス + Grafana
- ⬜ **7.2** Sentry エラートラッキング
- ⬜ **7.3** Dockerfile (マルチステージ) + デプロイ
- ⬜ **7.4** GitHub Actions (テスト + ビルド + デプロイ)
- ⬜ **7.5** 負荷試験 (k6) と最適化

---

## 🧭 現在地

**フェーズ 2.3 完了**: 本番スキーマ設計と sqlc クエリ定義。

### 2.3 で追加したもの
- マイグレーション 4本 (users / cpu_sessions / online_matches / rating_history)
- sqlc クエリ定義 30件 (db/queries/*.sql)
- `db/SCHEMA.md`: ER図、テーブル責務、トランザクション境界、設計判断の記録

### 採用した設計
- **ID**: UUID v7 (時刻ソート可能、推測されない)
- **パスワード**: argon2id (現代標準のハッシュ)
- **暗証保存**: 平文 (3桁数字は暗号化に意味がないため)
- **試合履歴**: 全ターン記録 (replay/分析可能)
- **レーティング**: users 列に現在値、rating_history で履歴
- **削除**: users はソフトデリート、対戦履歴は物理保持
- **未ログインプレイ**: user_id NULL 許容
- **rated戦**: 両者ログイン時のみ true (レーティングが動くのはこの試合のみ)

### 設計の妥協と注意点
- pgx の副作用 import がコメントアウト状態 (環境制約)
  → お手元で `go get` 後にコメント解除
- sqlc generate を実行していないので `internal/adapter/persistence/sqlc/` は空
  → お手元で `sqlc generate` 実行で Goコード生成
- SQL構文は厳密検証していない (PostgreSQL を起動できない環境のため)
  → お手元で `migrate up` 時にエラーが出たら修正

次のセッションでは **フェーズ 2.4** (リポジトリのDB実装) に進みます。

---

## 📝 設計判断の記録

なぜこの選択をしたか、後から見ても分かるように記録します。

### WebSocket と認証の順序 (WS→認証 を採用)
WS化は通信基盤の根本変更。先に認証を long-poll の上に作ると、WS切り替え時に書き直しになる。
WS基盤を固めてから、その上に認証 (接続時にCookie/トークン検証) を被せるほうが、変更が小さい。
**ただし注意**: フェーズ3完了時点ではログイン無しで誰でも対戦できる状態。本番公開はフェーズ4完了後。

### フロントReact化を後ろに置いた理由
ログイン/ランキング画面が増える前に React 化したいところだが、丸ごとReact化は重い。
妥協案として **フェーズ2.5 で React+Vite+TS の足場だけ作り、ログイン以降は React で書く**。
既存画面のReact移行はフェーズ6で実施。

### Glicko-2 を選んだ理由
ELO より精緻 (レーティング不確実性を追跡)、Lichess・スマブラSP等の現代のオンラインゲームで標準。
Go ライブラリあり (`github.com/zelenin/go-glicko2` 等)。

### sqlc を選んだ理由
ORM (GORM等) より型安全で、SQLそのものを書くため SQL力が付く。
生成コードが素直で、複雑なクエリで詰まない。

### モジュール名 `github.com/numeron/numeron`
将来GitHubに公開する前提で正規形式にしておく。実際のリポジトリ名は変える可能性あり。

### テスト方針 (フェーズ1.2で確立)
- **テーブル駆動テスト**を基本とする (Goの慣習)
- **t.Run でサブテスト化**して失敗時の特定を容易に
- **不変条件の明示**: 例えば `CheckEatBite` の結果 (eat+bite≤3, eat==3⇒bite==0) を毎テストで検証
- **時間に依存するテスト**: WaitEvents のような並行テストは閾値を緩く設定 (`< 100ms` 等)
- **競合検出**: `go test -race` を CI で常時実行
- **AddPlayer の二重チェック** のような防御的コードは「いま到達不能でも将来のため残す」場合あり。
  その意図はコメントで明示し、カバレッジ100%を強要しない。

### インターフェース設計 (フェーズ1.3で確立)
- **consumer-driven**: 使う側 (handler / usecase) のニーズでメソッドを決める。「全部入り」インターフェースは作らない
- **将来を見越したシグネチャ**: メモリ実装で `error` が不要でも、DB実装で必要になるなら最初から含める。
  シグネチャ変更による広範囲な書き換えを避ける
- **コンパイル時アサーション**: `var _ port.X = (*Y)(nil)` で実装漏れを早期検出
- **(nil, false, nil) 規約**: 「見つからない」と「I/Oエラー」は別物として表現
  - `Get(id) (*T, bool, error)` の bool は「見つかったかどうか」
  - `Get(id) (*T, error)` の error 単独だと「見つからない」と「DB断」が混ざる

### usecase 層の責務 (フェーズ1.4で確立)
- **入力の正規化と検証**: ParseSecret、name のトリム等
- **複数 domain 操作の組み合わせ**: 「Get → 状態確認 → 計算 → Save」のような業務フロー
- **HTTPを知らない**: req/res オブジェクトを受け取らない。生の値 (string, int) を受け取る
- **業務エラーは型で表現**:
  - `var ErrXxx = errors.New("...")` をエクスポート
  - `fmt.Errorf("%w: 詳細", ErrXxx)` で詳細を付加して wrap
  - handler 側で `errors.Is(err, usecase.ErrXxx)` でディスパッチ
- **DTOで戻す**: ドメインモデルそのままだとフィールドが多すぎる場合は usecase 固有の DTO を返す
  (例: `CreateRoomResult`, `JoinRoomResult`)
- **将来の入口を想定**: 同じ usecase を HTTP / WebSocket / CLI から呼べる構造にする

### エラーレスポンス設計 (フェーズ1.5で確立)
- **形式**: `{"error":{"code":"...","message":"...","details":{...}}}`
  - エラー時のJSONルートは error フィールドのみ。成功時とのスキーマ衝突を避ける
- **code は SCREAMING_SNAKE_CASE**: `INVALID_INPUT`, `ROOM_NOT_FOUND` 等
  - クライアントとサーバーの「契約」。変更は破壊的変更扱い
- **message は人間向け**: 現状は日本語。将来 i18n するならフロント側で code → 翻訳辞書
- **details は任意**: フィールド別検証エラー等の補足。`omitempty` で省略可
- **エラーコード一覧**: `httphandler/api_error.go` に定数として集約
- **`fmt.Errorf("%w: <詳細>", baseErr)` パターンの活用**:
  - usecase は `ErrXxx` を返す + 日本語の詳細を wrap
  - handler は `errors.Is` で型判定 + `unwrapUserMessage` で詳細を抽出
  - これにより「エラー型による分岐」と「人間向けメッセージ」を同時に実現
- **フロント側**: `ApiError` クラスで code/message/details/httpStatus をプロパティ化

### ログ・観測性 (フェーズ1.6で確立)
- **`log/slog` のみ使用**: 標準ライブラリで完結。zerolog/zap 等の外部依存なし
- **2形式切り替え**: 開発時 text、本番 json。`LOG_FORMAT` 環境変数で切替
- **ログレベル**: `LOG_LEVEL` 環境変数で `debug`/`info`/`warn`/`error` を切替
- **usecase / domain はログを書かない**: エラーを返すだけ。ログは middleware / handler で集約
  - これにより usecase テストでロガーをモックする必要がない
  - 「ログ責務」をひとつの層に集約することで、出力先変更やフォーマット変更が局所化
- **リクエストID**: 16桁hex (8バイト乱数)。UUIDより短くログで読みやすい
  - context.Context で全層に伝播
  - レスポンスヘッダ `X-Request-ID` でエコー
  - クライアント提供IDを尊重 (分散トレース対応)
- **ミドルウェア順**: `Recover → RequestID → Logger → AccessLog → handler`
  - Recover を最外側にし、他のミドルウェア内 panic も吸収
- **静的ファイルアクセスはログ抑制**: API以外のパスはアクセスログから除外 (ノイズ削減)

### テスト戦略 (フェーズ1.7で確立)
- **3階層のテスト**:
  1. **ユニットテスト** (domain層): 純粋関数を網羅 → 99%カバレッジ
  2. **ユニットテスト + モック** (usecase層): port をモックで差し替え → 91%
  3. **統合テスト** (handler層): `httptest.NewServer` で本物のサーバー起動 → 95%
- **TestMain で slog を静音化**: 統合テストで `request rejected` のINFOログが標準出力を埋めるのを抑制
- **`t.Cleanup` でリソース解放**: テスト終了時に自動 `srv.Close()` でリーク防止
- **`net/http/httptest` のみ使用**: 外部テストライブラリ不要 (testify等は導入しない)
- **時間に依存するテスト**: poll の起床テストは閾値 < 2秒で判定 (テストごとに環境差を吸収)
- **persistence 層は フェーズ2 で testcontainers + 統合テスト**: メモリ実装のテストは今は不要

### 設定管理 (フェーズ2.1で確立)
- **12-factor App 原則**: 設定は環境変数経由のみ。コードや設定ファイルにハードコードしない
- **自前で軽量実装**: `envconfig` 等の外部ライブラリ不要。`os.Getenv` + 型変換 + バリデーションを config パッケージに集約
- **検証は起動時に**: 不正な値 (例: `PORT=abc`) なら main() が即時 fail。`os.Stderr` に明確な理由を書いて `os.Exit(1)`
- **デフォルト値を必ず用意**: `getEnv(key, default)` パターンで「環境変数なし」でも開発時は動く
- **.env は自動読み込みしない**: シンプルさ重視。ユーザーは `env $(cat .env | xargs) go run ./cmd/server` 等で渡す
  - 別案: `direnv` を使えば自動。本プロジェクトは direnv 推奨だが必須にしない

### ヘルスチェック設計 (フェーズ2.1で確立)
- **`HealthChecker` インターフェース**: `Name()` と `Check(ctx) error` の2メソッド
  - DB/Redis/外部API等、依存先ごとに別の実装
  - フェーズ2.2 で `DBHealthChecker` を追加していく
- **タイムアウト**: 各チェッカーごとに 2秒のコンテキストタイムアウト
  - ヘルスチェック自体がハングするのを防ぐ
- **ステータスコード**: 全依存OK → 200, 1つでも fail → 503
  - ロードバランサ (k8s liveness probe等) で自動切り離しに使える
- **失敗の詳細はレスポンスに含める**: `{name: "...", status: "fail", error: "..."}` で運用者が原因特定可能

### DB アクセス設計 (フェーズ2.2で確立)
- **sqlc + golang-migrate**: 「SQLは手書き、Goコードは自動生成」の方針
  - ORM (GORM等) より型安全で、SQL力が付く
  - 複雑なクエリで詰まない
- **pgx/v5 をドライバに採用**: PostgreSQL 向けの高性能ドライバ、sqlcの推奨
- **データベース層は段階的に有効化**: `DATABASE_URL` が空ならメモリ実装で起動
  - フェーズ2.4まではメモリ実装が現役、DBはフェーズ2.4で差し替え
  - これによりDBが使えない環境でも開発・テストできる
- **起動時 Ping リトライ**: 5回・指数バックオフで Docker Compose 起動順の差を吸収
- **接続プール**: `DB_MAX_OPEN_CONNS=25`, `IDLE=5` がデフォルト。本番では負荷に応じて調整
- **マイグレーション**: 番号付きで管理 (`000001_xxx.up.sql`)。`up` と `down` を必ず両方書く
- **生成コードはコミット**: `sqlc generate` の結果はリポジトリに含める (再現性のため)
- **クエリは `db/queries/*.sql` に集約**: `-- name: GetXxx :one` のようなアノテーションでメソッド生成
- **循環依存回避**: persistence パッケージが httphandler に直接依存しないよう、
  `HealthChecker` インターフェースを構造的型付けで満たす形にする

### スキーマ設計 (フェーズ2.3で確立)
- **ID は UUID v7**: 推測されない + 時刻ソート可能。BIGSERIAL より広い適用範囲
- **大小無視のUNIQUE**: `LOWER(username)` の関数インデックスで実現。
  `Alice` と `alice` を同一視できる
- **論理削除 (users のみ)**: `deleted_at IS NULL` の部分インデックスでアクティブユーザーを高速検索
- **対戦履歴は物理削除しない**: 他ユーザーの履歴に紐づくため
- **部分UNIQUEインデックス**: `code` のように「アクティブ中のみ一意」を表現
  - `WHERE code IS NOT NULL` でNULL同士の衝突を回避、コードの再利用が可能
- **CHECK制約で不変条件を表現**: 例えばEAT/BITE の `eat + bite <= 3` を DB レベルで保証
- **トランザクション境界**: レーティング更新(users 2件 + rating_history 2件 + online_matches)は
  1トランザクション。sqlc から直接書けないので usecase 層で `*sql.Tx` を渡す
- **非正規化**: `users.games_played` 等の統計列は集計クエリ高速化のため意図的に冗長保持
- **JSONB の活用**: `cpu_sessions.cpu_candidates` は構造化された配列なのでJSONB で十分

---

## 🚀 環境セットアップ

```bash
# Go 1.22 以上が必要
go version

# 依存取得 (現状は標準ライブラリのみ)
cd numeron
go mod tidy

# 起動
go run ./cmd/server
# → http://localhost:8080
```

---

## 🔄 次セッションへの引き継ぎ

### 次にやること: フェーズ2.4 (リポジトリのDB実装 + testcontainers)

**目的**: メモリストア (`MemorySessionStore` / `MemoryRoomStore`) を **PostgreSQL 実装に差し替え**。
これでメモリ揮発性のデータが永続化される。フェーズ1で確立した port インターフェース経由なので、上位層 (usecase / handler) は無変更で済むはず。

**2.4 で具体的にやる作業**:

1. **port インターフェース拡張**:
   - `SessionRepository` に `Get` がエラー戻り値しかない → DB I/O 失敗を表現できているか確認
   - 必要に応じて context.Context を受け取るようシグネチャ変更
     (DB操作にはcontext必須)
2. **DB実装**:
   - `internal/adapter/persistence/postgres_session_store.go`
   - `internal/adapter/persistence/postgres_room_store.go`
   - sqlc 生成コードを呼んでドメインモデルに詰め替える
3. **ドメインモデル ⇔ DB行 のマッピング**:
   - `domain.Session` ↔ sqlc の `CPUSession` 構造体
   - `domain.Room` ↔ sqlc の `OnlineMatch` 構造体
   - 注意: `domain.Room` の sync.Mutex / sync.Cond は **DBには保存しない**。
     これらは「同一プロセス内の現在ルームの状態管理」に必要なので、
     DB保存とは別問題 (フェーズ3 で WS hub に逃がす計画)
4. **トランザクション**:
   - `*sql.Tx` を usecase に渡す or リポジトリ内で使う設計
   - レーティング更新は1トランザクション
5. **testcontainers**:
   - `internal/adapter/persistence/postgres_*_test.go`
   - 各テストの先頭で PostgreSQL コンテナ起動
   - マイグレーション適用 → クエリ実行 → 検証
6. **段階的切替**:
   - `DATABASE_URL` 空ならメモリ実装 (現状維持)
   - 設定されていればDB実装を使う

**ポイント**:
- フェーズ1.7 の handler 統合テストが通れば、リファクタリングは安全
- `domain.Room` の sync 機構の扱いが微妙: DB に保存するのはステート(phase, turn等)だけで、
  待機中の poll を起こす仕組みはプロセスローカル
- メモリ実装は一旦残す: テスト時にDB起動するのは重いので、ユニットテストはメモリで継続
- UUID 生成: `github.com/google/uuid` を導入。GoogleのライブラリでBSD-3でOK
- お手元での実行が前提: `sqlc generate` 後でないとビルドできない

### 開始時のコマンド

```bash
cd numeron
go get github.com/google/uuid
sqlc generate
docker compose up -d
migrate -path db/migrations -database "$DATABASE_URL" up
go test ./... -race
```

### セッション継続のための合言葉

「フェーズ2.4をやろう」で再開できます。

---

## 📂 現状のファイル構成

```
numeron/
├── DEVELOPMENT.md                    ← このファイル
├── CLAUDE.md                         Claude Code 用指示書
├── MIGRATE_TO_CLAUDE_CODE.md         Claude Code 移行手順
├── README.md
├── go.mod
├── docker-compose.yml                PostgreSQL + Redis (開発用)
├── sqlc.yaml                         sqlc コード生成設定
├── .env.example
├── .gitignore
├── db/
│   ├── SCHEMA.md                     ER図・テーブル責務・設計判断の記録
│   ├── migrations/                   golang-migrate 用
│   │   ├── 000001_create_users.{up,down}.sql
│   │   ├── 000002_create_cpu_sessions.{up,down}.sql
│   │   ├── 000003_create_online_matches.{up,down}.sql
│   │   └── 000004_create_rating_history.{up,down}.sql
│   └── queries/                      sqlc 用クエリ定義 (計30件)
│       ├── users.sql
│       ├── cpu_sessions.sql
│       ├── online_matches.sql
│       └── rating_history.sql
├── cmd/
│   └── server/main.go
├── internal/
│   ├── config/
│   ├── domain/
│   ├── usecase/
│   ├── observability/
│   ├── port/
│   └── adapter/
│       ├── httphandler/
│       └── persistence/
│           ├── session_store.go       メモリ実装
│           ├── room_store.go          メモリ実装
│           ├── db.go                  DB接続ヘルパー
│           ├── db_health.go           DBヘルスチェッカー
│           ├── db_test.go
│           └── sqlc/                  sqlc 生成コード (要 `sqlc generate`)
│               └── README.md
└── web/static/index.html
```

---

## 🐛 既知の負債 (今後解消予定)

- `domain/room.go` に sync 機構が同居 → フェーズ3で WS hub に移す
- `TurnLog` のフィールド名 `player_*`/`cpu_*` がオンライン対戦で不自然 → フロント書き換えと同時にリネーム検討
- `usecase.SubmitSecret` / `SubmitGuess` で domain エラーを文字列で判定している → 後続で domain 層のエラー型化検討
- persistence 層のDBテストが無い → フェーズ2.4 で testcontainers で書く
- フロントは `console.log` を使っていない (将来 sentry等を入れる場合に検討)
- メモリストア (RoomStore) のGCループに停止手段がない → context.Context 受け取りに変える
- `db.go` の pgx import がコメントアウト状態 → お手元で `go get` 実行後に有効化
- `sqlc generate` を実行していない → お手元で実行が必要
- domain モデルと DB スキーマの**マッピング**がまだ無い → フェーズ2.4 で実装
- UUID生成ライブラリの選定がまだ → フェーズ2.4 で `github.com/google/uuid` 等を導入

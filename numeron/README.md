# NUMER0N

3桁の暗証番号を当て合う対戦ゲーム「Numeron」のWebサービス実装。

CPU対戦とオンライン2人対戦に対応しています。

## クイックスタート (DB なし)

```bash
# Go 1.22 以上
go run ./cmd/server
```

→ ブラウザで http://localhost:8080 を開く。CPU対戦は揮発性メモリで動作します。

## ビルドモード

このプロジェクトは2モードのビルドをサポートします:

| モード | コマンド | CPU対戦 | 外部依存 |
|---|---|---|---|
| **デフォルト** | `go build ./cmd/server` | メモリ実装のみ | 標準ライブラリのみ |
| **postgres** | `go build -tags postgres ./cmd/server` | DB ありなら永続化、なければメモリ | pgx, sqlc生成コード, google/uuid |

`postgres` モードは `DATABASE_URL` 環境変数の有無で実装を自動切替します。

## ローカル開発環境 (DB あり)

### 必要なもの
- Go 1.22+
- Docker + Docker Compose
- sqlc
- golang-migrate

### 初回セットアップ

```bash
# 1. 必要な依存を取得
go get github.com/jackc/pgx/v5/stdlib@v5.5.5    # Go 1.22 の場合
go get github.com/google/uuid
# テスト用
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
go get github.com/golang-migrate/migrate/v4

# 2. internal/adapter/persistence/db.go の冒頭で
#    _ "github.com/jackc/pgx/v5/stdlib"
#    のコメントを外す

# 3. 環境変数を準備
cp .env.example .env

# 4. PostgreSQL / Redis を起動
docker compose up -d

# 5. マイグレーション実行
migrate -path db/migrations \
  -database "postgres://numeron:numeron@localhost:5432/numeron?sslmode=disable" \
  up

# 6. sqlc でGoコード生成
sqlc generate

# 7. テスト (DB統合テスト含む)
go test -tags postgres ./... -race

# 8. Goサーバー起動 (postgres モード)
env $(cat .env | xargs) go run -tags postgres ./cmd/server
```

### 日常的な開発フロー

```bash
# 起動
docker compose up -d
env $(cat .env | xargs) go run -tags postgres ./cmd/server

# SQLクエリ変更時
vim db/queries/xxxxx.sql
sqlc generate
go test -tags postgres ./...

# スキーマ変更時
migrate create -ext sql -dir db/migrations -seq add_xxx
vim db/migrations/000005_add_xxx.up.sql
vim db/migrations/000005_add_xxx.down.sql
migrate -path db/migrations -database "$DATABASE_URL" up
sqlc generate

# 停止
docker compose down       # コンテナのみ
docker compose down -v    # ボリュームも (DBデータ消える)
```

## 環境変数

| 変数 | 値 | デフォルト | 説明 |
|---|---|---|---|
| `PORT` | `8080` | `8080` | 待ち受けポート |
| `LOG_FORMAT` | `text` / `json` | `text` | ログ出力形式 |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` | ログレベル |
| `ENVIRONMENT` | `development` / `production` / `test` | `development` | 動作モード |
| `DATABASE_URL` | postgres URL | (空) | 空ならDB機能無効 |
| `DB_MAX_OPEN_CONNS` | 数値 | `25` | DB接続プール最大数 |
| `DB_MAX_IDLE_CONNS` | 数値 | `5` | DBアイドル接続数 |
| `DB_CONN_MAX_LIFETIME_SEC` | 数値 | `300` | DB接続の最大寿命 (秒) |
| `REDIS_URL` | redis URL | (空) | Redis接続文字列 |

## エンドポイント

| メソッド | パス | 説明 |
|---|---|---|
| GET | `/` | フロントエンド |
| GET | `/api/health` | ヘルスチェック |
| POST | `/api/start` | CPU対戦開始 |
| POST | `/api/guess` | CPU対戦の予想 |
| POST | `/api/online/create` | ルーム作成 |
| POST | `/api/online/join` | ルーム参加 |
| GET | `/api/online/state` | ルーム状態取得 |
| POST | `/api/online/secret` | 暗証設定 |
| POST | `/api/online/guess` | オンライン予想 |
| GET | `/api/online/poll` | ロングポーリング |

## フロントエンド開発 (React)

新規画面は `web/app/` の React + Vite + TypeScript で開発します。
既存のゲーム本体 (`web/static/index.html`) はそのまま並列稼働しています。

### セットアップ

```bash
cd web/app
npm install
```

### 開発

```bash
# ターミナル1: Go サーバー
cd numeron
go run ./cmd/server      # http://localhost:8080

# ターミナル2: Vite dev サーバー
cd numeron/web/app
npm run dev              # http://localhost:5173
```

Vite が `/api` と `/ws` を `localhost:8080` にプロキシするので、
ブラウザで `http://localhost:5173/app` を開けば動作確認できます。

### ビルド

```bash
cd web/app
npm run build            # → dist/ に出力
npm run preview          # ビルド済みファイルでローカルプレビュー
```

### ディレクトリ

```
web/app/
├── package.json
├── vite.config.ts        Vite設定 (proxy 含む)
├── tsconfig.json         TypeScript strict mode
├── index.html
└── src/
    ├── main.tsx          エントリーポイント
    ├── App.tsx           ルート定義
    ├── styles.css        ベーススタイル
    ├── api/
    │   ├── client.ts     fetch ラッパー
    │   ├── error.ts      ApiError クラス
    │   ├── types.ts      サーバー API レスポンス型
    │   └── numeron.ts    API ラッパー関数
    ├── pages/
    │   ├── HomePage.tsx
    │   └── HealthPage.tsx
    ├── components/
    ├── hooks/
    └── store/
        └── auth.ts       Zustand 認証ストア (フェーズ4 で本格使用)
```

## テスト

### バックエンド (Go)
```bash
# デフォルト (メモリ実装のみ)
go test ./... -race -cover

# postgres モード (Docker 起動済み前提)
go test -tags postgres ./... -race -cover
```

### フロントエンド (型チェックのみ)
```bash
cd web/app
npm run lint    # TypeScript の型チェック
npm run build   # ビルド成否で検証
```

## 開発

開発を続ける場合は **[DEVELOPMENT.md](./DEVELOPMENT.md)** を最初に読んでください。

## ディレクトリ構成

```
cmd/server/main.go        エントリーポイント
db/
  SCHEMA.md               ER図と設計判断
  migrations/             golang-migrate
  queries/                sqlc クエリ
sqlc.yaml
internal/
  config/                 環境変数 → Config
  domain/                 ドメインモデル
  usecase/                ビジネスフロー
  port/                   インターフェース
  observability/          ロガー + ミドルウェア
  adapter/
    httphandler/          HTTPハンドラ
    persistence/          ストレージ
      session_store.go         メモリ実装 (CPU)
      room_store.go            メモリ実装 (オンライン)
      postgres_session_store.go (postgres タグでビルド)
      factory.go               デフォルト: メモリのみ
      factory_postgres.go      postgres タグ: DB有無で切替
      sqlc/                    sqlc 生成コード
web/static/               フロントエンド
docker-compose.yml
```

依存方向: `cmd → adapter → usecase → port → domain`

## ライセンス

未定。

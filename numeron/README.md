# NUMER0N

3桁の暗証番号を当て合う対戦ゲーム「Numeron」のWebサービス実装。

CPU対戦とオンライン2人対戦に対応しています。

## クイックスタート (DB なし)

```bash
# Go 1.22 以上が必要
go run ./cmd/server
```

ブラウザで http://localhost:8080 を開く。

## ローカル開発環境 (DB あり)

### 必要なもの
- Go 1.22+
- Docker + Docker Compose (PostgreSQL / Redis 用)
- sqlc (DBクエリのコード生成)
- golang-migrate (マイグレーション)

### ツールのインストール (1度だけ)

```bash
# sqlc
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
# または: brew install sqlc

# golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
# または: brew install golang-migrate
```

### 初回セットアップ

```bash
# 1. PostgreSQL ドライバを追加
go get github.com/jackc/pgx/v5/stdlib@v5.5.5    # Go 1.22 推奨
# Go 1.23+ なら v5.7.5、Go 1.25+ なら最新版

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

# 7. Goサーバー起動
env $(cat .env | xargs) go run ./cmd/server
```

### 日常的な開発フロー

```bash
# 起動
docker compose up -d
env $(cat .env | xargs) go run ./cmd/server

# SQLクエリ変更時
vim db/queries/xxxxx.sql
sqlc generate    # コード再生成
go test ./...

# スキーマ変更時
migrate create -ext sql -dir db/migrations -seq add_users  # 新ファイル作成
vim db/migrations/000002_add_users.up.sql    # 編集
vim db/migrations/000002_add_users.down.sql
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
| GET | `/api/health` | ヘルスチェック (依存先のDB等含む) |
| POST | `/api/start` | CPU対戦開始 |
| POST | `/api/guess` | CPU対戦の予想 |
| POST | `/api/online/create` | ルーム作成 |
| POST | `/api/online/join` | ルーム参加 |
| GET | `/api/online/state` | ルーム状態取得 |
| POST | `/api/online/secret` | 暗証設定 |
| POST | `/api/online/guess` | オンライン予想 |
| GET | `/api/online/poll` | ロングポーリング |

## テスト

```bash
go test ./... -race -cover
```

## 開発

開発を続ける場合は **[DEVELOPMENT.md](./DEVELOPMENT.md)** を最初に読んでください。
ロードマップ、技術選定、進行中のフェーズ、次にやることが全て書いてあります。

## ディレクトリ構成

```
cmd/server/main.go           エントリーポイント
db/
  migrations/                golang-migrate 用
  queries/                   sqlc 用クエリ定義
sqlc.yaml                    sqlc 設定
internal/
  config/                    環境変数 → Config 構造体
  domain/                    ドメインモデル (依存なし)
  usecase/                   アプリケーションロジック
  port/                      インターフェース定義
  observability/             ロガー + ミドルウェア
  adapter/
    httphandler/             HTTPハンドラ
    persistence/             ストレージ (メモリ + DB)
      sqlc/                  sqlc 生成コード (要 `sqlc generate`)
web/static/                  フロントエンド
docker-compose.yml           開発用 PostgreSQL + Redis
.env.example                 環境変数サンプル
```

依存方向: `cmd → adapter → usecase → port → domain`

## ライセンス

未定。

# NUMER0N

3桁の暗証番号を当て合う対戦ゲーム「Numeron」のWebサービス実装。

CPU対戦とオンライン2人対戦に対応しています。

## クイックスタート (DB なし)

```bash
# Go 1.22 以上が必要
go run ./cmd/server
```

ブラウザで http://localhost:8080 を開く。

## ローカル開発環境 (DB あり、フェーズ2.2 以降)

### 必要なもの
- Go 1.22+
- Docker + Docker Compose (PostgreSQL / Redis 用)

### セットアップ
```bash
# 環境変数ファイルを準備
cp .env.example .env

# PostgreSQL / Redis を起動
docker compose up -d

# 接続確認
docker compose ps

# Goサーバー起動 (環境変数を読み込む)
env $(cat .env | xargs) go run ./cmd/server
# または direnv ユーザーは direnv allow
```

### 停止
```bash
docker compose down       # コンテナのみ
docker compose down -v    # ボリュームも削除 (DBデータ消える)
```

## 環境変数

| 変数 | 値 | デフォルト | 説明 |
|---|---|---|---|
| `PORT` | `8080` | `8080` | 待ち受けポート |
| `LOG_FORMAT` | `text` / `json` | `text` | ログ出力形式 |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` | ログレベル |
| `ENVIRONMENT` | `development` / `production` / `test` | `development` | 動作モード |
| `DATABASE_URL` | postgres URL | (空) | DB接続文字列 |
| `DB_MAX_OPEN_CONNS` | 数値 | `25` | DB接続プール最大数 |
| `DB_MAX_IDLE_CONNS` | 数値 | `5` | DBアイドル接続数 |
| `DB_CONN_MAX_LIFETIME_SEC` | 数値 | `300` | DB接続の最大寿命 (秒) |
| `REDIS_URL` | redis URL | (空) | Redis接続文字列 |

詳細は `.env.example` を参照。

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

## テスト

```bash
go test ./... -race -cover
```

## 開発

開発を続ける場合は **[DEVELOPMENT.md](./DEVELOPMENT.md)** を最初に読んでください。
ロードマップ、技術選定、進行中のフェーズ、次にやることが全て書いてあります。

## ディレクトリ構成

```
cmd/server/main.go      エントリーポイント
internal/
  config/               環境変数 → Config 構造体
  domain/               ドメインモデル (依存なし)
  usecase/              アプリケーションロジック
  port/                 インターフェース定義
  observability/        ロガー + ミドルウェア
  adapter/
    httphandler/        HTTPハンドラ
    persistence/        ストレージ (メモリ → DB)
web/static/             フロントエンド
docker-compose.yml      開発用 PostgreSQL + Redis
.env.example            環境変数サンプル
```

依存方向: `cmd → adapter → usecase → port → domain`

## ライセンス

未定。

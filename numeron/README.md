# NUMER0N

3桁の暗証番号を当て合う対戦ゲーム「Numeron」のWebサービス実装。

CPU対戦とオンライン2人対戦に対応しています。

## クイックスタート

```bash
# Go 1.22 以上が必要
go run ./cmd/server
```

ブラウザで http://localhost:8080 を開く。

## 環境変数

| 変数 | 値 | デフォルト | 説明 |
|---|---|---|---|
| `PORT` | `8080` | `8080` | 待ち受けポート |
| `LOG_FORMAT` | `text` / `json` | `text` | ログ出力形式。本番は json 推奨 |
| `LOG_LEVEL` | `debug` / `info` / `warn` / `error` | `info` | ログレベル |

例:
```bash
# 本番向け (JSON, info以上のみ)
LOG_FORMAT=json LOG_LEVEL=info go run ./cmd/server

# デバッグ (テキスト, 全レベル)
LOG_LEVEL=debug go run ./cmd/server
```

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
  domain/               ドメインモデル (依存なし)
  usecase/              アプリケーションロジック
  port/                 インターフェース定義
  observability/        ロガー + ミドルウェア
  adapter/
    httphandler/        HTTPハンドラ
    persistence/        ストレージ (現メモリ、将来DB)
web/static/             フロントエンド
```

依存方向: `cmd → adapter → usecase → port → domain`

## ライセンス

未定。

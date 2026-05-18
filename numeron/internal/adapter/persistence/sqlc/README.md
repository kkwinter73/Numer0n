# sqlc 生成コード

このディレクトリは `sqlc generate` コマンドによって自動生成されるGoコードを格納します。

## ファイルが見当たらない場合

リポジトリをクローンした直後など、まだ生成されていない場合があります。以下を実行してください:

```bash
# sqlc のインストール (1度だけ)
# 方法1: Go install
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# 方法2: Homebrew (macOS)
brew install sqlc

# プロジェクトルートで実行
sqlc generate
```

生成されるファイル (例):
- `db.go` — 接続抽象とトランザクション
- `models.go` — テーブルから生成された Go 構造体
- `querier.go` — クエリのインターフェース定義
- `schema_check.sql.go` — `db/queries/schema_check.sql` から生成されたGoコード

## クエリ追加・変更時

1. `db/queries/*.sql` を編集
2. `sqlc generate` を再実行
3. このディレクトリの内容が更新される
4. `go test ./...` で動作確認

## 生成コードはコミットする?

**Yes, コミットすべき**:
- `sqlc generate` が外部ツールに依存している
- 生成コードを diff で見られる方が安心
- CI で `sqlc generate` を毎回走らせるより速い

ただし手動で編集はしないこと (次回 generate 時に上書きされる)。

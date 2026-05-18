# Claude Code への移行手順

このプロジェクトを Claude Code で開発するための初期セットアップガイドです。

## 前提

- macOS / Linux / Windows (WSL推奨)
- ターミナル(コマンドライン)が使えること
- Anthropic アカウントを持っていること(Claude Pro/Max プラン または API キー)

---

## Step 1: 必要なツールのインストール

### Node.js (Claude Code が必要とする)

- **macOS**:
  ```bash
  # Homebrew がない場合は https://brew.sh からインストール
  brew install node
  ```
- **Linux**:
  ```bash
  # Debian/Ubuntu の場合
  curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash -
  sudo apt-get install -y nodejs
  ```
- **Windows**: WSL2 上で Linux 手順、または https://nodejs.org からインストーラ

### 確認
```bash
node --version    # v18 以上が望ましい
npm --version
```

### Claude Code 本体

```bash
npm install -g @anthropic-ai/claude-code
```

確認:
```bash
claude --version
```

---

## Step 2: ログイン

```bash
claude
```

初回起動時にログイン画面が出ます。

- **Pro / Max プランの場合**: ブラウザが開いて Anthropic アカウントでログイン
- **API キーの場合**: 環境変数 `ANTHROPIC_API_KEY` を設定するか、対話的に入力

---

## Step 3: プロジェクトを展開して Git で管理

```bash
# 適当なディレクトリへ
cd ~/Documents   # または好きな場所

# zip を展開
unzip /path/to/numeron.zip
cd numeron

# Git 初期化
git init
git add -A
git commit -m "Initial commit (Phase 2.2 complete)"
```

GitHub にリモートを設定するなら(任意):
```bash
# GitHub で空リポジトリを作ってから
git remote add origin git@github.com:yourname/numeron.git
git push -u origin main
```

---

## Step 4: 開発ツールのインストール (フェーズ2.3 以降で必要)

### Docker Desktop

PostgreSQL / Redis をローカルで起動するために必要。

- **macOS**: https://www.docker.com/products/docker-desktop からダウンロード
- **Linux**: `sudo apt install docker.io docker-compose-plugin` 等
- **Windows**: Docker Desktop for Windows (WSL2 統合)

確認:
```bash
docker --version
docker compose version
```

### Go 1.22 以上

```bash
# macOS
brew install go

# Linux (Ubuntu)
sudo apt install golang-go
# または公式バイナリ: https://go.dev/dl/

# 確認
go version   # 1.22 以上であること
```

### sqlc

```bash
# macOS
brew install sqlc

# その他
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

### golang-migrate

```bash
# macOS
brew install golang-migrate

# その他
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

---

## Step 5: pgx ドライバを有効化

`internal/adapter/persistence/db.go` の冒頭で副作用 import をコメントアウトしてあります。
これを有効化:

```bash
# pgx 取得 (Go バージョンに合わせる)
go get github.com/jackc/pgx/v5/stdlib@v5.5.5   # Go 1.22 推奨
# go get github.com/jackc/pgx/v5/stdlib@v5.7.5  # Go 1.23+ なら
# go get github.com/jackc/pgx/v5/stdlib         # Go 1.25+ なら最新

# db.go を編集:
#   - _ "github.com/jackc/pgx/v5/stdlib"   ← この行のコメント `// ` を削除
```

`vim` か VS Code で開いて編集。または Claude Code に頼んでもOK。

---

## Step 6: 動作確認

```bash
# テスト
go test ./... -race -cover

# ビルド
go build -o /tmp/numeron ./cmd/server

# DB なしで起動
go run ./cmd/server
# ブラウザで http://localhost:8080 を開いて遊べることを確認

# DB ありで起動
docker compose up -d
cp .env.example .env
# .env を必要に応じて編集

env $(cat .env | xargs) go run ./cmd/server
```

---

## Step 7: Claude Code でフェーズを進める

プロジェクトディレクトリで:

```bash
claude
```

最初の指示:

```
DEVELOPMENT.md と CLAUDE.md を読んで、現在の状況を教えて
```

そして本題:

```
フェーズ2.3をやろう
```

Claude Code は CLAUDE.md と DEVELOPMENT.md を読み、フェーズ2.3の作業を始めます。

---

## トラブルシューティング

### `claude` コマンドが見つからない
PATH 設定が不完全な可能性。`npm config get prefix` でグローバル npm パッケージのインストール先を確認し、その `bin/` を PATH に追加。

### `go: module ... requires go >= 1.25.0`
pgx の最新版は新しい Go を要求します。`@v5.5.5` を指定してダウングレード。

### `docker: command not found`
Docker Desktop が起動しているか確認 (macOS は Dock のクジラアイコン)。

### Claude Code の応答が遅い / 止まる
長時間のセッションでは context が削れます。一度終了 (`/exit` または Ctrl+D) して新セッションを始めると、CLAUDE.md が再度読み込まれる。

### コミットメッセージを自動生成したい
Claude Code に `git diff してコミットメッセージを作って` と頼める。

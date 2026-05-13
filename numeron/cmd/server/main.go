// Command server は Numeron のHTTPサーバーを起動します。
//
// アーキテクチャ:
//
//	cmd/server/main.go         エントリーポイント (依存組み立て)
//	    │
//	    ├─ internal/config/    環境変数 → Config構造体
//	    │
//	    ├─ internal/adapter/   外界とのIO層 (HTTP, DB)
//	    │   ├─ httphandler/    HTTPプロトコル変換
//	    │   └─ persistence/    ストレージ実装 (現在はメモリ、将来DB)
//	    │
//	    ├─ internal/usecase/   アプリケーション固有のビジネスフロー
//	    │
//	    ├─ internal/observability/  ロガー・ミドルウェア
//	    │
//	    ├─ internal/port/      インターフェース定義 (リポジトリ等)
//	    │
//	    └─ internal/domain/    ドメインモデル (依存なし)
//
// 依存方向: cmd → adapter → usecase → port → domain
// observability/config はどの層からでも利用可
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/numeron/numeron/internal/adapter/httphandler"
	"github.com/numeron/numeron/internal/adapter/persistence"
	"github.com/numeron/numeron/internal/config"
	"github.com/numeron/numeron/internal/observability"
	"github.com/numeron/numeron/internal/usecase"
)

func main() {
	// ----- 1. Config 読み込み -----
	cfg, err := config.Load()
	if err != nil {
		// この段階ではロガーが無いので標準 log で
		os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	// ----- 2. ロガー構築 -----
	logger := observability.NewLogger(observability.Config{
		Format: cfg.LogFormat,
		Level:  cfg.LogLevel,
		Output: os.Stdout,
	})
	slog.SetDefault(logger)

	// ----- 3. ストレージ層 -----
	sessionStore := persistence.NewMemorySessionStore()
	roomStore := persistence.NewMemoryRoomStore()

	// ----- 4. usecase 層 -----
	cpuUC := usecase.NewCPUUsecase(sessionStore)
	onlineUC := usecase.NewOnlineUsecase(roomStore)

	// ----- 5. ハンドラ層 -----
	cpuHandler := httphandler.NewCPUHandler(cpuUC)
	onlineHandler := httphandler.NewOnlineHandler(onlineUC)
	// ヘルスチェック: 現状は依存先がメモリのみなのでチェッカーなし
	// フェーズ2.2 でDB追加時に DBHealthChecker を渡す
	healthHandler := httphandler.NewHealthHandler()

	// ----- 6. ルーティング -----
	mux := http.NewServeMux()

	// 静的ファイル
	mux.Handle("/", http.FileServer(http.Dir("web/static")))

	// ヘルスチェック
	mux.HandleFunc("/api/health", healthHandler.HandleHealth)

	// CPU対戦
	mux.HandleFunc("/api/start", cpuHandler.HandleStart)
	mux.HandleFunc("/api/guess", cpuHandler.HandleGuess)

	// オンライン対戦
	mux.HandleFunc("/api/online/create", onlineHandler.HandleCreate)
	mux.HandleFunc("/api/online/join", onlineHandler.HandleJoin)
	mux.HandleFunc("/api/online/state", onlineHandler.HandleState)
	mux.HandleFunc("/api/online/secret", onlineHandler.HandleSecret)
	mux.HandleFunc("/api/online/guess", onlineHandler.HandleGuess)
	mux.HandleFunc("/api/online/poll", onlineHandler.HandlePoll)

	// ----- 7. ミドルウェアチェーン -----
	rootHandler := observability.Chain(
		observability.RecoverMiddleware(),
		observability.RequestIDMiddleware(),
		observability.LoggerMiddleware(logger),
		observability.AccessLogMiddleware(),
	)(mux)

	// ----- 8. サーバー起動 -----
	logger.Info("server starting",
		slog.String("addr", cfg.ListenAddr()),
		slog.String("environment", cfg.Environment),
		slog.String("log_format", cfg.LogFormat),
		slog.String("log_level", cfg.LogLevel),
	)

	if err := http.ListenAndServe(cfg.ListenAddr(), rootHandler); err != nil {
		logger.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

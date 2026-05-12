// Command server は Numeron のHTTPサーバーを起動します。
//
// アーキテクチャ:
//
//	cmd/server/main.go         エントリーポイント (依存組み立て)
//	    │
//	    ├─ internal/adapter/   外界とのIO層 (HTTP, DB)
//	    │   ├─ httphandler/    HTTPプロトコル変換
//	    │   └─ persistence/    ストレージ実装 (現在はメモリ)
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
// observability はどの層からでも利用可 (context経由)
package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/numeron/numeron/internal/adapter/httphandler"
	"github.com/numeron/numeron/internal/adapter/persistence"
	"github.com/numeron/numeron/internal/observability"
	"github.com/numeron/numeron/internal/usecase"
)

func main() {
	// ----- ロガーをまず構築 -----
	logger := observability.NewLogger(observability.NewConfigFromEnv())
	slog.SetDefault(logger) // 標準logger としても使えるように設定

	// ----- ストレージ層 -----
	sessionStore := persistence.NewMemorySessionStore()
	roomStore := persistence.NewMemoryRoomStore()

	// ----- usecase 層 -----
	cpuUC := usecase.NewCPUUsecase(sessionStore)
	onlineUC := usecase.NewOnlineUsecase(roomStore)

	// ----- ハンドラ層 -----
	cpuHandler := httphandler.NewCPUHandler(cpuUC)
	onlineHandler := httphandler.NewOnlineHandler(onlineUC)

	// ----- ルーティング -----
	mux := http.NewServeMux()

	// 静的ファイル
	mux.Handle("/", http.FileServer(http.Dir("web/static")))

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

	// ----- ミドルウェアチェーン -----
	// 順序: Recover (最外側) → RequestID → Logger → AccessLog → mux
	// Recover を最外側にすることで、他のミドルウェア内の panic も拾える
	rootHandler := observability.Chain(
		observability.RecoverMiddleware(),
		observability.RequestIDMiddleware(),
		observability.LoggerMiddleware(logger),
		observability.AccessLogMiddleware(),
	)(mux)

	// ----- サーバー起動 -----
	port := getEnvOrDefault("PORT", ":8080")
	if port[0] != ':' {
		port = ":" + port
	}

	logger.Info("server starting",
		slog.String("port", port),
		slog.String("log_format", os.Getenv("LOG_FORMAT")),
		slog.String("log_level", os.Getenv("LOG_LEVEL")),
	)

	if err := http.ListenAndServe(port, rootHandler); err != nil {
		logger.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

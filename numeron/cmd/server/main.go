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
//	    │   └─ persistence/    ストレージ実装 (メモリ + DB接続)
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
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"time"

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

	// ----- 3. (オプション) DB接続 -----
	// DATABASE_URL が設定されていれば接続を試みる。
	// 空の場合はメモリストアのみで動作 (フェーズ2.2 までの後方互換)。
	var db *sql.DB
	var healthCheckers []httphandler.HealthChecker
	if cfg.DatabaseURL != "" {
		logger.Info("connecting to database")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		db, err = persistence.OpenDB(ctx, persistence.DBConfig{
			URL:             cfg.DatabaseURL,
			MaxOpenConns:    cfg.DBMaxOpenConns,
			MaxIdleConns:    cfg.DBMaxIdleConns,
			ConnMaxLifetime: time.Duration(cfg.DBConnMaxLifetime) * time.Second,
		})
		cancel()
		if err != nil {
			logger.Error("database connection failed", slog.Any("error", err))
			os.Exit(1)
		}
		defer db.Close()
		logger.Info("database connected")
		healthCheckers = append(healthCheckers, persistence.NewDBHealthChecker(db))
	} else {
		logger.Info("database URL not set, running in memory-only mode")
	}

	// ----- 4. ストレージ層 -----
	// フェーズ2.4 まではメモリ実装を使う。DBが繋がっていても今は読み書きしない。
	sessionStore := persistence.NewMemorySessionStore()
	roomStore := persistence.NewMemoryRoomStore()

	// ----- 5. usecase 層 -----
	cpuUC := usecase.NewCPUUsecase(sessionStore)
	onlineUC := usecase.NewOnlineUsecase(roomStore)

	// ----- 6. ハンドラ層 -----
	cpuHandler := httphandler.NewCPUHandler(cpuUC)
	onlineHandler := httphandler.NewOnlineHandler(onlineUC)
	healthHandler := httphandler.NewHealthHandler(healthCheckers...)

	// ----- 7. ルーティング -----
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

	// ----- 8. ミドルウェアチェーン -----
	rootHandler := observability.Chain(
		observability.RecoverMiddleware(),
		observability.RequestIDMiddleware(),
		observability.LoggerMiddleware(logger),
		observability.AccessLogMiddleware(),
	)(mux)

	// ----- 9. サーバー起動 -----
	logger.Info("server starting",
		slog.String("addr", cfg.ListenAddr()),
		slog.String("environment", cfg.Environment),
		slog.String("log_format", cfg.LogFormat),
		slog.String("log_level", cfg.LogLevel),
		slog.Bool("db_connected", db != nil),
	)

	if err := http.ListenAndServe(cfg.ListenAddr(), rootHandler); err != nil {
		logger.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

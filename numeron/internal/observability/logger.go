// Package observability は構造化ログ、リクエストID、ミドルウェアを提供します。
//
// 設計方針:
//   - 標準ライブラリ `log/slog` のみで完結 (zerolog/zap 等の外部依存なし)
//   - 開発時: テキスト形式 (人間が読みやすい)
//   - 本番時: JSON 形式 (Datadog/CloudWatch/Loki 等のログ集約に最適)
//   - リクエストIDは context.Context で運搬。ハンドラ内では LoggerFromContext で取得
//
// usecase / domain 層はロガーを持ちません。エラーを返すだけで、
// ログを書くのは middleware と handler の責務です。
// これにより usecase のテストでロガーをモックする必要がなくなります。
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Config はロガーの設定です。環境変数からの構築は NewConfigFromEnv を使用。
type Config struct {
	// Format は "text" または "json"。
	// 開発中は text、本番は json を推奨。
	Format string

	// Level は "debug", "info", "warn", "error"。デフォルトは info。
	Level string

	// Output はログの出力先。通常は os.Stdout。テスト用に差し替え可能。
	Output io.Writer
}

// NewConfigFromEnv は環境変数から Config を構築します。
//
// 環境変数:
//   - LOG_FORMAT: "text" (default) | "json"
//   - LOG_LEVEL:  "debug" | "info" (default) | "warn" | "error"
func NewConfigFromEnv() Config {
	return Config{
		Format: getEnvOrDefault("LOG_FORMAT", "text"),
		Level:  getEnvOrDefault("LOG_LEVEL", "info"),
		Output: os.Stdout,
	}
}

// NewLogger は Config に従って slog.Logger を構築します。
func NewLogger(cfg Config) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
	}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(cfg.Output, opts)
	default:
		handler = slog.NewTextHandler(cfg.Output, opts)
	}

	return slog.New(handler)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// ============================================================
// Context への logger / request_id 埋め込み
// ============================================================

// 内部キー型 (string ではなく独自型にして衝突回避: Go の context 慣習)
type ctxKey int

const (
	ctxKeyLogger ctxKey = iota
	ctxKeyRequestID
)

// ContextWithLogger は context に logger を埋め込みます。
func ContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, logger)
}

// LoggerFromContext は context から logger を取り出します。
// 埋め込まれていなければ slog.Default() を返します (panic しない)。
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKeyLogger).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// ContextWithRequestID は context に request_id を埋め込みます。
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// RequestIDFromContext は context から request_id を取り出します。
// 埋め込まれていなければ空文字列を返します。
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// ============================================================
// Request ID 生成
// ============================================================

// NewRequestID は16文字の16進数 (8バイトの暗号学的乱数) のリクエストIDを生成します。
// UUIDより短く、ログで読みやすい長さです。衝突確率は実用上無視できます (誕生日衝突で
// 約42億リクエストに1回程度)。
func NewRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

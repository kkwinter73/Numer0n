// Package config はアプリケーション全体の設定を集約します。
//
// 設計方針:
//   - 環境変数から読み込み、構造体にまとめる (12-factor app の原則)
//   - 外部ライブラリ (envconfig 等) を使わず標準ライブラリで完結
//   - DSN等の機密情報は環境変数経由のみ。コードにハードコードしない
//   - 開発時は .env ファイルからも読み込み可能 (ただし main.go 側で対応)
//
// 使い方:
//
//	cfg, err := config.Load()
//	if err != nil { ... }
//	fmt.Println(cfg.Port)
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config はアプリケーション全体の設定です。
type Config struct {
	// HTTP サーバー設定
	Port string // 例: "8080" (コロンは付けない)

	// ログ設定
	LogFormat string // "text" | "json"
	LogLevel  string // "debug" | "info" | "warn" | "error"

	// データベース設定 (フェーズ2.2 で使用開始)
	// 形式: postgres://user:pass@host:port/dbname?sslmode=disable
	DatabaseURL string

	// データベース接続プール設定
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int // 秒単位

	// Redis設定 (フェーズ3/4 で使用開始)
	// 形式: redis://[user:password@]host:port[/db]
	RedisURL string

	// 動作モード (本番 / 開発 / テスト)
	// 一部の挙動 (静的ファイルパス等) に影響する場合がある
	Environment string // "development" | "production" | "test"
}

// Load は環境変数から Config を構築します。
// 必須項目が欠けている場合はエラーを返します。
// (現状は DatabaseURL/RedisURL は必須でない: 段階導入のため)
func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		LogFormat:         getEnv("LOG_FORMAT", "text"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 300),
		RedisURL:          getEnv("REDIS_URL", ""),
		Environment:       getEnv("ENVIRONMENT", "development"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate は Config の整合性をチェックします。
func (c *Config) validate() error {
	// Port は数値であること
	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("PORT must be numeric, got %q", c.Port)
	}

	// LogFormat は許可された値のみ
	switch strings.ToLower(c.LogFormat) {
	case "text", "json":
	default:
		return fmt.Errorf("LOG_FORMAT must be text or json, got %q", c.LogFormat)
	}

	// LogLevel は許可された値のみ
	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "warning", "error":
	default:
		return fmt.Errorf("LOG_LEVEL invalid, got %q", c.LogLevel)
	}

	// Environment は許可された値のみ
	switch strings.ToLower(c.Environment) {
	case "development", "production", "test":
	default:
		return fmt.Errorf("ENVIRONMENT must be development|production|test, got %q", c.Environment)
	}

	// 接続プール数は正の値
	if c.DBMaxOpenConns < 1 {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be >= 1, got %d", c.DBMaxOpenConns)
	}
	if c.DBMaxIdleConns < 0 {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be >= 0, got %d", c.DBMaxIdleConns)
	}

	return nil
}

// IsProduction は本番環境かどうかを返します。
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// ListenAddr は HTTPサーバー の bind アドレスを返します。
// ":8080" のような形式。
func (c *Config) ListenAddr() string {
	return ":" + c.Port
}

// ---------- helpers ----------

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}

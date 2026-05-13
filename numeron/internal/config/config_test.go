package config

import (
	"strings"
	"testing"
)

// setEnv は t.Setenv で環境変数を設定 (テスト終了時に自動復元)
// 標準の t.Setenv を直接使うのと同じ。テストの可読性のため別名にしている。

// =====================================================
// Load
// =====================================================

func TestLoad_defaults(t *testing.T) {
	// 環境変数を全部クリアして、デフォルト値が入ることを確認
	for _, k := range []string{"PORT", "LOG_FORMAT", "LOG_LEVEL", "DATABASE_URL", "REDIS_URL", "ENVIRONMENT", "DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME_SEC"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	expected := map[string]interface{}{
		"Port":              "8080",
		"LogFormat":         "text",
		"LogLevel":          "info",
		"DatabaseURL":       "",
		"RedisURL":          "",
		"Environment":       "development",
		"DBMaxOpenConns":    25,
		"DBMaxIdleConns":    5,
		"DBConnMaxLifetime": 300,
	}

	if cfg.Port != expected["Port"] {
		t.Errorf("Port = %q, want %q", cfg.Port, expected["Port"])
	}
	if cfg.LogFormat != expected["LogFormat"] {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, expected["LogFormat"])
	}
	if cfg.LogLevel != expected["LogLevel"] {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, expected["LogLevel"])
	}
	if cfg.Environment != expected["Environment"] {
		t.Errorf("Environment = %q, want %q", cfg.Environment, expected["Environment"])
	}
	if cfg.DBMaxOpenConns != expected["DBMaxOpenConns"] {
		t.Errorf("DBMaxOpenConns = %d, want %d", cfg.DBMaxOpenConns, expected["DBMaxOpenConns"])
	}
	if cfg.DBMaxIdleConns != expected["DBMaxIdleConns"] {
		t.Errorf("DBMaxIdleConns = %d, want %d", cfg.DBMaxIdleConns, expected["DBMaxIdleConns"])
	}
}

func TestLoad_fromEnv(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/numeron")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_MAX_OPEN_CONNS", "100")
	t.Setenv("DB_MAX_IDLE_CONNS", "10")
	t.Setenv("DB_CONN_MAX_LIFETIME_SEC", "600")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Port != "9000" {
		t.Errorf("Port = %q, want 9000", cfg.Port)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json", cfg.LogFormat)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", cfg.LogLevel)
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/numeron" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://localhost:6379" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if cfg.Environment != "production" {
		t.Errorf("Environment = %q", cfg.Environment)
	}
	if cfg.DBMaxOpenConns != 100 {
		t.Errorf("DBMaxOpenConns = %d, want 100", cfg.DBMaxOpenConns)
	}
}

// =====================================================
// Validation
// =====================================================

func TestLoad_invalidPort(t *testing.T) {
	t.Setenv("PORT", "not-a-number")
	_, err := Load()
	if err == nil {
		t.Fatal("不正なPortでエラーを期待")
	}
	if !strings.Contains(err.Error(), "PORT") {
		t.Errorf("エラーメッセージにPORTが含まれない: %v", err)
	}
}

func TestLoad_invalidLogFormat(t *testing.T) {
	t.Setenv("LOG_FORMAT", "xml")
	_, err := Load()
	if err == nil {
		t.Fatal("不正なLOG_FORMATでエラーを期待")
	}
}

func TestLoad_invalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "verbose")
	_, err := Load()
	if err == nil {
		t.Fatal("不正なLOG_LEVELでエラーを期待")
	}
}

func TestLoad_invalidEnvironment(t *testing.T) {
	t.Setenv("ENVIRONMENT", "staging")
	_, err := Load()
	if err == nil {
		t.Fatal("不正なENVIRONMENTでエラーを期待")
	}
}

func TestLoad_invalidDBPoolSize(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "0")
	_, err := Load()
	if err == nil {
		t.Fatal("DB_MAX_OPEN_CONNS=0 でエラーを期待")
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "5")
	t.Setenv("DB_MAX_IDLE_CONNS", "-1")
	_, err = Load()
	if err == nil {
		t.Fatal("DB_MAX_IDLE_CONNS=-1 でエラーを期待")
	}
}

// =====================================================
// 不正な数値環境変数はデフォルトにフォールバック
// =====================================================

func TestLoad_invalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "not-a-number")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("数値変換失敗時はデフォルトに落ちるべき: %v", err)
	}
	if cfg.DBMaxOpenConns != 25 {
		t.Errorf("DBMaxOpenConns = %d, want default 25", cfg.DBMaxOpenConns)
	}
}

// =====================================================
// ヘルパーメソッド
// =====================================================

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"PRODUCTION", true},
		{"development", false},
		{"test", false},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			c := &Config{Environment: tt.env}
			if got := c.IsProduction(); got != tt.want {
				t.Errorf("IsProduction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_ListenAddr(t *testing.T) {
	c := &Config{Port: "8080"}
	if got := c.ListenAddr(); got != ":8080" {
		t.Errorf("ListenAddr() = %q, want :8080", got)
	}
}

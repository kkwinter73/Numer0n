package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"testing"
)

// =====================================================
// NewLogger
// =====================================================

func TestNewLogger_textFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(Config{Format: "text", Level: "info", Output: &buf})

	logger.Info("hello", slog.String("key", "value"))

	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Errorf("ログに 'hello' が含まれない: %q", out)
	}
	if !strings.Contains(out, "key=value") {
		t.Errorf("属性 key=value が含まれない: %q", out)
	}
	// JSON ではないこと
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("textフォーマットなのにJSON: %q", out)
	}
}

func TestNewLogger_jsonFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(Config{Format: "json", Level: "info", Output: &buf})

	logger.Info("hello", slog.String("key", "value"))

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSONパース失敗: %v, out=%q", err, buf.String())
	}
	if parsed["msg"] != "hello" {
		t.Errorf("msg = %v, want hello", parsed["msg"])
	}
	if parsed["key"] != "value" {
		t.Errorf("key = %v, want value", parsed["key"])
	}
	if parsed["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", parsed["level"])
	}
}

func TestNewLogger_levelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(Config{Format: "text", Level: "warn", Output: &buf})

	logger.Debug("debug msg")
	logger.Info("info msg")
	logger.Warn("warn msg")
	logger.Error("error msg")

	out := buf.String()
	if strings.Contains(out, "debug msg") {
		t.Errorf("debug がフィルタされていない: %q", out)
	}
	if strings.Contains(out, "info msg") {
		t.Errorf("info がフィルタされていない: %q", out)
	}
	if !strings.Contains(out, "warn msg") {
		t.Errorf("warn が出力されない: %q", out)
	}
	if !strings.Contains(out, "error msg") {
		t.Errorf("error が出力されない: %q", out)
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"unknown", slog.LevelInfo}, // デフォルト
		{"", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseLevel(tt.input); got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// =====================================================
// Context 関数
// =====================================================

func TestContextWithLogger_andFrom(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(Config{Format: "text", Level: "info", Output: &buf})

	ctx := ContextWithLogger(context.Background(), logger)
	got := LoggerFromContext(ctx)
	if got != logger {
		t.Errorf("取り出したloggerが入れたものと違う")
	}

	got.Info("test")
	if !strings.Contains(buf.String(), "test") {
		t.Errorf("ログが書かれていない")
	}
}

func TestLoggerFromContext_defaultWhenAbsent(t *testing.T) {
	// 何も埋め込まれていない context でも panic せず default を返す
	got := LoggerFromContext(context.Background())
	if got == nil {
		t.Errorf("LoggerFromContext が nil を返した")
	}
	// slog.Default() と一致する (実装の意図)
	if got != slog.Default() {
		t.Errorf("default logger 以外が返された")
	}
}

func TestContextWithRequestID_andFrom(t *testing.T) {
	ctx := ContextWithRequestID(context.Background(), "test-id-123")
	if got := RequestIDFromContext(ctx); got != "test-id-123" {
		t.Errorf("RequestIDFromContext = %q, want test-id-123", got)
	}
}

func TestRequestIDFromContext_emptyWhenAbsent(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("空contextから %q が返ってきた", got)
	}
}

// =====================================================
// NewRequestID
// =====================================================

func TestNewRequestID(t *testing.T) {
	hexRe := regexp.MustCompile(`^[0-9a-f]{16}$`)

	id := NewRequestID()
	if !hexRe.MatchString(id) {
		t.Errorf("ID形式が不正: %q", id)
	}

	// 一意性 (確率的)
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := NewRequestID()
		if seen[id] {
			t.Fatalf("ID重複: %s", id)
		}
		seen[id] = true
	}
}

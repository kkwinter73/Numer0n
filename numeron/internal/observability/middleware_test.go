package observability

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =====================================================
// Chain
// =====================================================

func TestChain_order(t *testing.T) {
	// ミドルウェア適用順を検証
	var calls []string
	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, "before:"+name)
				next.ServeHTTP(w, r)
				calls = append(calls, "after:"+name)
			})
		}
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "handler")
	})

	chained := Chain(mw("A"), mw("B"), mw("C"))(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	chained.ServeHTTP(httptest.NewRecorder(), req)

	want := []string{
		"before:A", "before:B", "before:C",
		"handler",
		"after:C", "after:B", "after:A",
	}
	if len(calls) != len(want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	for i, c := range calls {
		if c != want[i] {
			t.Errorf("calls[%d] = %q, want %q", i, c, want[i])
		}
	}
}

// =====================================================
// RequestIDMiddleware
// =====================================================

func TestRequestIDMiddleware_generatesAndEchoes(t *testing.T) {
	var capturedID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestIDFromContext(r.Context())
	})

	mw := RequestIDMiddleware()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	if capturedID == "" {
		t.Errorf("context にrequest_idが付かなかった")
	}
	if w.Header().Get("X-Request-ID") != capturedID {
		t.Errorf("レスポンスヘッダの X-Request-ID が context のものと一致しない")
	}
}

func TestRequestIDMiddleware_honorsClientID(t *testing.T) {
	// クライアントが X-Request-ID を送ってきた場合はそれを尊重する
	var capturedID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestIDFromContext(r.Context())
	})

	mw := RequestIDMiddleware()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "client-provided-id")
	w := httptest.NewRecorder()
	mw(handler).ServeHTTP(w, req)

	if capturedID != "client-provided-id" {
		t.Errorf("クライアント提供のIDが尊重されていない: got %q", capturedID)
	}
	if w.Header().Get("X-Request-ID") != "client-provided-id" {
		t.Errorf("レスポンスヘッダにエコーされていない")
	}
}

// =====================================================
// LoggerMiddleware
// =====================================================

func TestLoggerMiddleware_attachesRequestID(t *testing.T) {
	var buf bytes.Buffer
	base := NewLogger(Config{Format: "text", Level: "info", Output: &buf})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		LoggerFromContext(r.Context()).Info("from-handler")
	})

	chained := Chain(RequestIDMiddleware(), LoggerMiddleware(base))(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "abc123")
	chained.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	if !strings.Contains(out, "from-handler") {
		t.Errorf("ハンドラからのログが出ていない: %q", out)
	}
	if !strings.Contains(out, "request_id=abc123") {
		t.Errorf("ログに request_id が付与されていない: %q", out)
	}
}

// =====================================================
// AccessLogMiddleware
// =====================================================

func TestAccessLogMiddleware_logsAPIRequests(t *testing.T) {
	var buf bytes.Buffer
	base := NewLogger(Config{Format: "text", Level: "info", Output: &buf})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})

	chained := Chain(LoggerMiddleware(base), AccessLogMiddleware())(handler)
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	chained.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	if !strings.Contains(out, "request started") {
		t.Errorf("'request started' ログがない: %q", out)
	}
	if !strings.Contains(out, "request completed") {
		t.Errorf("'request completed' ログがない: %q", out)
	}
	if !strings.Contains(out, "status=201") {
		t.Errorf("ステータスがログに記録されていない: %q", out)
	}
	if !strings.Contains(out, "duration_ms=") {
		t.Errorf("duration_ms がログに記録されていない: %q", out)
	}
}

func TestAccessLogMiddleware_skipsStaticRequests(t *testing.T) {
	// /api/ 以外のリクエスト (静的ファイル等) はログを出さない
	var buf bytes.Buffer
	base := NewLogger(Config{Format: "text", Level: "info", Output: &buf})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>"))
	})

	chained := Chain(LoggerMiddleware(base), AccessLogMiddleware())(handler)
	req := httptest.NewRequest(http.MethodGet, "/index.html", nil)
	chained.ServeHTTP(httptest.NewRecorder(), req)

	if buf.Len() > 0 {
		t.Errorf("静的ファイルのログが出ている: %q", buf.String())
	}
}

// =====================================================
// RecoverMiddleware
// =====================================================

func TestRecoverMiddleware_recoversFromPanic(t *testing.T) {
	var buf bytes.Buffer
	base := NewLogger(Config{Format: "text", Level: "error", Output: &buf})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	chained := Chain(LoggerMiddleware(base), RecoverMiddleware())(handler)
	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	// panicでプロセス終了しないこと
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("RecoverMiddleware が panic を吸収していない")
		}
	}()
	chained.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Errorf("panic ログが出ていない: %q", buf.String())
	}
	if !strings.Contains(w.Body.String(), "INTERNAL_ERROR") {
		t.Errorf("レスポンスにINTERNAL_ERROR がない: %q", w.Body.String())
	}
}

// =====================================================
// statusRecorder
// =====================================================

func TestStatusRecorder_capturesStatus(t *testing.T) {
	inner := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: inner, status: http.StatusOK}

	rec.WriteHeader(http.StatusCreated)
	if rec.status != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.status)
	}

	// 2回目の WriteHeader は無視される (net/http の挙動と一致)
	rec.WriteHeader(http.StatusBadRequest)
	if rec.status != http.StatusCreated {
		t.Errorf("2回目のWriteHeaderで上書きされた: %d", rec.status)
	}
}

func TestStatusRecorder_defaultsTo200(t *testing.T) {
	// WriteHeader を呼ばずに Write しただけなら 200
	inner := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: inner, status: http.StatusOK}
	_, _ = rec.Write([]byte("hello"))
	if rec.status != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.status)
	}
}

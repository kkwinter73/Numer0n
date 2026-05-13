package httphandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeChecker はテスト用の HealthChecker 実装
type fakeChecker struct {
	name string
	err  error
	// チェック呼び出し時に呼ばれるフック (タイムアウト確認用)
	beforeCheck func(ctx context.Context) error
}

func (f *fakeChecker) Name() string { return f.name }
func (f *fakeChecker) Check(ctx context.Context) error {
	if f.beforeCheck != nil {
		if err := f.beforeCheck(ctx); err != nil {
			return err
		}
	}
	return f.err
}

// =====================================================
// HandleHealth
// =====================================================

func TestHealthHandler_noCheckers(t *testing.T) {
	h := NewHealthHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	h.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if len(resp.Dependencies) != 0 {
		t.Errorf("空のcheckersなら Dependencies は空: %v", resp.Dependencies)
	}
}

func TestHealthHandler_allHealthy(t *testing.T) {
	h := NewHealthHandler(
		&fakeChecker{name: "database"},
		&fakeChecker{name: "redis"},
	)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	h.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want ok", resp.Status)
	}
	if len(resp.Dependencies) != 2 {
		t.Fatalf("Dependencies = %d, want 2", len(resp.Dependencies))
	}
	for _, dep := range resp.Dependencies {
		if dep.Status != "ok" {
			t.Errorf("%s status = %q, want ok", dep.Name, dep.Status)
		}
	}
}

func TestHealthHandler_oneDependencyFailed(t *testing.T) {
	h := NewHealthHandler(
		&fakeChecker{name: "database"},
		&fakeChecker{name: "redis", err: fmt.Errorf("connection refused")},
	)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	h.HandleHealth(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", w.Code)
	}
	var resp healthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != "degraded" {
		t.Errorf("status = %q, want degraded", resp.Status)
	}

	// 失敗したdependencyのエラーメッセージが含まれる
	var redisDep dependencyStatus
	for _, d := range resp.Dependencies {
		if d.Name == "redis" {
			redisDep = d
		}
	}
	if redisDep.Status != "fail" {
		t.Errorf("redis status = %q, want fail", redisDep.Status)
	}
	if redisDep.Error != "connection refused" {
		t.Errorf("redis error = %q, want 'connection refused'", redisDep.Error)
	}
}

func TestHealthHandler_invalidMethod(t *testing.T) {
	h := NewHealthHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	w := httptest.NewRecorder()
	h.HandleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHealthHandler_timeoutAppliedToCheckers(t *testing.T) {
	// チェッカーに 2秒タイムアウトが効いていることを確認
	// 3秒スリープするチェッカー → コンテキストキャンセルされる
	slowChecker := &fakeChecker{
		name: "slow",
		beforeCheck: func(ctx context.Context) error {
			select {
			case <-time.After(3 * time.Second):
				return nil
			case <-ctx.Done():
				return ctx.Err() // タイムアウトエラー
			}
		},
	}

	h := NewHealthHandler(slowChecker)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	h.HandleHealth(w, req)
	elapsed := time.Since(start)

	// 2秒のタイムアウトが効いて、3秒もかからずに返ること
	if elapsed > 2500*time.Millisecond {
		t.Errorf("タイムアウトが効いていない: %v", elapsed)
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 (timeout)", w.Code)
	}
}

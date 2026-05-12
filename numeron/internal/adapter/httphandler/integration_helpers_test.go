package httphandler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/numeron/numeron/internal/adapter/persistence"
	"github.com/numeron/numeron/internal/usecase"
)

// TestMain でテスト中の slog 出力を抑制します。
// デフォルトロガーが Info を出してしまうと、テスト出力が「rejected」ログで埋まり読みにくい。
// `go test -v` でテスト結果に集中できるよう、Error レベル以上のみ出力するロガーに差し替えます。
func TestMain(m *testing.M) {
	silentLogger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError, // テスト中は Error 以上のみ
	}))
	slog.SetDefault(silentLogger)
	os.Exit(m.Run())
}

// =====================================================
// 統合テスト用のサーバー起動ヘルパー
// =====================================================

// testServer は cmd/server/main.go と同じ依存組み立てを行い、
// httptest.Server を返します。テスト終了時に Close() を忘れないこと。
//
// 本物のサーバーと同じく、メモリストア → usecase → handler の流れを
// 再現するため、永続化挙動を含む統合テストが可能になります。
//
// 注意: ミドルウェア (RequestID/Logger/AccessLog/Recover) はここでは付けていません。
// それらは observability パッケージ側でテスト済みのため、handler 統合テストでは
// 純粋に handler+usecase+persistence の組み合わせのみを検証します。
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	sessionStore := persistence.NewMemorySessionStore()
	roomStore := persistence.NewMemoryRoomStore()

	cpuUC := usecase.NewCPUUsecase(sessionStore)
	onlineUC := usecase.NewOnlineUsecase(roomStore)

	cpuHandler := NewCPUHandler(cpuUC)
	onlineHandler := NewOnlineHandler(onlineUC)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/start", cpuHandler.HandleStart)
	mux.HandleFunc("/api/guess", cpuHandler.HandleGuess)
	mux.HandleFunc("/api/online/create", onlineHandler.HandleCreate)
	mux.HandleFunc("/api/online/join", onlineHandler.HandleJoin)
	mux.HandleFunc("/api/online/state", onlineHandler.HandleState)
	mux.HandleFunc("/api/online/secret", onlineHandler.HandleSecret)
	mux.HandleFunc("/api/online/guess", onlineHandler.HandleGuess)
	mux.HandleFunc("/api/online/poll", onlineHandler.HandlePoll)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close) // テスト終了時に自動Close
	return srv
}

// =====================================================
// HTTP リクエスト用ユーティリティ
// =====================================================

// postJSON は POST + JSON body の便利関数です。
// レスポンスJSONを out にデコードして返します。
func postJSON(t *testing.T, url string, body interface{}, out interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("http post: %v", err)
	}
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			resp.Body.Close()
			t.Fatalf("json decode: %v", err)
		}
	}
	return resp
}

// getJSON は GET + JSONレスポンスデコードの便利関数です。
func getJSON(t *testing.T, url string, out interface{}) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			resp.Body.Close()
			t.Fatalf("json decode: %v", err)
		}
	}
	return resp
}

// decodeAPIError はエラーレスポンス body を APIError に変換します。
// status >= 400 のレスポンスで呼び出すこと。
func decodeAPIError(t *testing.T, resp *http.Response) APIError {
	t.Helper()
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var env errorResponseEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("エラーレスポンスのJSON parse失敗: %v, body: %s", err, body)
	}
	return env.Error
}

// assertErrorResponse はエラーレスポンスのステータスとコードを検証します。
func assertErrorResponse(t *testing.T, resp *http.Response, wantStatus int, wantCode string) {
	t.Helper()
	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("status = %d, want %d. body: %s", resp.StatusCode, wantStatus, body)
		return
	}
	apiErr := decodeAPIError(t, resp)
	if apiErr.Code != wantCode {
		t.Errorf("code = %q, want %q (message: %q)", apiErr.Code, wantCode, apiErr.Message)
	}
}

package httphandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/numeron/numeron/internal/usecase"
)

// =====================================================
// writeUsecaseError のマッピングテスト
// =====================================================

func TestWriteUsecaseError_mapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsgIn  string // 期待するメッセージの部分一致
	}{
		{
			name:       "ErrInvalidInput",
			err:        fmt.Errorf("%w: 3桁で入力してください", usecase.ErrInvalidInput),
			wantStatus: 400,
			wantCode:   "INVALID_INPUT",
			wantMsgIn:  "3桁",
		},
		{
			name:       "ErrSessionNotFound",
			err:        fmt.Errorf("%w: セッションが見つかりません", usecase.ErrSessionNotFound),
			wantStatus: 400,
			wantCode:   "SESSION_NOT_FOUND",
			wantMsgIn:  "見つかり",
		},
		{
			name:       "ErrRoomNotFound",
			err:        fmt.Errorf("%w: ルームが見つかりません", usecase.ErrRoomNotFound),
			wantStatus: 404,
			wantCode:   "ROOM_NOT_FOUND",
			wantMsgIn:  "ルーム",
		},
		{
			name:       "ErrUnauthorized",
			err:        fmt.Errorf("%w: トークンが不正です", usecase.ErrUnauthorized),
			wantStatus: 401,
			wantCode:   "UNAUTHORIZED",
			wantMsgIn:  "トークン",
		},
		{
			name:       "ErrConflict",
			err:        fmt.Errorf("%w: 満員です", usecase.ErrConflict),
			wantStatus: 400,
			wantCode:   "CONFLICT",
			wantMsgIn:  "満員",
		},
		{
			name:       "unknown system error",
			err:        errors.New("DB connection lost"),
			wantStatus: 500,
			wantCode:   "INTERNAL_ERROR",
			wantMsgIn:  "内部エラー",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeUsecaseError(context.Background(), w, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var env errorResponseEnvelope
			if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
				t.Fatalf("レスポンスが不正なJSON: %v, body: %s", err, w.Body.String())
			}
			if env.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", env.Error.Code, tt.wantCode)
			}
			if tt.wantMsgIn != "" && !contains(env.Error.Message, tt.wantMsgIn) {
				t.Errorf("message = %q, want to contain %q", env.Error.Message, tt.wantMsgIn)
			}

			// レスポンスは Content-Type: application/json
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

// =====================================================
// unwrapUserMessage の挙動
// =====================================================

func TestUnwrapUserMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			// wrap された場合は base のプレフィックスを剥がす
			name: "wrapped",
			err:  fmt.Errorf("%w: ユーザー向けメッセージ", usecase.ErrInvalidInput),
			want: "ユーザー向けメッセージ",
		},
		{
			// wrap されていない裸エラーは全体を返す
			name: "naked",
			err:  errors.New("そのままのメッセージ"),
			want: "そのままのメッセージ",
		},
		{
			// base error のみで wrap されている場合はそのままを返す
			name: "wrapped without colon",
			err:  fmt.Errorf("%w", usecase.ErrInvalidInput),
			want: "invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unwrapUserMessage(tt.err)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// =====================================================
// writeMethodNotAllowed / writeBadRequest の確認
// =====================================================

func TestWriteMethodNotAllowed(t *testing.T) {
	w := httptest.NewRecorder()
	writeMethodNotAllowed(w)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
	var env errorResponseEnvelope
	json.Unmarshal(w.Body.Bytes(), &env)
	if env.Error.Code != CodeMethodNotAllowed {
		t.Errorf("code = %q, want METHOD_NOT_ALLOWED", env.Error.Code)
	}
}

func TestWriteBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	writeBadRequest(w, "Invalid JSON")

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var env errorResponseEnvelope
	json.Unmarshal(w.Body.Bytes(), &env)
	if env.Error.Code != CodeBadRequest {
		t.Errorf("code = %q, want BAD_REQUEST", env.Error.Code)
	}
	if env.Error.Message != "Invalid JSON" {
		t.Errorf("message = %q, want 'Invalid JSON'", env.Error.Message)
	}
}

// =====================================================
// APIError の JSON 形式
// =====================================================

func TestAPIError_JSONShape(t *testing.T) {
	// details なし
	e := APIError{Code: "INVALID_INPUT", Message: "msg"}
	b, _ := json.Marshal(e)
	if string(b) != `{"code":"INVALID_INPUT","message":"msg"}` {
		t.Errorf("JSON shape (no details): %s", b)
	}

	// details あり
	e = APIError{
		Code:    "INVALID_INPUT",
		Message: "msg",
		Details: map[string]interface{}{"field": "secret"},
	}
	b, _ = json.Marshal(e)
	if string(b) != `{"code":"INVALID_INPUT","message":"msg","details":{"field":"secret"}}` {
		t.Errorf("JSON shape (with details): %s", b)
	}
}

// =====================================================
// ヘルパー
// =====================================================

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

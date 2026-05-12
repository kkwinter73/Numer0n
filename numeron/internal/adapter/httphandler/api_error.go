package httphandler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/numeron/numeron/internal/observability"
	"github.com/numeron/numeron/internal/usecase"
)

// APIError は構造化エラーレスポンスのペイロードです。
//
// レスポンス形式 (HTTP body):
//
//	{
//	  "error": {
//	    "code": "INVALID_INPUT",
//	    "message": "3桁で入力してください",
//	    "details": { ... }     // optional
//	  }
//	}
//
// `code` は機械可読の識別子で、クライアントが分岐に使います。
// `message` は人間向けの説明 (現状は日本語)。i18n対応するなら将来 code から
// クライアント側で翻訳辞書を引く形にできます。
// `details` は フィールド別検証エラー等の補足情報。
type APIError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// エラーコード定数。
// クライアントとサーバーで共有される「契約」なので、変更する場合は破壊的変更扱い。
const (
	CodeInvalidInput     = "INVALID_INPUT"
	CodeSessionNotFound  = "SESSION_NOT_FOUND"
	CodeRoomNotFound     = "ROOM_NOT_FOUND"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeConflict         = "CONFLICT"
	CodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	CodeBadRequest       = "BAD_REQUEST"
	CodeInternalError    = "INTERNAL_ERROR"
)

// errorResponseEnvelope は { "error": APIError } の外側を表します。
type errorResponseEnvelope struct {
	Error APIError `json:"error"`
}

// WriteAPIError は構造化エラーレスポンスを書き出します。
func WriteAPIError(w http.ResponseWriter, status int, e APIError) {
	WriteJSON(w, status, errorResponseEnvelope{Error: e})
}

// writeUsecaseError は usecase エラーを HTTP ステータス + APIError へマッピングします。
//
// 業務エラー (400/401/404) は Info ログ、システムエラー (500) は Error ログを出力します。
// ロガーは context から取り出し、リクエストIDが自動で付与されます。
//
// マッピング規則:
//   - ErrInvalidInput      -> 400 / INVALID_INPUT
//   - ErrSessionNotFound   -> 400 / SESSION_NOT_FOUND (旧API互換のため 400)
//   - ErrRoomNotFound      -> 404 / ROOM_NOT_FOUND
//   - ErrUnauthorized      -> 401 / UNAUTHORIZED
//   - ErrConflict          -> 400 / CONFLICT
//   - その他               -> 500 / INTERNAL_ERROR (詳細はクライアントに見せない)
func writeUsecaseError(ctx context.Context, w http.ResponseWriter, err error) {
	msg := unwrapUserMessage(err)
	logger := observability.LoggerFromContext(ctx)

	var apiErr APIError
	var status int

	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		apiErr = APIError{Code: CodeInvalidInput, Message: msg}
		status = http.StatusBadRequest
	case errors.Is(err, usecase.ErrSessionNotFound):
		apiErr = APIError{Code: CodeSessionNotFound, Message: msg}
		status = http.StatusBadRequest
	case errors.Is(err, usecase.ErrRoomNotFound):
		apiErr = APIError{Code: CodeRoomNotFound, Message: msg}
		status = http.StatusNotFound
	case errors.Is(err, usecase.ErrUnauthorized):
		apiErr = APIError{Code: CodeUnauthorized, Message: msg}
		status = http.StatusUnauthorized
	case errors.Is(err, usecase.ErrConflict):
		apiErr = APIError{Code: CodeConflict, Message: msg}
		status = http.StatusBadRequest
	default:
		// 想定外のシステムエラー。詳細はクライアントに見せない。
		apiErr = APIError{Code: CodeInternalError, Message: "内部エラーが発生しました"}
		status = http.StatusInternalServerError
	}

	// ログ: システムエラーは Error、業務エラーは Info
	if status >= 500 {
		logger.Error("internal error",
			slog.String("code", apiErr.Code),
			slog.Any("error", err),
		)
	} else {
		logger.Info("request rejected",
			slog.String("code", apiErr.Code),
			slog.String("message", apiErr.Message),
			slog.Int("status", status),
		)
	}

	WriteAPIError(w, status, apiErr)
}

// unwrapUserMessage は usecase エラーから ユーザー向けメッセージを抽出します。
// `fmt.Errorf("%w: <詳細>", baseErr)` 形式のエラーから "<詳細>" の部分を取り出します。
// 詳細部分が無い場合は base のエラーメッセージを返します。
func unwrapUserMessage(err error) string {
	msg := err.Error()
	if base := errors.Unwrap(err); base != nil {
		prefix := base.Error() + ": "
		if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
			return msg[len(prefix):]
		}
	}
	return msg
}

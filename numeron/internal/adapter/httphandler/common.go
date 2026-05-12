// Package httphandler はHTTPリクエストを受け取り、usecase層に処理を委譲します。
//
// このパッケージの責務:
//   - HTTPメソッドの確認
//   - JSONリクエストのデコード
//   - usecase 呼び出し
//   - 結果のJSONエンコードとHTTPステータス決定
//   - usecase の業務エラーを HTTP ステータス + 構造化エラーへマッピング (api_error.go)
//
// ビジネスロジックは持ちません。入力検証や名前正規化等は usecase 層に置きます。
package httphandler

import (
	"encoding/json"
	"net/http"
)

// WriteJSON は JSON レスポンスを書き出します。
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeMethodNotAllowed は HTTPメソッド違反のレスポンスを返します。
func writeMethodNotAllowed(w http.ResponseWriter) {
	WriteAPIError(w, http.StatusMethodNotAllowed, APIError{
		Code:    CodeMethodNotAllowed,
		Message: "Method not allowed",
	})
}

// writeBadRequest は JSON parse 失敗等、リクエスト形式の問題に対する汎用エラーを返します。
// 業務エラー (入力検証失敗等) は usecase 経由で writeUsecaseError を使います。
func writeBadRequest(w http.ResponseWriter, msg string) {
	WriteAPIError(w, http.StatusBadRequest, APIError{
		Code:    CodeBadRequest,
		Message: msg,
	})
}

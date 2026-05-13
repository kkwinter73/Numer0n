package httphandler

import (
	"context"
	"net/http"
	"time"
)

// HealthChecker は外部依存(DB、Redis等)の死活確認を行うインターフェースです。
// 各依存ごとに実装を追加していきます。
//
// 例: フェーズ2.2 でDBを追加したら、
//
//	type DBHealthChecker struct { db *sql.DB }
//	func (c *DBHealthChecker) Check(ctx context.Context) error {
//	    return c.db.PingContext(ctx)
//	}
type HealthChecker interface {
	// Name は依存先の識別子を返します (例: "database", "redis")
	Name() string
	// Check は依存先の死活を確認します。健全なら nil、問題があれば error。
	Check(ctx context.Context) error
}

// HealthHandler はヘルスチェックエンドポイントを提供します。
type HealthHandler struct {
	checkers []HealthChecker
}

// NewHealthHandler はヘルスチェックハンドラを生成します。
// checkers が空でも問題ありません (どの依存もチェックされない = アプリ自体の生存確認のみ)。
func NewHealthHandler(checkers ...HealthChecker) *HealthHandler {
	return &HealthHandler{checkers: checkers}
}

// healthResponse はヘルスチェックのレスポンス構造です。
type healthResponse struct {
	Status       string             `json:"status"`     // "ok" | "degraded"
	Dependencies []dependencyStatus `json:"dependencies,omitempty"`
}

type dependencyStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`          // "ok" | "fail"
	Error  string `json:"error,omitempty"` // 失敗時のメッセージ
}

// HandleHealth (GET /api/health)
//
// レスポンス例 (全依存OK):
//
//	200 {"status":"ok","dependencies":[{"name":"database","status":"ok"}]}
//
// レスポンス例 (DB断):
//
//	503 {"status":"degraded","dependencies":[{"name":"database","status":"fail","error":"..."}]}
//
// 依存先のチェックは並行ではなく順次。各チェックには 2秒のタイムアウトを設定します。
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	resp := healthResponse{Status: "ok"}
	httpStatus := http.StatusOK

	for _, c := range h.checkers {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		err := c.Check(ctx)
		cancel()

		dep := dependencyStatus{Name: c.Name(), Status: "ok"}
		if err != nil {
			dep.Status = "fail"
			dep.Error = err.Error()
			resp.Status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}
		resp.Dependencies = append(resp.Dependencies, dep)
	}

	WriteJSON(w, httpStatus, resp)
}

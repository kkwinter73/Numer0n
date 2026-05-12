package observability

import (
	"log/slog"
	"net/http"
	"time"
)

// Middleware はHTTPミドルウェア関数型のエイリアスです。
type Middleware func(http.Handler) http.Handler

// Chain は複数のミドルウェアを連結します。
// 引数の順に外側→内側の順で適用されます (最初のミドルウェアが最も外側)。
//
// 例: Chain(A, B, C)(h) は A(B(C(h))) と等価。
// リクエスト処理時の実行順は A → B → C → handler → C → B → A。
func Chain(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		// 内側から包む (逆順に適用)
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

// RequestIDMiddleware は各リクエストにユニークIDを付与し、
// context.Context に保存します。レスポンスヘッダ X-Request-ID にもエコーします。
//
// クライアントが X-Request-ID ヘッダで先に送ってきた場合はそれを尊重します
// (分散システムでトレース継続するため)。
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = NewRequestID()
			}

			w.Header().Set("X-Request-ID", id)
			ctx := ContextWithRequestID(r.Context(), id)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggerMiddleware は context にロガーを埋め込みます。
// 埋め込むロガーは「リクエストID付き」になっており、
// ハンドラ内で `observability.LoggerFromContext(ctx).Info(...)` するだけで
// 自動的にリクエストIDが付くようになります。
func LoggerMiddleware(base *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			id := RequestIDFromContext(ctx)

			// リクエストID付きロガーを生成
			logger := base
			if id != "" {
				logger = base.With(slog.String("request_id", id))
			}

			ctx = ContextWithLogger(ctx, logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AccessLogMiddleware はリクエスト開始・終了をログ出力します。
//
// 出力例:
//
//	level=INFO msg="request started" method=POST path=/api/start
//	level=INFO msg="request completed" method=POST path=/api/start status=200 duration_ms=12
func AccessLogMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := LoggerFromContext(r.Context())
			start := time.Now()

			// 静的ファイル配信のログは抑制 (リクエストが多すぎてノイズになる)
			isAPI := len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api"

			if isAPI {
				logger.Info("request started",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
				)
			}

			// レスポンスステータスをキャプチャするためのラッパー
			rw := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			if isAPI {
				duration := time.Since(start)
				logger.Info("request completed",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", rw.status),
					slog.Int64("duration_ms", duration.Milliseconds()),
				)
			}
		})
	}
}

// RecoverMiddleware はハンドラ内のパニックを捕捉し、
// 500レスポンス + Errorログ を返します。
// プロセスごとクラッシュさせない安全弁。
func RecoverMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger := LoggerFromContext(r.Context())
					logger.Error("panic recovered",
						slog.Any("panic", rec),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)
					http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"内部エラーが発生しました"}}`,
						http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusRecorder は http.ResponseWriter をラップし、書き込まれた status code を記録します。
// AccessLogMiddleware からレスポンスステータスを参照するために使います。
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *statusRecorder) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// Write は WriteHeader が呼ばれていない場合、暗黙的に 200 として記録します
// (net/http の挙動と一致)。
func (rw *statusRecorder) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
		// status はデフォルトで 200 のまま
	}
	return rw.ResponseWriter.Write(b)
}

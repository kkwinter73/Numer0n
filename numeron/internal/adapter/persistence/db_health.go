package persistence

import (
	"context"
	"database/sql"
)

// DBHealthChecker は httphandler.HealthChecker インターフェースを実装します。
//
// 実装上の注意:
//   - httphandler パッケージのインターフェース定義に直接依存させず、
//     構造的型付け (Go の interface satisfaction) で対応します。
//   - これにより persistence → httphandler の循環依存を回避できます。
type DBHealthChecker struct {
	db *sql.DB
}

// NewDBHealthChecker は DB の死活確認用 HealthChecker を生成します。
func NewDBHealthChecker(db *sql.DB) *DBHealthChecker {
	return &DBHealthChecker{db: db}
}

// Name はチェッカーの識別子を返します。
func (c *DBHealthChecker) Name() string {
	return "database"
}

// Check はDBへの ping で死活確認します。
// 呼び出し側 (HealthHandler) でタイムアウト付き context が渡されます。
func (c *DBHealthChecker) Check(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

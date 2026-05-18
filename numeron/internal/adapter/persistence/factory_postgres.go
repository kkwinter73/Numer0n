// `postgres` ビルドタグ時のファクトリ実装。
// DB が利用可能なら PostgresSessionStore、そうでなければ MemorySessionStore を返す。
//go:build postgres

package persistence

import (
	"database/sql"

	"github.com/numeron/numeron/internal/port"
)

// NewSessionRepository は DATABASE_URL の有無で実装を切り替えます。
// db == nil なら メモリ実装、db != nil なら PostgreSQL 実装。
func NewSessionRepository(db *sql.DB) port.SessionRepository {
	if db == nil {
		return NewMemorySessionStore()
	}
	return NewPostgresSessionStore(db)
}

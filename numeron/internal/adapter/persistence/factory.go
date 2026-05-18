// このファイルは通常ビルド時に使われ、メモリ実装を返します。
//
// `postgres` ビルドタグ時には factory_postgres.go の方が使われ、
// DB接続を受け取って Postgres 実装を返すよう挙動が変わります。
//go:build !postgres

package persistence

import (
	"database/sql"

	"github.com/numeron/numeron/internal/port"
)

// NewSessionRepository は SessionRepository の実装を返します。
//
// 通常ビルド (デフォルト):
//   - 引数 db に関わらず常に MemorySessionStore を返す
//   - DB依存 (pgx, sqlcgen 等) がプロジェクトに無くてもビルド可能
//
// `postgres` ビルドタグを付けた場合 (factory_postgres.go が有効):
//   - db != nil なら PostgresSessionStore
//   - db == nil なら MemorySessionStore
func NewSessionRepository(db *sql.DB) port.SessionRepository {
	return NewMemorySessionStore()
}

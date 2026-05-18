package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	// ⚠️ お手元で `go get github.com/jackc/pgx/v5/stdlib` を実行した後、
	// この副作用 import 行のコメントを外してください。
	//
	// pgx の database/sql ドライバを副作用 import で登録します。
	// 実際のクエリ実行は sqlc 生成コード経由で pgx を直接使うが、
	// シンプルな Ping 等は database/sql で十分なので両方使う形になります。
	//
	// Go のバージョンによっては pgx v5 最新版が要求するGoバージョンに引っかかる場合があります:
	//   - Go 1.22 → pgx v5.5.5 を指定: `go get github.com/jackc/pgx/v5/stdlib@v5.5.5`
	//   - Go 1.23 → pgx v5.7.5
	//   - Go 1.25+ → pgx 最新版OK
	//
	// _ "github.com/jackc/pgx/v5/stdlib"
)

// DBConfig はDB接続の設定です。config.Config から必要な値を抽出して渡します。
type DBConfig struct {
	URL             string // postgres://...
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// OpenDB は PostgreSQL に接続し、接続プールを設定して *sql.DB を返します。
//
// 起動時に Ping で疎通確認し、失敗した場合は最大 5回(計約15秒)リトライします。
// これはDocker Composeの起動順 (Goアプリが PostgreSQL より先に起動した場合)
// に対応するためです。
//
// 呼び出し側は終了時に必ず db.Close() を呼ぶこと。
func OpenDB(ctx context.Context, cfg DBConfig) (*sql.DB, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// 接続プール設定
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 起動時に Ping して疎通確認 (リトライあり)
	if err := pingWithRetry(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("DB ping failed: %w", err)
	}

	return db, nil
}

// pingWithRetry は 最大5回、指数バックオフで Ping をリトライします。
// 待機時間: 1秒, 2秒, 4秒, 8秒, 16秒 (合計 約31秒で諦める)
func pingWithRetry(ctx context.Context, db *sql.DB) error {
	const maxAttempts = 5
	wait := 1 * time.Second

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := db.PingContext(pingCtx)
		cancel()

		if err == nil {
			return nil
		}
		lastErr = err

		// 最終試行の後は待たずに失敗
		if i == maxAttempts-1 {
			break
		}

		// ctx がキャンセルされたら即座に終了
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		wait *= 2
	}
	return fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

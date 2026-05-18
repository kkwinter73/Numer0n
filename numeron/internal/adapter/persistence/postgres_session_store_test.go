// PostgreSQL バックエンドの統合テスト。
// testcontainers-go で実際のPostgreSQLコンテナを起動して検証します。
//
// 実行:
//   go test -tags postgres ./internal/adapter/persistence/
//
// 必要なもの:
//   - Docker daemon が起動していること
//   - お手元で以下を実行済みであること:
//       go get github.com/testcontainers/testcontainers-go
//       go get github.com/testcontainers/testcontainers-go/modules/postgres
//       go get github.com/golang-migrate/migrate/v4
//       sqlc generate
//
//go:build postgres

package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/numeron/numeron/internal/domain"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupPostgresContainer は testcontainers で PostgreSQL を起動し、
// マイグレーション適用済みの *sql.DB を返します。
// 失敗時に t.Fatal、t.Cleanup でコンテナを後始末します。
func setupPostgresContainer(ctx context.Context, t *testing.T) *sql.DB {
	t.Helper()

	pgContainer, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		tcpostgres.WithDatabase("numeron_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	// マイグレーション適用
	migrationsPath, err := filepath.Abs("../../../db/migrations")
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		t.Fatalf("migrate.New: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("migrate up: %v", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// =====================================================
// PostgresSessionStore のテスト
// =====================================================

func TestPostgresSessionStore_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	db := setupPostgresContainer(ctx, t)
	store := NewPostgresSessionStore(db)

	// 新規セッション保存
	session := domain.NewSession(domain.Secret{1, 2, 3})
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 取得して内容が一致するか確認
	got, ok, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("session not found after save")
	}
	if got.PlayerSecret.String() != "123" {
		t.Errorf("PlayerSecret = %s, want 123", got.PlayerSecret)
	}
	if got.Status != domain.SessionPlaying {
		t.Errorf("Status = %s, want playing", got.Status)
	}
	if got.Turn != 1 {
		t.Errorf("Turn = %d, want 1", got.Turn)
	}
	if len(got.CpuCandidates) != 720 {
		t.Errorf("CpuCandidates count = %d, want 720", len(got.CpuCandidates))
	}
}

func TestPostgresSessionStore_GetNonExistent(t *testing.T) {
	ctx := context.Background()
	db := setupPostgresContainer(ctx, t)
	store := NewPostgresSessionStore(db)

	_, ok, err := store.Get(ctx, "nonexistent-id")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Error("expected not found, but got session")
	}
}

func TestPostgresSessionStore_UpdateAfterTurn(t *testing.T) {
	ctx := context.Background()
	db := setupPostgresContainer(ctx, t)
	store := NewPostgresSessionStore(db)

	// 初回保存
	session := domain.NewSession(domain.Secret{1, 2, 3})
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	// 1ターン進めて更新
	session.Turn = 2
	session.Logs = append(session.Logs, domain.TurnLog{
		Turn:        1,
		PlayerGuess: "456",
		PlayerEat:   0,
		PlayerBite:  0,
		CpuGuess:    "789",
		CpuEat:      0,
		CpuBite:     0,
	})
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("update Save: %v", err)
	}

	// 取得して状態を確認
	got, ok, err := store.Get(ctx, session.ID)
	if err != nil || !ok {
		t.Fatalf("Get: %v, ok=%v", err, ok)
	}
	if got.Turn != 2 {
		t.Errorf("Turn = %d, want 2", got.Turn)
	}
	if len(got.Logs) != 1 {
		t.Errorf("Logs count = %d, want 1", len(got.Logs))
	}
	if got.Logs[0].PlayerGuess != "456" {
		t.Errorf("Log.PlayerGuess = %s, want 456", got.Logs[0].PlayerGuess)
	}
}

func TestPostgresSessionStore_EndedGame(t *testing.T) {
	// 試合終了状態を保存・取得し、reveal フィールドが正しく入ること
	ctx := context.Background()
	db := setupPostgresContainer(ctx, t)
	store := NewPostgresSessionStore(db)

	session := domain.NewSession(domain.Secret{1, 2, 3})
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// 勝利状態にして再保存
	session.Status = domain.SessionPlayerWin
	session.Logs = append(session.Logs, domain.TurnLog{
		Turn:        1,
		PlayerGuess: session.CpuSecret.String(),
		PlayerEat:   3,
		CpuGuess:    "000",
		CpuEat:      0,
	})
	session.FinalizeReveal()
	if err := store.Save(ctx, session); err != nil {
		t.Fatalf("Save (ended): %v", err)
	}

	// 取得時にrevealが入っていること
	got, ok, _ := store.Get(ctx, session.ID)
	if !ok || !got.IsOver() {
		t.Fatalf("not over after save")
	}
	if got.RevealedYou != "123" {
		t.Errorf("RevealedYou = %s, want 123", got.RevealedYou)
	}
	if got.RevealedCpu == "" {
		t.Errorf("RevealedCpu is empty")
	}
}

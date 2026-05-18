package persistence

import (
	"context"
	"strings"
	"testing"
	"time"
)

// =====================================================
// OpenDB の引数チェック
// =====================================================
//
// 注意: DB を使う本格的なテストは testcontainers + PostgreSQL を使う
// 統合テストとしてフェーズ2.4 で書きます。
// ここでは「URLが空ならエラー」「設定値が反映されている」のような
// DB に接続せずに検証できる項目のみテストします。

func TestOpenDB_emptyURL(t *testing.T) {
	_, err := OpenDB(context.Background(), DBConfig{URL: ""})
	if err == nil {
		t.Fatal("空URLでエラーを期待")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("エラーメッセージにDATABASE_URLが含まれない: %v", err)
	}
}

func TestOpenDB_invalidURL(t *testing.T) {
	// 構文的に不正なURLでも sql.Open は通常成功する (実際の接続は Ping 時)
	// よって、ping失敗 → リトライ→ 諦め のフローが走る。
	// pgx ドライバ未登録 (現状) なら sql.Open 自体が失敗する。
	//
	// ここでは「明らかに接続できないURL」+「短い ctx タイムアウト」で
	// OpenDB が適切に失敗することを確認する。
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := OpenDB(ctx, DBConfig{
		URL:             "postgres://nonexistent:5432/db",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err == nil {
		t.Fatal("到達不能URLでエラーを期待")
	}
	// pgx ドライバが未登録なら "unknown driver" エラー、
	// 登録済みなら "ping failed" エラー、いずれも OK。
}

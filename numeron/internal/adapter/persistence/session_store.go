// Package persistence は永続化層を提供します。
//
// 構成:
//   - メモリ実装: MemorySessionStore, MemoryRoomStore (常駐、デフォルト)
//   - DB実装:    PostgresSessionStore (DATABASE_URL設定時に利用、フェーズ2.4で追加)
//
// 上位層 (usecase / handler) は port.SessionRepository インターフェース経由で
// 使用するため、実装の差し替えに影響を受けません。
package persistence

import (
	"context"
	"sync"

	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/port"
)

// MemorySessionStore はCPU対戦セッションをメモリ上で管理します。
// プロセス再起動でデータは消失します。
//
// 利用シーン:
//   - 開発時に DB を立ち上げたくない場合
//   - 単体テスト (PostgresSessionStore のテストには testcontainers を使う)
type MemorySessionStore struct {
	mu   sync.RWMutex
	data map[string]*domain.Session
}

// コンパイル時に port.SessionRepository を満たすことを保証する。
var _ port.SessionRepository = (*MemorySessionStore)(nil)

// NewMemorySessionStore は空のストアを生成します。
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		data: make(map[string]*domain.Session),
	}
}

// Save はセッションを保存します (既存IDなら上書き)。
// メモリ実装では context は使用しません (即時完了するためキャンセル不要)。
func (s *MemorySessionStore) Save(_ context.Context, session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[session.ID] = session
	return nil
}

// Get はセッションを取得します。
// 存在しない場合は (nil, false, nil)。エラーはI/O失敗等の異常系のみ。
func (s *MemorySessionStore) Get(_ context.Context, id string) (*domain.Session, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.data[id]
	return session, ok, nil
}

// Package persistence は永続化層 (現状はメモリ実装) を提供します。
// フェーズ2でこの層を PostgreSQL 実装に差し替えることを想定しています。
// その際、上位層 (usecase / handler) はインターフェース (port パッケージ) に依存することで
// 実装の差し替えに影響を受けません。
package persistence

import (
	"sync"

	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/port"
)

// MemorySessionStore はCPU対戦セッションをメモリ上で管理します。
// プロセス再起動でデータは消失します。
// フェーズ2でDB実装に差し替え予定。
type MemorySessionStore struct {
	mu   sync.RWMutex
	data map[string]*domain.Session
}

// コンパイル時に port.SessionRepository を満たすことを保証する。
// この行があると、インターフェースから外れたメソッドシグネチャ変更を
// コンパイル時に検出できる (interface satisfaction assertion パターン)。
var _ port.SessionRepository = (*MemorySessionStore)(nil)

// NewMemorySessionStore は空のストアを生成します。
func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		data: make(map[string]*domain.Session),
	}
}

// Save はセッションを保存します (既存IDなら上書き)。
// メモリ実装では常に nil を返します。
func (s *MemorySessionStore) Save(session *domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[session.ID] = session
	return nil
}

// Get はセッションを取得します。
// 存在しない場合は (nil, false, nil)。エラーはI/O失敗等の異常系のみ。
func (s *MemorySessionStore) Get(id string) (*domain.Session, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.data[id]
	return session, ok, nil
}

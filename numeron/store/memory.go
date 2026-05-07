package store

import (
	"numeron/game"
	"sync"
)

// SessionStore はセッションをメモリ上で管理します
type SessionStore struct {
	data map[string]*game.Session
	mu   sync.RWMutex
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		data: make(map[string]*game.Session),
	}
}

func (s *SessionStore) Save(session *game.Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[session.ID] = session
}

func (s *SessionStore) Get(id string) (*game.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.data[id]
	return session, exists
}

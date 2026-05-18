package usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/port"
)

// fakeSessionRepo は port.SessionRepository のテスト用実装。
type fakeSessionRepo struct {
	mu       sync.Mutex
	sessions map[string]*domain.Session

	// テスト用の挙動制御
	saveError error
	getError  error
	saveCalls int
	getCalls  int
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: make(map[string]*domain.Session)}
}

var _ port.SessionRepository = (*fakeSessionRepo)(nil)

func (f *fakeSessionRepo) Save(_ context.Context, s *domain.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saveCalls++
	if f.saveError != nil {
		return f.saveError
	}
	f.sessions[s.ID] = s
	return nil
}

func (f *fakeSessionRepo) Get(_ context.Context, id string) (*domain.Session, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	if f.getError != nil {
		return nil, false, f.getError
	}
	s, ok := f.sessions[id]
	return s, ok, nil
}

// fakeRoomRepo は port.RoomRepository のテスト用実装。
type fakeRoomRepo struct {
	mu    sync.Mutex
	rooms map[string]*domain.Room

	createError error
	joinError   error
	getError    error
	codeCounter int
}

func newFakeRoomRepo() *fakeRoomRepo {
	return &fakeRoomRepo{rooms: make(map[string]*domain.Room)}
}

var _ port.RoomRepository = (*fakeRoomRepo)(nil)

func (f *fakeRoomRepo) CreateRoom(name string) (*domain.Room, *domain.OnlinePlayer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createError != nil {
		return nil, nil, f.createError
	}
	f.codeCounter++
	code := fmt.Sprintf("CODE%02d", f.codeCounter)
	host := &domain.OnlinePlayer{
		Token: fmt.Sprintf("host-token-%d", f.codeCounter),
		Slot:  0,
		Name:  name,
	}
	room := domain.NewRoom(code)
	room.SetHost(host)
	f.rooms[code] = room
	return room, host, nil
}

func (f *fakeRoomRepo) JoinRoom(code, name string) (*domain.Room, *domain.OnlinePlayer, error) {
	f.mu.Lock()
	room, ok := f.rooms[code]
	f.mu.Unlock()
	if f.joinError != nil {
		return nil, nil, f.joinError
	}
	if !ok {
		return nil, nil, fmt.Errorf("該当のルームが見つかりません")
	}
	guest := &domain.OnlinePlayer{
		Token: fmt.Sprintf("guest-token-%s", code),
		Slot:  1,
		Name:  name,
	}
	if err := room.AddPlayer(guest); err != nil {
		return nil, nil, err
	}
	return room, guest, nil
}

func (f *fakeRoomRepo) GetRoom(code string) (*domain.Room, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getError != nil {
		return nil, false, f.getError
	}
	r, ok := f.rooms[code]
	return r, ok, nil
}

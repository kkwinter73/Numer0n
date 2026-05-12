package persistence

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/port"
)

// MemoryRoomStore はオンライン対戦のルームをメモリ上で管理します。
// 30分以上アクティビティのないルームを自動GCします。
type MemoryRoomStore struct {
	mu    sync.Mutex
	rooms map[string]*domain.Room
}

var _ port.RoomRepository = (*MemoryRoomStore)(nil)

// NewMemoryRoomStore は空のストアを生成し、GCループを開始します。
func NewMemoryRoomStore() *MemoryRoomStore {
	rs := &MemoryRoomStore{rooms: make(map[string]*domain.Room)}
	go rs.gcLoop()
	return rs
}

// gcLoop は5分ごとに、30分以上アクティビティのないルームを削除します。
// プロセス終了まで永久に走るため、長寿命プロセスでは問題なし。
// フェーズ2以降で context.Context を受け取って停止可能にする予定。
func (rs *MemoryRoomStore) gcLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-30 * time.Minute)
		rs.mu.Lock()
		for code, room := range rs.rooms {
			if room.IsIdle(cutoff) {
				delete(rs.rooms, code)
			}
		}
		rs.mu.Unlock()
	}
}

// codeAlphabet は紛らわしい文字 (I,1,O,0,U,V) を除外した文字集合。
var codeAlphabet = []byte("ABCDEFGHJKLMNPQRSTWXYZ23456789")

func generateCode() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	out := make([]byte, 6)
	for i, x := range b {
		out[i] = codeAlphabet[int(x)%len(codeAlphabet)]
	}
	return string(out)
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateRoom は新ルームを作成し、ホストプレイヤーを登録します。
func (rs *MemoryRoomStore) CreateRoom(name string) (*domain.Room, *domain.OnlinePlayer, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	var code string
	for i := 0; i < 10; i++ {
		c := generateCode()
		if _, exists := rs.rooms[c]; !exists {
			code = c
			break
		}
	}
	if code == "" {
		return nil, nil, fmt.Errorf("コード生成に失敗しました")
	}

	host := &domain.OnlinePlayer{
		Token:        generateToken(),
		Slot:         0,
		Name:         name,
		LastSeen:     time.Now(),
		WasConnected: true,
	}
	room := domain.NewRoom(code)
	room.SetHost(host)
	rs.rooms[code] = room
	return room, host, nil
}

// JoinRoom はゲストとして既存ルームに参加します。
func (rs *MemoryRoomStore) JoinRoom(code, name string) (*domain.Room, *domain.OnlinePlayer, error) {
	rs.mu.Lock()
	room, ok := rs.rooms[code]
	rs.mu.Unlock()
	if !ok {
		return nil, nil, fmt.Errorf("該当のルームが見つかりません")
	}

	guest := &domain.OnlinePlayer{
		Token:        generateToken(),
		Slot:         1,
		Name:         name,
		LastSeen:     time.Now(),
		WasConnected: true,
	}
	if err := room.AddPlayer(guest); err != nil {
		return nil, nil, err
	}
	return room, guest, nil
}

// GetRoom はコードでルームを取得します。
// 存在しなければ (nil, false, nil)。
// エラーはI/O失敗等の異常系のみ (メモリ実装では常に nil)。
func (rs *MemoryRoomStore) GetRoom(code string) (*domain.Room, bool, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	r, ok := rs.rooms[code]
	return r, ok, nil
}

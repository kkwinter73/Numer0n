package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"numeron/game"
	"sync"
	"time"
)

// フェーズ定義
const (
	PhaseLobby = "lobby" // ホスト待機中（参加者待ち）
	PhaseSetup = "setup" // 両者揃った、秘密の数字を設定中
	PhasePlay  = "play"  // 対戦中（毎ターン双方が予想を提出）
	PhaseEnded = "ended" // 終了
)

// イベントタイプ（フロントエンドが切り替えに使う）
const (
	EvOpponentJoined  = "opponent_joined"
	EvOpponentReady   = "opponent_ready"   // 相手の秘密設定完了
	EvGameStarted     = "game_started"     // 両者秘密設定完了 → 対戦開始
	EvGuessPending    = "guess_pending"    // 自分または相手が予想を提出（待機中）
	EvTurnResolved    = "turn_resolved"    // 双方の予想が揃いターン進行
	EvGameOver        = "game_over"        // 試合終了
	EvOpponentLeft    = "opponent_left"    // 相手切断
	EvOpponentReturn  = "opponent_return"  // 相手が再接続
)

// OnlinePlayer は対戦中の1プレイヤーの状態
type OnlinePlayer struct {
	Token        string
	Slot         int // 0=ホスト, 1=ゲスト
	Name         string
	Secret       []int
	SecretSet    bool
	PendingGuess []int // 当該ターンで提出済みの予想（相手の提出を待ち中）
	LastSeen     time.Time
	WasConnected bool
}

// OnlineEvent はLongPollで配信されるイベント
type OnlineEvent struct {
	ID   int                    `json:"id"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// Room はオンライン対戦の1部屋
type Room struct {
	mu           sync.Mutex
	cond         *sync.Cond
	Code         string
	Phase        string
	Players      [2]*OnlinePlayer
	Turn         int
	Logs         []game.TurnLog
	EndStatus    string // "" / "p0_win" / "p1_win" / "draw"
	Events       []OnlineEvent
	nextEventID  int
	LastActivity time.Time
}

func newRoom(code string) *Room {
	r := &Room{
		Code:         code,
		Phase:        PhaseLobby,
		Turn:         1,
		nextEventID:  1,
		LastActivity: time.Now(),
	}
	r.cond = sync.NewCond(&r.mu)
	return r
}

// addEventLocked はロック取得済みでイベントを追加し、待機中のpoll子を起こす
func (r *Room) addEventLocked(typ string, data map[string]interface{}) {
	r.Events = append(r.Events, OnlineEvent{
		ID:   r.nextEventID,
		Type: typ,
		Data: data,
	})
	r.nextEventID++
	r.LastActivity = time.Now()
	r.cond.Broadcast()
}

func (r *Room) lastEventIDLocked() int {
	if len(r.Events) == 0 {
		return 0
	}
	return r.Events[len(r.Events)-1].ID
}

func (r *Room) eventsAfterLocked(sinceID int) []OnlineEvent {
	out := []OnlineEvent{}
	for _, e := range r.Events {
		if e.ID > sinceID {
			out = append(out, e)
		}
	}
	return out
}

// authPlayerLocked はロック取得済みでトークン認証
func (r *Room) authPlayerLocked(token string) (*OnlinePlayer, bool) {
	for _, p := range r.Players {
		if p != nil && p.Token == token {
			return p, true
		}
	}
	return nil, false
}

// WaitEvents は新イベントが来るまでtimeoutまで待機して返す
func (r *Room) WaitEvents(token string, sinceID int, timeout time.Duration) ([]OnlineEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.authPlayerLocked(token)
	if !ok {
		return nil, fmt.Errorf("認証エラー")
	}
	p.LastSeen = time.Now()

	// 既に新イベントがあれば即返却
	if r.lastEventIDLocked() > sinceID {
		return r.eventsAfterLocked(sinceID), nil
	}

	// timeout後にBroadcastで起こすgoroutine
	timer := time.AfterFunc(timeout, func() {
		r.mu.Lock()
		r.cond.Broadcast()
		r.mu.Unlock()
	})
	defer timer.Stop()

	deadline := time.Now().Add(timeout)
	for r.lastEventIDLocked() <= sinceID {
		if time.Now().After(deadline) {
			return nil, nil // タイムアウト = 空配列で正常終了
		}
		r.cond.Wait()
	}
	return r.eventsAfterLocked(sinceID), nil
}

// SetSecret は秘密を設定。両者揃ったら自動的にPlayフェーズへ移行
func (r *Room) SetSecret(token string, secret []int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.authPlayerLocked(token)
	if !ok {
		return fmt.Errorf("認証エラー")
	}
	if r.Phase != PhaseSetup {
		return fmt.Errorf("いまは秘密設定の段階ではありません")
	}
	if p.SecretSet {
		return fmt.Errorf("既に設定済みです")
	}

	p.Secret = secret
	p.SecretSet = true

	r.addEventLocked(EvOpponentReady, map[string]interface{}{
		"slot": p.Slot,
	})

	// 両者完了？
	if r.Players[0] != nil && r.Players[0].SecretSet &&
		r.Players[1] != nil && r.Players[1].SecretSet {
		r.Phase = PhasePlay
		r.addEventLocked(EvGameStarted, nil)
	}
	return nil
}

// SubmitGuess は予想を提出。両者揃ったら判定してターン進行
func (r *Room) SubmitGuess(token string, guess []int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.authPlayerLocked(token)
	if !ok {
		return fmt.Errorf("認証エラー")
	}
	if r.Phase != PhasePlay {
		return fmt.Errorf("いまは予想を提出する段階ではありません")
	}
	if p.PendingGuess != nil {
		return fmt.Errorf("既に提出済みです (相手の提出を待ってください)")
	}

	p.PendingGuess = guess

	r.addEventLocked(EvGuessPending, map[string]interface{}{
		"slot": p.Slot,
	})

	// 両者揃ったら判定
	if r.Players[0].PendingGuess != nil && r.Players[1].PendingGuess != nil {
		r.resolveTurnLocked()
	}

	return nil
}

func (r *Room) resolveTurnLocked() {
	p0, p1 := r.Players[0], r.Players[1]
	e0, b0 := game.CheckEatBite(p1.Secret, p0.PendingGuess)
	e1, b1 := game.CheckEatBite(p0.Secret, p1.PendingGuess)

	log := game.TurnLog{
		Turn:        r.Turn,
		PlayerGuess: game.FormatSecret(p0.PendingGuess),
		PlayerEat:   e0,
		PlayerBite:  b0,
		CpuGuess:    game.FormatSecret(p1.PendingGuess), // フィールド名は既存の都合上 CPU を流用
		CpuEat:      e1,
		CpuBite:     b1,
	}
	r.Logs = append(r.Logs, log)

	p0.PendingGuess = nil
	p1.PendingGuess = nil

	switch {
	case e0 == 3 && e1 == 3:
		r.EndStatus = "draw"
	case e0 == 3:
		r.EndStatus = "p0_win"
	case e1 == 3:
		r.EndStatus = "p1_win"
	}

	if r.EndStatus != "" {
		r.Phase = PhaseEnded
		r.addEventLocked(EvGameOver, map[string]interface{}{
			"status":    r.EndStatus,
			"p0_secret": game.FormatSecret(p0.Secret),
			"p1_secret": game.FormatSecret(p1.Secret),
		})
	} else {
		r.Turn++
		r.addEventLocked(EvTurnResolved, map[string]interface{}{
			"turn":  r.Turn,
			"log":   log,
		})
	}
}

// RoomSnapshot はクライアントに返す状態スナップショット
type RoomSnapshot struct {
	Code           string         `json:"code"`
	Phase          string         `json:"phase"`
	YourSlot       int            `json:"your_slot"`
	YourName       string         `json:"your_name"`
	OppName        string         `json:"opp_name"`
	YourSecretSet  bool           `json:"your_secret_set"`
	OppSecretSet   bool           `json:"opp_secret_set"`
	YourGuessReady bool           `json:"your_guess_ready"`
	OppGuessReady  bool           `json:"opp_guess_ready"`
	Turn           int            `json:"turn"`
	Logs           []game.TurnLog `json:"logs"`
	EndStatus      string         `json:"end_status,omitempty"`
	YourSecret     string         `json:"your_secret,omitempty"`
	OppSecret      string         `json:"opp_secret,omitempty"`
	LastEventID    int            `json:"last_event_id"`
}

// Snapshot は当該プレイヤー視点の現在の状態を返す
func (r *Room) Snapshot(token string) (*RoomSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.authPlayerLocked(token)
	if !ok {
		return nil, fmt.Errorf("認証エラー")
	}

	me := r.Players[p.Slot]
	var opp *OnlinePlayer
	if me.Slot == 0 {
		opp = r.Players[1]
	} else {
		opp = r.Players[0]
	}

	s := &RoomSnapshot{
		Code:           r.Code,
		Phase:          r.Phase,
		YourSlot:       me.Slot,
		YourName:       me.Name,
		YourSecretSet:  me.SecretSet,
		YourGuessReady: me.PendingGuess != nil,
		Turn:           r.Turn,
		Logs:           append([]game.TurnLog{}, r.Logs...), // nilでなく空配列で返す
		EndStatus:      r.EndStatus,
		LastEventID:    r.lastEventIDLocked(),
	}
	if opp != nil {
		s.OppName = opp.Name
		s.OppSecretSet = opp.SecretSet
		s.OppGuessReady = opp.PendingGuess != nil
	}
	if r.Phase == PhaseEnded {
		s.YourSecret = game.FormatSecret(me.Secret)
		if opp != nil {
			s.OppSecret = game.FormatSecret(opp.Secret)
		}
	}
	return s, nil
}

// ============ RoomStore ============

// RoomStore は全部屋を管理（自動GC付き）
type RoomStore struct {
	mu    sync.Mutex
	rooms map[string]*Room
}

func NewRoomStore() *RoomStore {
	rs := &RoomStore{rooms: make(map[string]*Room)}
	go rs.gcLoop()
	return rs
}

// gcLoop は30分以上アクティビティのないルームを定期削除
func (rs *RoomStore) gcLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cutoff := time.Now().Add(-30 * time.Minute)
		rs.mu.Lock()
		for code, room := range rs.rooms {
			room.mu.Lock()
			stale := room.LastActivity.Before(cutoff)
			room.mu.Unlock()
			if stale {
				delete(rs.rooms, code)
			}
		}
		rs.mu.Unlock()
	}
}

// 紛らわしい文字 (I,1,O,0,U,V) は除外
var codeAlphabet = []byte("ABCDEFGHJKLMNPQRSTWXYZ23456789")

func generateCode() string {
	b := make([]byte, 6)
	rand.Read(b)
	out := make([]byte, 6)
	for i, x := range b {
		out[i] = codeAlphabet[int(x)%len(codeAlphabet)]
	}
	return string(out)
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateRoom は新ルームを作成しホストを登録
func (rs *RoomStore) CreateRoom(name string) (*Room, *OnlinePlayer, error) {
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

	host := &OnlinePlayer{
		Token:        generateToken(),
		Slot:         0,
		Name:         name,
		LastSeen:     time.Now(),
		WasConnected: true,
	}
	room := newRoom(code)
	room.Players[0] = host
	rs.rooms[code] = room
	return room, host, nil
}

// JoinRoom はゲストとして参加
func (rs *RoomStore) JoinRoom(code, name string) (*Room, *OnlinePlayer, error) {
	rs.mu.Lock()
	room, ok := rs.rooms[code]
	rs.mu.Unlock()
	if !ok {
		return nil, nil, fmt.Errorf("該当のルームが見つかりません")
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Phase != PhaseLobby {
		return nil, nil, fmt.Errorf("ルームは満員または対戦中です")
	}
	if room.Players[1] != nil {
		return nil, nil, fmt.Errorf("ルームは満員です")
	}

	guest := &OnlinePlayer{
		Token:        generateToken(),
		Slot:         1,
		Name:         name,
		LastSeen:     time.Now(),
		WasConnected: true,
	}
	room.Players[1] = guest
	room.Phase = PhaseSetup
	room.addEventLocked(EvOpponentJoined, map[string]interface{}{
		"name": name,
	})
	return room, guest, nil
}

// GetRoom はコードでルームを取得
func (rs *RoomStore) GetRoom(code string) (*Room, bool) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	r, ok := rs.rooms[code]
	return r, ok
}

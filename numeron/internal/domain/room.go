package domain

import (
	"fmt"
	"sync"
	"time"
)

// Phase は対戦の進行段階を表します。
type Phase string

const (
	PhaseLobby Phase = "lobby" // ホスト待機中（参加者待ち）
	PhaseSetup Phase = "setup" // 両者揃った、暗証番号を設定中
	PhasePlay  Phase = "play"  // 対戦中（毎ターン双方が予想を提出）
	PhaseEnded Phase = "ended" // 終了
)

// EndStatus は対戦終了時の結果を表します。
type EndStatus string

const (
	EndStatusNone EndStatus = ""       // まだ終わっていない
	EndStatusP0Win EndStatus = "p0_win" // slot 0 (ホスト) の勝ち
	EndStatusP1Win EndStatus = "p1_win" // slot 1 (ゲスト) の勝ち
	EndStatusDraw  EndStatus = "draw"   // 引き分け
)

// EventType はLongPoll/WebSocketで配信されるイベントの種類です。
type EventType string

const (
	EvOpponentJoined EventType = "opponent_joined"
	EvOpponentReady  EventType = "opponent_ready"  // 相手の暗証設定完了
	EvGameStarted    EventType = "game_started"    // 両者暗証設定完了 → 対戦開始
	EvGuessPending   EventType = "guess_pending"   // 自分または相手が予想を提出（待機中）
	EvTurnResolved   EventType = "turn_resolved"   // 双方の予想が揃いターン進行
	EvGameOver       EventType = "game_over"       // 試合終了
	EvOpponentLeft   EventType = "opponent_left"   // 相手切断
	EvOpponentReturn EventType = "opponent_return" // 相手が再接続
)

// OnlinePlayer は対戦中の1プレイヤーの状態を保持します。
type OnlinePlayer struct {
	Token        string
	Slot         int // 0=ホスト, 1=ゲスト
	Name         string
	Secret       Secret
	SecretSet    bool
	PendingGuess Secret // 当該ターンで提出済みの予想（相手の提出を待ち中）
	LastSeen     time.Time
	WasConnected bool
}

// OnlineEvent は対戦中に発生するイベントを表します。
// Data は将来的に型付きペイロード(map ではなく構造体)へ移行予定。
type OnlineEvent struct {
	ID   int                    `json:"id"`
	Type EventType              `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// Room はオンライン対戦の1部屋 (集約ルート) です。
//
// 注意: 並行制御のため sync.Mutex と sync.Cond を内包しています。
// 長期的には「ドメインモデルに sync を持つのは責務違反」ですが、
// 現状の long-poll 実装に必要な最小限の構造として残しています。
// フェーズ3で WebSocket に移行する際に、この同期機構は外部 (hub) へ移します。
type Room struct {
	mu           sync.Mutex
	cond         *sync.Cond
	Code         string
	Phase        Phase
	Players      [2]*OnlinePlayer
	Turn         int
	Logs         []TurnLog
	EndStatus    EndStatus
	Events       []OnlineEvent
	nextEventID  int
	LastActivity time.Time
}

// NewRoom は新規Roomを生成します。Code は外部 (RoomStore) が払い出します。
func NewRoom(code string) *Room {
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

// AddPlayer はゲストプレイヤーを追加します。空きが無ければエラー。
// 既存のホストは Players[0] に居る想定。
func (r *Room) AddPlayer(player *OnlinePlayer) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Phase != PhaseLobby {
		return fmt.Errorf("ルームは満員または対戦中です")
	}
	// 防御的チェック: 通常は Phase != Lobby で先に弾かれるが、
	// 将来 Phase の意味が変わった場合に備えて二重チェック。
	if r.Players[1] != nil {
		return fmt.Errorf("ルームは満員です")
	}
	r.Players[1] = player
	r.Phase = PhaseSetup
	r.addEventLocked(EvOpponentJoined, map[string]interface{}{
		"name": player.Name,
	})
	return nil
}

// SetHost は初回作成時にホストプレイヤーを設定します。
func (r *Room) SetHost(player *OnlinePlayer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Players[0] = player
}

// addEventLocked はロック取得済みでイベントを追加し、待機中のpoll子を起こします。
func (r *Room) addEventLocked(typ EventType, data map[string]interface{}) {
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

// authPlayerLocked はロック取得済みでトークン認証を行います。
func (r *Room) authPlayerLocked(token string) (*OnlinePlayer, bool) {
	for _, p := range r.Players {
		if p != nil && p.Token == token {
			return p, true
		}
	}
	return nil, false
}

// IsIdle は最終アクティビティから指定時間以上経過しているかを返します。
// GC判定に使用されます。
func (r *Room) IsIdle(cutoff time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.LastActivity.Before(cutoff)
}

// WaitEvents は新イベントが来るまで timeout まで待機して返します。
// 既に新イベントがあれば即返却、無ければブロックします。
// timeout 経過時は空配列 + nil error を返します。
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

// SetSecret はプレイヤーの暗証を設定します。両者揃ったら自動的にPlayフェーズへ移行。
func (r *Room) SetSecret(token string, secret Secret) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.authPlayerLocked(token)
	if !ok {
		return fmt.Errorf("認証エラー")
	}
	if r.Phase != PhaseSetup {
		return fmt.Errorf("いまは暗証設定の段階ではありません")
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

// SubmitGuess は予想を提出します。両者揃ったら判定してターン進行。
func (r *Room) SubmitGuess(token string, guess Secret) error {
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
	e0, b0 := CheckEatBite(p1.Secret, p0.PendingGuess)
	e1, b1 := CheckEatBite(p0.Secret, p1.PendingGuess)

	log := TurnLog{
		Turn:        r.Turn,
		PlayerGuess: p0.PendingGuess.String(),
		PlayerEat:   e0,
		PlayerBite:  b0,
		CpuGuess:    p1.PendingGuess.String(), // 旧フィールド名を維持
		CpuEat:      e1,
		CpuBite:     b1,
	}
	r.Logs = append(r.Logs, log)

	p0.PendingGuess = nil
	p1.PendingGuess = nil

	switch {
	case e0 == 3 && e1 == 3:
		r.EndStatus = EndStatusDraw
	case e0 == 3:
		r.EndStatus = EndStatusP0Win
	case e1 == 3:
		r.EndStatus = EndStatusP1Win
	}

	if r.EndStatus != EndStatusNone {
		r.Phase = PhaseEnded
		r.addEventLocked(EvGameOver, map[string]interface{}{
			"status":    string(r.EndStatus),
			"p0_secret": p0.Secret.String(),
			"p1_secret": p1.Secret.String(),
		})
	} else {
		r.Turn++
		r.addEventLocked(EvTurnResolved, map[string]interface{}{
			"turn": r.Turn,
			"log":  log,
		})
	}
}

// RoomSnapshot はクライアントに返す状態スナップショットです。
// プレイヤー視点 (POV) に応じて your_* / opp_* が決まります。
type RoomSnapshot struct {
	Code           string    `json:"code"`
	Phase          Phase     `json:"phase"`
	YourSlot       int       `json:"your_slot"`
	YourName       string    `json:"your_name"`
	OppName        string    `json:"opp_name"`
	YourSecretSet  bool      `json:"your_secret_set"`
	OppSecretSet   bool      `json:"opp_secret_set"`
	YourGuessReady bool      `json:"your_guess_ready"`
	OppGuessReady  bool      `json:"opp_guess_ready"`
	Turn           int       `json:"turn"`
	Logs           []TurnLog `json:"logs"`
	EndStatus      EndStatus `json:"end_status,omitempty"`
	YourSecret     string    `json:"your_secret,omitempty"`
	OppSecret      string    `json:"opp_secret,omitempty"`
	LastEventID    int       `json:"last_event_id"`
}

// Snapshot は当該プレイヤー視点の現在の状態を返します。
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
		Logs:           append([]TurnLog{}, r.Logs...), // nilでなく空配列で返す
		EndStatus:      r.EndStatus,
		LastEventID:    r.lastEventIDLocked(),
	}
	if opp != nil {
		s.OppName = opp.Name
		s.OppSecretSet = opp.SecretSet
		s.OppGuessReady = opp.PendingGuess != nil
	}
	if r.Phase == PhaseEnded {
		s.YourSecret = me.Secret.String()
		if opp != nil {
			s.OppSecret = opp.Secret.String()
		}
	}
	return s, nil
}

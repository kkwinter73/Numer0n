package domain

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// テスト用のヘルパー: ホストと参加者がいる setup フェーズの Room を作る
func setupRoomWithBothPlayers(t *testing.T) (room *Room, hostToken, guestToken string) {
	t.Helper()
	room = NewRoom("TEST01")

	host := &OnlinePlayer{
		Token: "host-token",
		Slot:  0,
		Name:  "Alice",
	}
	room.SetHost(host)

	guest := &OnlinePlayer{
		Token: "guest-token",
		Slot:  1,
		Name:  "Bob",
	}
	if err := room.AddPlayer(guest); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	return room, host.Token, guest.Token
}

// =====================================================
// NewRoom
// =====================================================

func TestNewRoom(t *testing.T) {
	r := NewRoom("ABC123")

	if r.Code != "ABC123" {
		t.Errorf("Code = %q, want ABC123", r.Code)
	}
	if r.Phase != PhaseLobby {
		t.Errorf("Phase = %q, want lobby", r.Phase)
	}
	if r.Turn != 1 {
		t.Errorf("Turn = %d, want 1", r.Turn)
	}
	if r.Players[0] != nil || r.Players[1] != nil {
		t.Errorf("初期状態でプレイヤーが居る: %v %v", r.Players[0], r.Players[1])
	}
	if r.EndStatus != EndStatusNone {
		t.Errorf("EndStatus = %q, want empty", r.EndStatus)
	}
}

// =====================================================
// SetHost / AddPlayer
// =====================================================

func TestRoom_SetHostAndAddPlayer(t *testing.T) {
	r := NewRoom("X")

	host := &OnlinePlayer{Token: "t1", Slot: 0, Name: "A"}
	r.SetHost(host)
	if r.Players[0] != host {
		t.Errorf("ホストが設定されていない")
	}
	if r.Phase != PhaseLobby {
		t.Errorf("ホスト追加後 Phase = %q, want lobby", r.Phase)
	}

	guest := &OnlinePlayer{Token: "t2", Slot: 1, Name: "B"}
	if err := r.AddPlayer(guest); err != nil {
		t.Fatalf("AddPlayer 失敗: %v", err)
	}
	if r.Players[1] != guest {
		t.Errorf("ゲストが設定されていない")
	}
	if r.Phase != PhaseSetup {
		t.Errorf("ゲスト追加後 Phase = %q, want setup", r.Phase)
	}
}

func TestRoom_AddPlayer_alreadyFull(t *testing.T) {
	r, _, _ := setupRoomWithBothPlayers(t)
	extra := &OnlinePlayer{Token: "extra", Slot: 1, Name: "C"}
	err := r.AddPlayer(extra)
	if err == nil {
		t.Fatalf("満員ルームへの追加でエラーを期待したが nil")
	}
	if !strings.Contains(err.Error(), "満員") && !strings.Contains(err.Error(), "対戦中") {
		t.Errorf("エラーメッセージが想定外: %v", err)
	}
}

func TestRoom_AddPlayer_notLobbyPhase(t *testing.T) {
	// setup フェーズで AddPlayer を呼ぶとエラー
	r := NewRoom("X")
	r.SetHost(&OnlinePlayer{Token: "t1", Slot: 0, Name: "A"})
	_ = r.AddPlayer(&OnlinePlayer{Token: "t2", Slot: 1, Name: "B"})
	// 既に setup
	err := r.AddPlayer(&OnlinePlayer{Token: "t3", Slot: 1, Name: "C"})
	if err == nil {
		t.Fatalf("setup フェーズでの AddPlayer はエラーを期待")
	}
}

// =====================================================
// SetSecret
// =====================================================

func TestRoom_SetSecret_normalFlow(t *testing.T) {
	r, hostToken, guestToken := setupRoomWithBothPlayers(t)

	// ホストが暗証設定
	if err := r.SetSecret(hostToken, Secret{1, 2, 3}); err != nil {
		t.Fatalf("ホストの SetSecret 失敗: %v", err)
	}
	if !r.Players[0].SecretSet {
		t.Errorf("ホストの SecretSet が true にならない")
	}
	if r.Phase != PhaseSetup {
		t.Errorf("ホストだけ設定の段階で Phase = %q, want setup", r.Phase)
	}

	// ゲストも暗証設定 → phase が play に
	if err := r.SetSecret(guestToken, Secret{4, 5, 6}); err != nil {
		t.Fatalf("ゲストの SetSecret 失敗: %v", err)
	}
	if r.Phase != PhasePlay {
		t.Errorf("両者設定後 Phase = %q, want play", r.Phase)
	}
}

func TestRoom_SetSecret_errors(t *testing.T) {
	r, hostToken, _ := setupRoomWithBothPlayers(t)

	t.Run("無効なトークン", func(t *testing.T) {
		err := r.SetSecret("invalid", Secret{1, 2, 3})
		if err == nil {
			t.Fatalf("無効トークンでエラーを期待")
		}
	})

	t.Run("重複設定", func(t *testing.T) {
		if err := r.SetSecret(hostToken, Secret{1, 2, 3}); err != nil {
			t.Fatalf("初回 SetSecret 失敗: %v", err)
		}
		err := r.SetSecret(hostToken, Secret{4, 5, 6})
		if err == nil {
			t.Fatalf("重複 SetSecret でエラーを期待")
		}
	})

	t.Run("Phase違い", func(t *testing.T) {
		// lobby フェーズの新Roomで SetSecret しても弾かれる
		newRoom := NewRoom("Y")
		newRoom.SetHost(&OnlinePlayer{Token: "t", Slot: 0, Name: "A"})
		err := newRoom.SetSecret("t", Secret{1, 2, 3})
		if err == nil {
			t.Fatalf("lobby フェーズの SetSecret でエラーを期待")
		}
	})
}

// =====================================================
// SubmitGuess / 状態遷移
// =====================================================

// テスト用ヘルパー: play フェーズの Room を作る (両者の暗証設定済み)
func playingRoom(t *testing.T, hostSecret, guestSecret Secret) (r *Room, hostToken, guestToken string) {
	t.Helper()
	r, hostToken, guestToken = setupRoomWithBothPlayers(t)
	if err := r.SetSecret(hostToken, hostSecret); err != nil {
		t.Fatalf("host SetSecret: %v", err)
	}
	if err := r.SetSecret(guestToken, guestSecret); err != nil {
		t.Fatalf("guest SetSecret: %v", err)
	}
	if r.Phase != PhasePlay {
		t.Fatalf("playingRoom: Phase != play (%q)", r.Phase)
	}
	return r, hostToken, guestToken
}

func TestRoom_SubmitGuess_oneSideOnly(t *testing.T) {
	// ホストだけ提出 → 待機状態
	r, hostToken, _ := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})

	if err := r.SubmitGuess(hostToken, Secret{4, 5, 6}); err != nil {
		t.Fatalf("SubmitGuess 失敗: %v", err)
	}
	if r.Players[0].PendingGuess == nil {
		t.Errorf("ホストの PendingGuess が記録されていない")
	}
	if r.Phase != PhasePlay {
		t.Errorf("片方のみ提出で Phase = %q, want play (待機継続)", r.Phase)
	}
	if r.Turn != 1 {
		t.Errorf("片方のみで Turn が進んだ: %d", r.Turn)
	}
	if len(r.Logs) != 0 {
		t.Errorf("片方のみで Logs が追加された: %d", len(r.Logs))
	}
}

func TestRoom_SubmitGuess_bothSubmitted_noWin(t *testing.T) {
	// 両者提出、勝者なし → turn が進む
	r, hostToken, guestToken := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})

	_ = r.SubmitGuess(hostToken, Secret{7, 8, 9})  // host が 4,5,6 を外す
	_ = r.SubmitGuess(guestToken, Secret{0, 9, 8}) // guest が 1,2,3 を外す

	if r.Phase != PhasePlay {
		t.Errorf("勝者なしで Phase = %q, want play", r.Phase)
	}
	if r.Turn != 2 {
		t.Errorf("ターン進行後 Turn = %d, want 2", r.Turn)
	}
	if len(r.Logs) != 1 {
		t.Errorf("Logs 数 = %d, want 1", len(r.Logs))
	}
	// pendingGuess がクリアされている
	if r.Players[0].PendingGuess != nil || r.Players[1].PendingGuess != nil {
		t.Errorf("ターン解決後も PendingGuess が残っている")
	}
}

func TestRoom_SubmitGuess_hostWins(t *testing.T) {
	// host=123, guest=456。host が 456 を完全一致 → P0Win
	r, hostToken, guestToken := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})

	_ = r.SubmitGuess(hostToken, Secret{4, 5, 6})  // 3 EAT
	_ = r.SubmitGuess(guestToken, Secret{7, 8, 9}) // miss

	if r.Phase != PhaseEnded {
		t.Errorf("Phase = %q, want ended", r.Phase)
	}
	if r.EndStatus != EndStatusP0Win {
		t.Errorf("EndStatus = %q, want p0_win", r.EndStatus)
	}
}

func TestRoom_SubmitGuess_guestWins(t *testing.T) {
	r, hostToken, guestToken := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})

	_ = r.SubmitGuess(hostToken, Secret{7, 8, 9})  // miss
	_ = r.SubmitGuess(guestToken, Secret{1, 2, 3}) // 3 EAT

	if r.EndStatus != EndStatusP1Win {
		t.Errorf("EndStatus = %q, want p1_win", r.EndStatus)
	}
}

func TestRoom_SubmitGuess_draw(t *testing.T) {
	// 両者同時に完全一致 → draw
	r, hostToken, guestToken := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})

	_ = r.SubmitGuess(hostToken, Secret{4, 5, 6})  // 3 EAT
	_ = r.SubmitGuess(guestToken, Secret{1, 2, 3}) // 3 EAT

	if r.EndStatus != EndStatusDraw {
		t.Errorf("EndStatus = %q, want draw", r.EndStatus)
	}
	if r.Phase != PhaseEnded {
		t.Errorf("Phase = %q, want ended", r.Phase)
	}
}

func TestRoom_SubmitGuess_duplicateSubmission(t *testing.T) {
	r, hostToken, _ := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})

	if err := r.SubmitGuess(hostToken, Secret{4, 5, 6}); err != nil {
		t.Fatalf("初回 SubmitGuess: %v", err)
	}
	err := r.SubmitGuess(hostToken, Secret{0, 1, 2})
	if err == nil {
		t.Fatalf("重複提出でエラーを期待")
	}
}

func TestRoom_SubmitGuess_invalidToken(t *testing.T) {
	r, _, _ := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})
	err := r.SubmitGuess("invalid-token", Secret{4, 5, 6})
	if err == nil {
		t.Fatalf("無効トークンでエラーを期待")
	}
}

func TestRoom_SubmitGuess_wrongPhase(t *testing.T) {
	// setup フェーズで SubmitGuess
	r, hostToken, _ := setupRoomWithBothPlayers(t)
	err := r.SubmitGuess(hostToken, Secret{4, 5, 6})
	if err == nil {
		t.Fatalf("setup フェーズの SubmitGuess でエラーを期待")
	}
}

// =====================================================
// Snapshot (POV による表示の切り替え)
// =====================================================

func TestRoom_Snapshot_hostPOV(t *testing.T) {
	r, hostToken, _ := setupRoomWithBothPlayers(t)
	snap, err := r.Snapshot(hostToken)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if snap.YourSlot != 0 {
		t.Errorf("YourSlot = %d, want 0", snap.YourSlot)
	}
	if snap.YourName != "Alice" {
		t.Errorf("YourName = %q, want Alice", snap.YourName)
	}
	if snap.OppName != "Bob" {
		t.Errorf("OppName = %q, want Bob", snap.OppName)
	}
	if snap.Phase != PhaseSetup {
		t.Errorf("Phase = %q, want setup", snap.Phase)
	}
	// プレイ中ではないので暗証は開示されない
	if snap.YourSecret != "" || snap.OppSecret != "" {
		t.Errorf("setup フェーズで Secret 開示: you=%q opp=%q",
			snap.YourSecret, snap.OppSecret)
	}
}

func TestRoom_Snapshot_guestPOV(t *testing.T) {
	r, _, guestToken := setupRoomWithBothPlayers(t)
	snap, err := r.Snapshot(guestToken)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if snap.YourSlot != 1 {
		t.Errorf("YourSlot = %d, want 1", snap.YourSlot)
	}
	if snap.YourName != "Bob" {
		t.Errorf("YourName = %q, want Bob", snap.YourName)
	}
	if snap.OppName != "Alice" {
		t.Errorf("OppName = %q, want Alice", snap.OppName)
	}
}

func TestRoom_Snapshot_endedRevealsSecrets(t *testing.T) {
	// ゲーム終了後、両者の暗証が開示される (POVに応じて)
	r, hostToken, guestToken := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})
	_ = r.SubmitGuess(hostToken, Secret{4, 5, 6})
	_ = r.SubmitGuess(guestToken, Secret{7, 8, 9})

	// host視点
	hSnap, _ := r.Snapshot(hostToken)
	if hSnap.YourSecret != "123" {
		t.Errorf("host YourSecret = %q, want 123", hSnap.YourSecret)
	}
	if hSnap.OppSecret != "456" {
		t.Errorf("host OppSecret = %q, want 456", hSnap.OppSecret)
	}

	// guest視点 (your/opp が反転する)
	gSnap, _ := r.Snapshot(guestToken)
	if gSnap.YourSecret != "456" {
		t.Errorf("guest YourSecret = %q, want 456", gSnap.YourSecret)
	}
	if gSnap.OppSecret != "123" {
		t.Errorf("guest OppSecret = %q, want 123", gSnap.OppSecret)
	}
}

func TestRoom_Snapshot_invalidToken(t *testing.T) {
	r, _, _ := setupRoomWithBothPlayers(t)
	_, err := r.Snapshot("nonexistent")
	if err == nil {
		t.Fatalf("無効トークンでエラーを期待")
	}
}

// =====================================================
// WaitEvents (ロングポーリング)
// =====================================================

func TestRoom_WaitEvents_immediateReturn(t *testing.T) {
	// 既に発火済みのイベントがある場合は即返却
	r, hostToken, _ := setupRoomWithBothPlayers(t)
	// AddPlayer によって opponent_joined イベントが発火している

	start := time.Now()
	events, err := r.WaitEvents(hostToken, 0, 5*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("WaitEvents: %v", err)
	}
	if len(events) == 0 {
		t.Errorf("events 空。opponent_joined を期待")
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("即返却を期待したが %v 待った", elapsed)
	}

	// opponent_joined が含まれているか
	found := false
	for _, e := range events {
		if e.Type == EvOpponentJoined {
			found = true
		}
	}
	if !found {
		t.Errorf("opponent_joined イベントが見つからない")
	}
}

func TestRoom_WaitEvents_timeout(t *testing.T) {
	// 新イベントが来なければタイムアウト
	r, hostToken, _ := setupRoomWithBothPlayers(t)
	// 既存イベントのIDを取得
	snap, _ := r.Snapshot(hostToken)
	currentID := snap.LastEventID

	start := time.Now()
	events, err := r.WaitEvents(hostToken, currentID, 200*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("WaitEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("タイムアウト時は空配列を期待: got %v", events)
	}
	if elapsed < 150*time.Millisecond {
		t.Errorf("タイムアウト前に返ってきた: %v", elapsed)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("タイムアウトが効いていない: %v", elapsed)
	}
}

func TestRoom_WaitEvents_wakesOnNewEvent(t *testing.T) {
	// 別ゴルーチンが SetSecret したとき、待機中の WaitEvents が起きる
	r, hostToken, guestToken := setupRoomWithBothPlayers(t)
	snap, _ := r.Snapshot(hostToken)
	currentID := snap.LastEventID

	// ホスト側で待機開始 (5秒タイムアウト)
	resultCh := make(chan []OnlineEvent, 1)
	go func() {
		events, _ := r.WaitEvents(hostToken, currentID, 5*time.Second)
		resultCh <- events
	}()

	// 100ms 待ってからゲストが暗証設定 → opponent_ready イベント発火
	time.Sleep(100 * time.Millisecond)
	if err := r.SetSecret(guestToken, Secret{4, 5, 6}); err != nil {
		t.Fatalf("SetSecret: %v", err)
	}

	// 1秒以内に events を受信できることを期待
	select {
	case events := <-resultCh:
		if len(events) == 0 {
			t.Errorf("events 空。opponent_ready を期待")
		}
		foundReady := false
		for _, e := range events {
			if e.Type == EvOpponentReady {
				foundReady = true
			}
		}
		if !foundReady {
			t.Errorf("opponent_ready イベントが含まれない: %v", events)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("WaitEvents が起きなかった")
	}
}

func TestRoom_WaitEvents_invalidToken(t *testing.T) {
	r, _, _ := setupRoomWithBothPlayers(t)
	_, err := r.WaitEvents("invalid", 0, 100*time.Millisecond)
	if err == nil {
		t.Fatalf("無効トークンでエラーを期待")
	}
}

// =====================================================
// 並行性: 競合状態でデータ破壊が起きないこと
// =====================================================

func TestRoom_concurrent_SubmitGuess(t *testing.T) {
	// 同じプレイヤーから同時に複数の SubmitGuess が来ても、1回だけ受理されるべき
	r, hostToken, _ := playingRoom(t, Secret{1, 2, 3}, Secret{4, 5, 6})

	const n = 50
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := r.SubmitGuess(hostToken, Secret{7, 8, 9})
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if successCount != 1 {
		t.Errorf("同時 SubmitGuess の成功回数 = %d, want 1", successCount)
	}
}

// =====================================================
// エッジケース (カバレッジ補完)
// =====================================================

func TestRoom_Snapshot_emptyEvents(t *testing.T) {
	// Room作成直後で Events が空のとき、LastEventID は 0
	r := NewRoom("X")
	r.SetHost(&OnlinePlayer{Token: "t1", Slot: 0, Name: "A"})

	snap, err := r.Snapshot("t1")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.LastEventID != 0 {
		t.Errorf("初期 LastEventID = %d, want 0", snap.LastEventID)
	}
}

func TestRoom_IsIdle(t *testing.T) {
	r := NewRoom("X")
	// 作成直後は最新なので idle ではない
	if r.IsIdle(time.Now().Add(-1 * time.Minute)) {
		t.Errorf("作成直後の Room が idle 判定された")
	}
	// 遠い未来から見れば idle
	if !r.IsIdle(time.Now().Add(1 * time.Hour)) {
		t.Errorf("遠い未来から見て idle にならない")
	}
}

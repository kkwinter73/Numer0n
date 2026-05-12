package usecase

import (
	"errors"
	"strings"
	"testing"
)

// =====================================================
// CreateRoom
// =====================================================

func TestOnlineUsecase_CreateRoom_success(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)

	res, err := uc.CreateRoom("Alice")
	if err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if res.Code == "" {
		t.Errorf("Code が空")
	}
	if res.Token == "" {
		t.Errorf("Token が空")
	}
	if res.Slot != 0 {
		t.Errorf("Slot = %d, want 0", res.Slot)
	}
}

func TestOnlineUsecase_CreateRoom_nameNormalized(t *testing.T) {
	// 空白除去・長すぎる名前の切り詰め・空ならデフォルト
	tests := []struct {
		input    string
		expected string
	}{
		{"  Alice  ", "Alice"},
		{"", "PLAYER"},
		{"   ", "PLAYER"},
		{"ABCDEFGHIJKLMNOPQRST", "ABCDEFGHIJKLMNOP"}, // 16文字に切り詰め
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			repo := newFakeRoomRepo()
			uc := NewOnlineUsecase(repo)
			_, err := uc.CreateRoom(tt.input)
			if err != nil {
				t.Fatalf("CreateRoom: %v", err)
			}
			// 内部状態を確認
			for _, room := range repo.rooms {
				if room.Players[0].Name != tt.expected {
					t.Errorf("Name = %q, want %q", room.Players[0].Name, tt.expected)
				}
			}
		})
	}
}

// =====================================================
// JoinRoom
// =====================================================

func TestOnlineUsecase_JoinRoom_success(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)

	host, _ := uc.CreateRoom("Alice")
	guest, err := uc.JoinRoom(host.Code, "Bob")
	if err != nil {
		t.Fatalf("JoinRoom: %v", err)
	}
	if guest.Slot != 1 {
		t.Errorf("Slot = %d, want 1", guest.Slot)
	}
	if guest.OppName != "Alice" {
		t.Errorf("OppName = %q, want Alice", guest.OppName)
	}
}

func TestOnlineUsecase_JoinRoom_codeNormalization(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")

	// 小文字 + 空白 → 大文字化 + trim
	mixedCode := "  " + strings.ToLower(host.Code) + "  "
	_, err := uc.JoinRoom(mixedCode, "Bob")
	if err != nil {
		t.Fatalf("コード正規化が機能していない: %v", err)
	}
}

func TestOnlineUsecase_JoinRoom_emptyCode(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	_, err := uc.JoinRoom("", "Bob")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("空コードで ErrInvalidInput を期待: %v", err)
	}
}

func TestOnlineUsecase_JoinRoom_notFound(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	_, err := uc.JoinRoom("NOTEXIST", "Bob")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("見つからない場合 ErrConflict を期待 (現状の実装): %v", err)
	}
}

func TestOnlineUsecase_JoinRoom_full(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	_, _ = uc.JoinRoom(host.Code, "Bob")

	// 3人目は参加できない
	_, err := uc.JoinRoom(host.Code, "Carol")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("満員で ErrConflict を期待: %v", err)
	}
}

// =====================================================
// GetSnapshot
// =====================================================

func TestOnlineUsecase_GetSnapshot_hostPOV(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")

	snap, err := uc.GetSnapshot(host.Code, host.Token)
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snap.YourSlot != 0 || snap.YourName != "Alice" {
		t.Errorf("YourSlot=%d YourName=%q", snap.YourSlot, snap.YourName)
	}
}

func TestOnlineUsecase_GetSnapshot_roomNotFound(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	_, err := uc.GetSnapshot("NOTEXIST", "any")
	if !errors.Is(err, ErrRoomNotFound) {
		t.Errorf("ErrRoomNotFound を期待: %v", err)
	}
}

func TestOnlineUsecase_GetSnapshot_invalidToken(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")

	_, err := uc.GetSnapshot(host.Code, "invalid-token")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("ErrUnauthorized を期待: %v", err)
	}
}

// =====================================================
// SubmitSecret
// =====================================================

func TestOnlineUsecase_SubmitSecret_success(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	guest, _ := uc.JoinRoom(host.Code, "Bob")

	snap, err := uc.SubmitSecret(host.Code, host.Token, "123")
	if err != nil {
		t.Fatalf("SubmitSecret: %v", err)
	}
	if !snap.YourSecretSet {
		t.Errorf("YourSecretSet = false")
	}

	// guest も設定したら phase = play に
	snap, err = uc.SubmitSecret(host.Code, guest.Token, "456")
	if err != nil {
		t.Fatalf("guest SubmitSecret: %v", err)
	}
	if snap.Phase != "play" {
		t.Errorf("Phase = %q, want play", snap.Phase)
	}
}

func TestOnlineUsecase_SubmitSecret_invalidSecret(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	_, _ = uc.JoinRoom(host.Code, "Bob")

	_, err := uc.SubmitSecret(host.Code, host.Token, "112")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("ErrInvalidInput を期待: %v", err)
	}
}

func TestOnlineUsecase_SubmitSecret_invalidToken(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	_, _ = uc.JoinRoom(host.Code, "Bob")

	_, err := uc.SubmitSecret(host.Code, "invalid", "123")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("ErrUnauthorized を期待: %v", err)
	}
}

func TestOnlineUsecase_SubmitSecret_roomNotFound(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	_, err := uc.SubmitSecret("NOTEXIST", "any", "123")
	if !errors.Is(err, ErrRoomNotFound) {
		t.Errorf("ErrRoomNotFound を期待: %v", err)
	}
}

// =====================================================
// SubmitGuess + 試合終了フロー
// =====================================================

func TestOnlineUsecase_SubmitGuess_drawFlow(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	guest, _ := uc.JoinRoom(host.Code, "Bob")
	_, _ = uc.SubmitSecret(host.Code, host.Token, "123")
	_, _ = uc.SubmitSecret(host.Code, guest.Token, "456")

	// 両者が完全一致 → draw
	_, _ = uc.SubmitGuess(host.Code, host.Token, "456")
	snap, err := uc.SubmitGuess(host.Code, guest.Token, "123")
	if err != nil {
		t.Fatalf("SubmitGuess: %v", err)
	}
	if snap.Phase != "ended" {
		t.Errorf("Phase = %q, want ended", snap.Phase)
	}
	if snap.EndStatus != "draw" {
		t.Errorf("EndStatus = %q, want draw", snap.EndStatus)
	}
}

func TestOnlineUsecase_SubmitGuess_invalidGuess(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	_, _ = uc.JoinRoom(host.Code, "Bob")

	_, err := uc.SubmitGuess(host.Code, host.Token, "112")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("ErrInvalidInput を期待: %v", err)
	}
}

func TestOnlineUsecase_SubmitGuess_wrongPhase(t *testing.T) {
	// setup フェーズで SubmitGuess
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	_, _ = uc.JoinRoom(host.Code, "Bob")

	_, err := uc.SubmitGuess(host.Code, host.Token, "456")
	if !errors.Is(err, ErrConflict) {
		t.Errorf("ErrConflict を期待 (wrong phase): %v", err)
	}
}

// =====================================================
// Poll
// =====================================================

func TestOnlineUsecase_Poll_immediateReturn(t *testing.T) {
	// 既にイベントがある場合は即返却
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	_, _ = uc.JoinRoom(host.Code, "Bob") // これで opponent_joined イベント発火

	res, err := uc.Poll(host.Code, host.Token, 0)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(res.Events) == 0 {
		t.Errorf("events 空。opponent_joined を期待")
	}
	if res.State == nil {
		t.Errorf("state は nil でないことを期待")
	}
}

func TestOnlineUsecase_Poll_roomNotFound(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	_, err := uc.Poll("NOTEXIST", "any", 0)
	if !errors.Is(err, ErrRoomNotFound) {
		t.Errorf("ErrRoomNotFound を期待: %v", err)
	}
}

func TestOnlineUsecase_Poll_invalidToken(t *testing.T) {
	repo := newFakeRoomRepo()
	uc := NewOnlineUsecase(repo)
	host, _ := uc.CreateRoom("Alice")
	_, err := uc.Poll(host.Code, "invalid", 0)
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("ErrUnauthorized を期待: %v", err)
	}
}

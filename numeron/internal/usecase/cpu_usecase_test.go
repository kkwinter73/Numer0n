package usecase

import (
	"errors"
	"fmt"
	"testing"

	"github.com/numeron/numeron/internal/domain"
)

// =====================================================
// StartGame
// =====================================================

func TestCPUUsecase_StartGame_success(t *testing.T) {
	repo := newFakeSessionRepo()
	uc := NewCPUUsecase(repo)

	session, err := uc.StartGame("123")
	if err != nil {
		t.Fatalf("StartGame failed: %v", err)
	}
	if session == nil {
		t.Fatal("session is nil")
	}
	if session.PlayerSecret.String() != "123" {
		t.Errorf("PlayerSecret = %v, want 123", session.PlayerSecret)
	}
	if session.Status != domain.SessionPlaying {
		t.Errorf("Status = %q, want playing", session.Status)
	}
	if repo.saveCalls != 1 {
		t.Errorf("Save calls = %d, want 1", repo.saveCalls)
	}
}

func TestCPUUsecase_StartGame_invalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"too short", "12"},
		{"non-digit", "abc"},
		{"duplicate", "112"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeSessionRepo()
			uc := NewCPUUsecase(repo)
			_, err := uc.StartGame(tt.input)
			if err == nil {
				t.Fatalf("エラーを期待")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("errors.Is(err, ErrInvalidInput) = false, err: %v", err)
			}
			if repo.saveCalls != 0 {
				t.Errorf("不正入力で Save が呼ばれた: %d回", repo.saveCalls)
			}
		})
	}
}

func TestCPUUsecase_StartGame_saveError(t *testing.T) {
	repo := newFakeSessionRepo()
	repo.saveError = fmt.Errorf("DB connection lost")
	uc := NewCPUUsecase(repo)

	_, err := uc.StartGame("123")
	if err == nil {
		t.Fatalf("エラーを期待")
	}
	// ErrInvalidInput では*ない*ことを確認 (システムエラー)
	if errors.Is(err, ErrInvalidInput) {
		t.Errorf("システムエラーが ErrInvalidInput と認識された")
	}
}

// =====================================================
// MakeGuess
// =====================================================

func TestCPUUsecase_MakeGuess_success(t *testing.T) {
	repo := newFakeSessionRepo()
	uc := NewCPUUsecase(repo)

	// セッションを作成
	session, err := uc.StartGame("123")
	if err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	updated, err := uc.MakeGuess(session.ID, "456")
	if err != nil {
		t.Fatalf("MakeGuess: %v", err)
	}
	if len(updated.Logs) != 1 {
		t.Errorf("Logs 数 = %d, want 1", len(updated.Logs))
	}
	if updated.Logs[0].PlayerGuess != "456" {
		t.Errorf("PlayerGuess = %q, want 456", updated.Logs[0].PlayerGuess)
	}
}

func TestCPUUsecase_MakeGuess_sessionNotFound(t *testing.T) {
	repo := newFakeSessionRepo()
	uc := NewCPUUsecase(repo)
	_, err := uc.MakeGuess("nonexistent", "456")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("ErrSessionNotFound を期待。got: %v", err)
	}
}

func TestCPUUsecase_MakeGuess_gameAlreadyOver(t *testing.T) {
	repo := newFakeSessionRepo()
	uc := NewCPUUsecase(repo)

	session, _ := uc.StartGame("123")
	session.Status = domain.SessionPlayerWin
	_ = repo.Save(session)

	_, err := uc.MakeGuess(session.ID, "456")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("終了済みセッションで ErrSessionNotFound を期待。got: %v", err)
	}
}

func TestCPUUsecase_MakeGuess_invalidGuess(t *testing.T) {
	repo := newFakeSessionRepo()
	uc := NewCPUUsecase(repo)
	session, _ := uc.StartGame("123")

	_, err := uc.MakeGuess(session.ID, "112")
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("ErrInvalidInput を期待。got: %v", err)
	}
}

func TestCPUUsecase_MakeGuess_playerWin(t *testing.T) {
	repo := newFakeSessionRepo()
	uc := NewCPUUsecase(repo)

	session, _ := uc.StartGame("123")
	// CpuSecret を固定値にセット (テストの決定性のため)
	session.CpuSecret = domain.Secret{4, 5, 6}
	_ = repo.Save(session)

	// 456 を当てる
	updated, err := uc.MakeGuess(session.ID, "456")
	if err != nil {
		t.Fatalf("MakeGuess: %v", err)
	}

	// player_eat = 3 なので、prayer_win または draw のはず
	if updated.Status != domain.SessionPlayerWin && updated.Status != domain.SessionDraw {
		t.Errorf("Status = %q, want player_win or draw", updated.Status)
	}
	// 終了状態 → reveal フィールドが入っている
	if updated.RevealedYou != "123" {
		t.Errorf("RevealedYou = %q, want 123", updated.RevealedYou)
	}
	if updated.RevealedCpu != "456" {
		t.Errorf("RevealedCpu = %q, want 456", updated.RevealedCpu)
	}
}

func TestCPUUsecase_MakeGuess_getError(t *testing.T) {
	repo := newFakeSessionRepo()
	repo.getError = fmt.Errorf("DB timeout")
	uc := NewCPUUsecase(repo)
	_, err := uc.MakeGuess("any", "456")
	if err == nil {
		t.Fatal("エラーを期待")
	}
	if errors.Is(err, ErrSessionNotFound) || errors.Is(err, ErrInvalidInput) {
		t.Errorf("システムエラーが業務エラーと混同された: %v", err)
	}
}

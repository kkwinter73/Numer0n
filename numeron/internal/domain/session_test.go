package domain

import (
	"testing"
)

// =====================================================
// NewSession
// =====================================================

func TestNewSession(t *testing.T) {
	playerSecret := Secret{1, 2, 3}
	s := NewSession(playerSecret)

	// ID は16桁hex
	if len(s.ID) != 16 {
		t.Errorf("ID 長さ = %d, want 16", len(s.ID))
	}

	// PlayerSecret は与えた値
	if s.PlayerSecret.String() != "123" {
		t.Errorf("PlayerSecret = %v, want 123", s.PlayerSecret)
	}

	// CpuSecret は3桁・重複なし
	if len(s.CpuSecret) != SecretLength {
		t.Errorf("CpuSecret 長さ = %d, want %d", len(s.CpuSecret), SecretLength)
	}
	seen := make(map[int]bool)
	for _, d := range s.CpuSecret {
		if seen[d] {
			t.Errorf("CpuSecret に重複: %v", s.CpuSecret)
		}
		seen[d] = true
	}

	// 初期候補数は720
	if len(s.CpuCandidates) != 720 {
		t.Errorf("CpuCandidates 数 = %d, want 720", len(s.CpuCandidates))
	}

	// 初期ターンは1
	if s.Turn != 1 {
		t.Errorf("Turn = %d, want 1", s.Turn)
	}

	// 初期Statusはplaying
	if s.Status != SessionPlaying {
		t.Errorf("Status = %q, want %q", s.Status, SessionPlaying)
	}

	// 初期Logsは空 (nil or 長さ0)
	if len(s.Logs) != 0 {
		t.Errorf("Logs 長さ = %d, want 0", len(s.Logs))
	}

	// プレイ中はRevealedフィールドが空
	if s.RevealedCpu != "" || s.RevealedYou != "" {
		t.Errorf("プレイ中なのに Revealed が空でない: cpu=%q you=%q",
			s.RevealedCpu, s.RevealedYou)
	}
}

func TestNewSession_uniqueIDs(t *testing.T) {
	// 連続生成してID重複が無いこと
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s := NewSession(Secret{1, 2, 3})
		if seen[s.ID] {
			t.Fatalf("ID重複: %s", s.ID)
		}
		seen[s.ID] = true
	}
}

// =====================================================
// IsOver
// =====================================================

func TestSession_IsOver(t *testing.T) {
	tests := []struct {
		name   string
		status SessionStatus
		want   bool
	}{
		{name: "playing", status: SessionPlaying, want: false},
		{name: "player_win", status: SessionPlayerWin, want: true},
		{name: "cpu_win", status: SessionCpuWin, want: true},
		{name: "draw", status: SessionDraw, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Session{Status: tt.status}
			if got := s.IsOver(); got != tt.want {
				t.Errorf("IsOver() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =====================================================
// FinalizeReveal
// =====================================================

func TestSession_FinalizeReveal(t *testing.T) {
	s := &Session{
		PlayerSecret: Secret{1, 2, 3},
		CpuSecret:    Secret{9, 8, 7},
	}

	// 初期状態では空
	if s.RevealedYou != "" || s.RevealedCpu != "" {
		t.Fatalf("初期状態で Revealed が設定されている")
	}

	s.FinalizeReveal()

	if s.RevealedYou != "123" {
		t.Errorf("RevealedYou = %q, want 123", s.RevealedYou)
	}
	if s.RevealedCpu != "987" {
		t.Errorf("RevealedCpu = %q, want 987", s.RevealedCpu)
	}
}

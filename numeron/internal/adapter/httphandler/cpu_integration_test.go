package httphandler

import (
	"net/http"
	"strings"
	"testing"
)

// =====================================================
// CPU /api/start
// =====================================================

func TestIntegration_CPU_StartGame_success(t *testing.T) {
	srv := newTestServer(t)

	var resp struct {
		ID     string `json:"id"`
		Turn   int    `json:"turn"`
		Status string `json:"status"`
	}
	r := postJSON(t, srv.URL+"/api/start", map[string]string{"player_secret": "123"}, &resp)
	if r.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", r.StatusCode)
	}
	if resp.Status != "playing" {
		t.Errorf("status = %q, want playing", resp.Status)
	}
	if resp.Turn != 1 {
		t.Errorf("turn = %d, want 1", resp.Turn)
	}
	if len(resp.ID) != 16 {
		t.Errorf("id length = %d, want 16", len(resp.ID))
	}
}

func TestIntegration_CPU_StartGame_errors(t *testing.T) {
	tests := []struct {
		name       string
		body       interface{}
		wantStatus int
		wantCode   string
	}{
		{"too short", map[string]string{"player_secret": "12"}, 400, CodeInvalidInput},
		{"non-digit", map[string]string{"player_secret": "abc"}, 400, CodeInvalidInput},
		{"duplicate digit", map[string]string{"player_secret": "112"}, 400, CodeInvalidInput},
		{"empty secret", map[string]string{"player_secret": ""}, 400, CodeInvalidInput},
	}
	srv := newTestServer(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := postJSON(t, srv.URL+"/api/start", tt.body, nil)
			assertErrorResponse(t, r, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestIntegration_CPU_StartGame_invalidMethod(t *testing.T) {
	srv := newTestServer(t)
	r, _ := http.Get(srv.URL + "/api/start")
	assertErrorResponse(t, r, 405, CodeMethodNotAllowed)
}

func TestIntegration_CPU_StartGame_badJSON(t *testing.T) {
	srv := newTestServer(t)
	r, _ := http.Post(srv.URL+"/api/start", "application/json", strings.NewReader("not json"))
	assertErrorResponse(t, r, 400, CodeBadRequest)
}

// =====================================================
// CPU /api/guess
// =====================================================

func TestIntegration_CPU_Guess_progress(t *testing.T) {
	srv := newTestServer(t)

	// セッション開始
	var start struct {
		ID string `json:"id"`
	}
	postJSON(t, srv.URL+"/api/start", map[string]string{"player_secret": "123"}, &start)

	// 1回 guess
	var guess struct {
		Turn   int    `json:"turn"`
		Status string `json:"status"`
		Logs   []struct {
			PlayerGuess string `json:"player_guess"`
			PlayerEat   int    `json:"player_eat"`
			PlayerBite  int    `json:"player_bite"`
			CpuGuess    string `json:"cpu_guess"`
			CpuEat      int    `json:"cpu_eat"`
			CpuBite     int    `json:"cpu_bite"`
		} `json:"logs"`
	}
	r := postJSON(t, srv.URL+"/api/guess", map[string]string{
		"session_id": start.ID, "guess": "456",
	}, &guess)
	if r.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", r.StatusCode)
	}
	if len(guess.Logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(guess.Logs))
	}
	log := guess.Logs[0]
	if log.PlayerGuess != "456" {
		t.Errorf("player_guess = %q, want 456", log.PlayerGuess)
	}
	if log.PlayerEat+log.PlayerBite > 3 {
		t.Errorf("eat+bite > 3: %d+%d", log.PlayerEat, log.PlayerBite)
	}
	if len(log.CpuGuess) != 3 {
		t.Errorf("cpu_guess length: %d", len(log.CpuGuess))
	}
}

func TestIntegration_CPU_Guess_sessionNotFound(t *testing.T) {
	srv := newTestServer(t)
	r := postJSON(t, srv.URL+"/api/guess", map[string]string{
		"session_id": "nonexistent", "guess": "456",
	}, nil)
	assertErrorResponse(t, r, 400, CodeSessionNotFound)
}

func TestIntegration_CPU_Guess_invalidGuess(t *testing.T) {
	srv := newTestServer(t)
	var start struct {
		ID string `json:"id"`
	}
	postJSON(t, srv.URL+"/api/start", map[string]string{"player_secret": "123"}, &start)
	r := postJSON(t, srv.URL+"/api/guess", map[string]string{
		"session_id": start.ID, "guess": "112",
	}, nil)
	assertErrorResponse(t, r, 400, CodeInvalidInput)
}

func TestIntegration_CPU_Guess_concurrentSessions(t *testing.T) {
	// 並行する複数セッションが独立して動作することを確認
	srv := newTestServer(t)

	var s1, s2 struct {
		ID string `json:"id"`
	}
	postJSON(t, srv.URL+"/api/start", map[string]string{"player_secret": "123"}, &s1)
	postJSON(t, srv.URL+"/api/start", map[string]string{"player_secret": "456"}, &s2)

	if s1.ID == s2.ID {
		t.Fatalf("並行セッションのIDが同一: %s", s1.ID)
	}

	// セッション1で1回、セッション2で2回 guess して、各々のターン番号が独立であることを確認
	var resp1, resp2 struct {
		Turn int `json:"turn"`
	}
	postJSON(t, srv.URL+"/api/guess", map[string]string{"session_id": s1.ID, "guess": "789"}, &resp1)
	postJSON(t, srv.URL+"/api/guess", map[string]string{"session_id": s2.ID, "guess": "789"}, nil)
	postJSON(t, srv.URL+"/api/guess", map[string]string{"session_id": s2.ID, "guess": "012"}, &resp2)

	if resp1.Turn != 2 {
		t.Errorf("session 1 turn = %d, want 2", resp1.Turn)
	}
	if resp2.Turn != 3 {
		t.Errorf("session 2 turn = %d, want 3", resp2.Turn)
	}
}

func TestIntegration_CPU_Guess_invalidMethod(t *testing.T) {
	srv := newTestServer(t)
	r, _ := http.Get(srv.URL + "/api/guess")
	assertErrorResponse(t, r, 405, CodeMethodNotAllowed)
}

func TestIntegration_CPU_Guess_badJSON(t *testing.T) {
	srv := newTestServer(t)
	r, _ := http.Post(srv.URL+"/api/guess", "application/json", strings.NewReader("not json"))
	assertErrorResponse(t, r, 400, CodeBadRequest)
}

func TestIntegration_CPU_Guess_gameEnd_revealsSecrets(t *testing.T) {
	// CPUを倒すまで連続 guess し、終了時に revealed_* が返ることを確認
	srv := newTestServer(t)
	var start struct {
		ID string `json:"id"`
	}
	postJSON(t, srv.URL+"/api/start", map[string]string{"player_secret": "123"}, &start)

	// 候補を網羅的に試して終了させる
	guesses := []string{
		"012", "345", "678", "901", "246", "135", "579", "024",
		"680", "793", "158", "267", "349", "450", "561", "672",
		"783", "894", "905", "016", "127", "238", "340", "405",
		"504", "603", "702", "823", "942", "086",
	}
	var final struct {
		Status      string `json:"status"`
		RevealedCpu string `json:"revealed_cpu"`
		RevealedYou string `json:"revealed_you"`
	}
	ended := false
	for _, g := range guesses {
		r := postJSON(t, srv.URL+"/api/guess", map[string]string{
			"session_id": start.ID, "guess": g,
		}, &final)
		if r.StatusCode != 200 {
			continue
		}
		if final.Status != "playing" {
			ended = true
			break
		}
	}
	if !ended {
		t.Fatalf("ゲームが30ターン以内に終わらなかった (CPU AIが弱すぎる?)")
	}
	if final.RevealedYou != "123" {
		t.Errorf("revealed_you = %q, want 123", final.RevealedYou)
	}
	if len(final.RevealedCpu) != 3 {
		t.Errorf("revealed_cpu length = %d, want 3 (%q)", len(final.RevealedCpu), final.RevealedCpu)
	}
}

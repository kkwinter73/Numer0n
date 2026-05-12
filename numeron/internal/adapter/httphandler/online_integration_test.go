package httphandler

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// オンライン対戦のレスポンス型 (テスト共通)
type createResponse struct {
	Code  string `json:"code"`
	Token string `json:"token"`
	Slot  int    `json:"slot"`
}

type joinResponse struct {
	Code    string `json:"code"`
	Token   string `json:"token"`
	Slot    int    `json:"slot"`
	OppName string `json:"opp_name"`
}

type snapshotResponse struct {
	Code           string `json:"code"`
	Phase          string `json:"phase"`
	YourSlot       int    `json:"your_slot"`
	YourName       string `json:"your_name"`
	OppName        string `json:"opp_name"`
	YourSecretSet  bool   `json:"your_secret_set"`
	OppSecretSet   bool   `json:"opp_secret_set"`
	YourGuessReady bool   `json:"your_guess_ready"`
	OppGuessReady  bool   `json:"opp_guess_ready"`
	Turn           int    `json:"turn"`
	EndStatus      string `json:"end_status"`
	YourSecret     string `json:"your_secret"`
	OppSecret      string `json:"opp_secret"`
	LastEventID    int    `json:"last_event_id"`
	Logs           []struct {
		Turn        int    `json:"turn"`
		PlayerGuess string `json:"player_guess"`
		PlayerEat   int    `json:"player_eat"`
		CpuGuess    string `json:"cpu_guess"`
		CpuEat      int    `json:"cpu_eat"`
	} `json:"logs"`
}

type pollResponseType struct {
	Events []struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
	} `json:"events"`
	State snapshotResponse `json:"state"`
}

// =====================================================
// CreateRoom + JoinRoom
// =====================================================

func TestIntegration_Online_createAndJoin(t *testing.T) {
	srv := newTestServer(t)

	var host createResponse
	r := postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "Alice"}, &host)
	if r.StatusCode != 200 {
		t.Fatalf("create status = %d", r.StatusCode)
	}
	if len(host.Code) != 6 {
		t.Errorf("code length = %d, want 6 (%q)", len(host.Code), host.Code)
	}
	if host.Slot != 0 {
		t.Errorf("host slot = %d, want 0", host.Slot)
	}

	var guest joinResponse
	r = postJSON(t, srv.URL+"/api/online/join", map[string]string{
		"code": host.Code, "name": "Bob",
	}, &guest)
	if r.StatusCode != 200 {
		t.Fatalf("join status = %d", r.StatusCode)
	}
	if guest.Slot != 1 {
		t.Errorf("guest slot = %d, want 1", guest.Slot)
	}
	if guest.OppName != "Alice" {
		t.Errorf("guest opp_name = %q, want Alice", guest.OppName)
	}
}

func TestIntegration_Online_joinFullRoom(t *testing.T) {
	srv := newTestServer(t)

	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "B"}, nil)
	// 3人目は満員
	r := postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "C"}, nil)
	assertErrorResponse(t, r, 400, CodeConflict)
}

func TestIntegration_Online_joinNonExistentRoom(t *testing.T) {
	srv := newTestServer(t)
	r := postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": "NOTEXIST", "name": "X"}, nil)
	assertErrorResponse(t, r, 400, CodeConflict)
}

func TestIntegration_Online_joinEmptyCode(t *testing.T) {
	srv := newTestServer(t)
	r := postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": "", "name": "X"}, nil)
	assertErrorResponse(t, r, 400, CodeInvalidInput)
}

// =====================================================
// GetState (snapshot)
// =====================================================

func TestIntegration_Online_state_hostPOV(t *testing.T) {
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "Alice"}, &host)
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "Bob"}, nil)

	var snap snapshotResponse
	getJSON(t, fmt.Sprintf("%s/api/online/state?code=%s&token=%s", srv.URL, host.Code, host.Token), &snap)

	if snap.YourSlot != 0 {
		t.Errorf("YourSlot = %d, want 0", snap.YourSlot)
	}
	if snap.YourName != "Alice" || snap.OppName != "Bob" {
		t.Errorf("your=%q opp=%q, want Alice/Bob", snap.YourName, snap.OppName)
	}
	if snap.Phase != "setup" {
		t.Errorf("phase = %q, want setup", snap.Phase)
	}
}

func TestIntegration_Online_state_invalidToken(t *testing.T) {
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	r, _ := http.Get(fmt.Sprintf("%s/api/online/state?code=%s&token=invalid", srv.URL, host.Code))
	assertErrorResponse(t, r, 401, CodeUnauthorized)
}

func TestIntegration_Online_state_roomNotFound(t *testing.T) {
	srv := newTestServer(t)
	r, _ := http.Get(srv.URL + "/api/online/state?code=NOTFOUND&token=any")
	assertErrorResponse(t, r, 404, CodeRoomNotFound)
}

// =====================================================
// SubmitSecret + 全マッチフロー
// =====================================================

func TestIntegration_Online_fullMatch_hostWins(t *testing.T) {
	// 完全なオンライン対戦フロー: create → join → secret×2 → guess×2 → ended
	srv := newTestServer(t)

	// セットアップ
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "Alice"}, &host)
	var guest joinResponse
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "Bob"}, &guest)

	// 暗証設定 (両者)
	var snap snapshotResponse
	r := postJSON(t, srv.URL+"/api/online/secret", map[string]string{
		"code": host.Code, "token": host.Token, "secret": "123",
	}, &snap)
	if r.StatusCode != 200 {
		t.Fatalf("host secret status = %d", r.StatusCode)
	}
	if snap.Phase != "setup" {
		t.Errorf("片方のみ設定で phase = %q, want setup", snap.Phase)
	}

	r = postJSON(t, srv.URL+"/api/online/secret", map[string]string{
		"code": host.Code, "token": guest.Token, "secret": "456",
	}, &snap)
	if r.StatusCode != 200 {
		t.Fatalf("guest secret status = %d", r.StatusCode)
	}
	if snap.Phase != "play" {
		t.Errorf("両者設定後 phase = %q, want play", snap.Phase)
	}

	// host が完璧に当てる (456)、guest は外す (789)
	r = postJSON(t, srv.URL+"/api/online/guess", map[string]string{
		"code": host.Code, "token": host.Token, "guess": "456",
	}, &snap)
	if r.StatusCode != 200 {
		t.Fatalf("host guess status = %d", r.StatusCode)
	}
	// guest はまだ submit していないので待ち状態
	if snap.Phase != "play" {
		t.Errorf("片方のみ guess で phase = %q, want play", snap.Phase)
	}
	if !snap.YourGuessReady {
		t.Errorf("YourGuessReady = false, want true")
	}

	r = postJSON(t, srv.URL+"/api/online/guess", map[string]string{
		"code": host.Code, "token": guest.Token, "guess": "789",
	}, &snap)
	if r.StatusCode != 200 {
		t.Fatalf("guest guess status = %d", r.StatusCode)
	}
	if snap.Phase != "ended" {
		t.Errorf("phase = %q, want ended", snap.Phase)
	}
	if snap.EndStatus != "p0_win" {
		t.Errorf("end_status = %q, want p0_win", snap.EndStatus)
	}

	// host視点の暗証開示
	getJSON(t, fmt.Sprintf("%s/api/online/state?code=%s&token=%s",
		srv.URL, host.Code, host.Token), &snap)
	if snap.YourSecret != "123" || snap.OppSecret != "456" {
		t.Errorf("host POV: your=%q opp=%q, want 123/456", snap.YourSecret, snap.OppSecret)
	}

	// guest視点 (your/opp が反転)
	getJSON(t, fmt.Sprintf("%s/api/online/state?code=%s&token=%s",
		srv.URL, host.Code, guest.Token), &snap)
	if snap.YourSecret != "456" || snap.OppSecret != "123" {
		t.Errorf("guest POV: your=%q opp=%q, want 456/123", snap.YourSecret, snap.OppSecret)
	}
}

func TestIntegration_Online_fullMatch_draw(t *testing.T) {
	// 同時完全一致 → draw
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	var guest joinResponse
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "B"}, &guest)

	postJSON(t, srv.URL+"/api/online/secret",
		map[string]string{"code": host.Code, "token": host.Token, "secret": "123"}, nil)
	postJSON(t, srv.URL+"/api/online/secret",
		map[string]string{"code": host.Code, "token": guest.Token, "secret": "456"}, nil)

	// 両者完全一致 → draw
	postJSON(t, srv.URL+"/api/online/guess",
		map[string]string{"code": host.Code, "token": host.Token, "guess": "456"}, nil)
	var final snapshotResponse
	postJSON(t, srv.URL+"/api/online/guess",
		map[string]string{"code": host.Code, "token": guest.Token, "guess": "123"}, &final)

	if final.EndStatus != "draw" {
		t.Errorf("end_status = %q, want draw", final.EndStatus)
	}
}

func TestIntegration_Online_secret_invalidInput(t *testing.T) {
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "B"}, nil)

	r := postJSON(t, srv.URL+"/api/online/secret", map[string]string{
		"code": host.Code, "token": host.Token, "secret": "112",
	}, nil)
	assertErrorResponse(t, r, 400, CodeInvalidInput)
}

func TestIntegration_Online_secret_invalidToken(t *testing.T) {
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "B"}, nil)

	r := postJSON(t, srv.URL+"/api/online/secret", map[string]string{
		"code": host.Code, "token": "invalid", "secret": "123",
	}, nil)
	assertErrorResponse(t, r, 401, CodeUnauthorized)
}

func TestIntegration_Online_secret_roomNotFound(t *testing.T) {
	srv := newTestServer(t)
	r := postJSON(t, srv.URL+"/api/online/secret", map[string]string{
		"code": "NOTFOUND", "token": "any", "secret": "123",
	}, nil)
	assertErrorResponse(t, r, 404, CodeRoomNotFound)
}

func TestIntegration_Online_guess_wrongPhase(t *testing.T) {
	// setup フェーズで guess
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "B"}, nil)

	r := postJSON(t, srv.URL+"/api/online/guess", map[string]string{
		"code": host.Code, "token": host.Token, "guess": "456",
	}, nil)
	assertErrorResponse(t, r, 400, CodeConflict)
}

// =====================================================
// Long-Poll
// =====================================================

func TestIntegration_Online_poll_immediateReturn(t *testing.T) {
	// 既存イベントがある場合、long-poll は即返却
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "B"}, nil)
	// opponent_joined イベントが既に発火している

	start := time.Now()
	var pr pollResponseType
	getJSON(t, fmt.Sprintf("%s/api/online/poll?code=%s&token=%s&since=0",
		srv.URL, host.Code, host.Token), &pr)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("即返却を期待したが %v 待った", elapsed)
	}
	if len(pr.Events) == 0 {
		t.Errorf("events 空。opponent_joined を期待")
	}
	hasJoinedEvent := false
	for _, e := range pr.Events {
		if e.Type == "opponent_joined" {
			hasJoinedEvent = true
			break
		}
	}
	if !hasJoinedEvent {
		t.Errorf("opponent_joined イベントが含まれない: %v", pr.Events)
	}
}

func TestIntegration_Online_poll_wakesOnEvent(t *testing.T) {
	// 別ゴルーチンが SubmitSecret したとき、待機中の poll が起きる
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)
	var guest joinResponse
	postJSON(t, srv.URL+"/api/online/join", map[string]string{"code": host.Code, "name": "B"}, &guest)

	// 既存イベント数を確認 (opponent_joined が1)
	var pr pollResponseType
	getJSON(t, fmt.Sprintf("%s/api/online/poll?code=%s&token=%s&since=0",
		srv.URL, host.Code, host.Token), &pr)
	currentID := pr.State.LastEventID

	// ホスト側で待機開始
	var wg sync.WaitGroup
	wg.Add(1)
	var waitElapsed time.Duration
	var waitResult pollResponseType
	go func() {
		defer wg.Done()
		start := time.Now()
		getJSON(t, fmt.Sprintf("%s/api/online/poll?code=%s&token=%s&since=%d",
			srv.URL, host.Code, host.Token, currentID), &waitResult)
		waitElapsed = time.Since(start)
	}()

	// 100ms後にゲストが暗証設定 → opponent_ready イベント発火
	time.Sleep(100 * time.Millisecond)
	postJSON(t, srv.URL+"/api/online/secret", map[string]string{
		"code": host.Code, "token": guest.Token, "secret": "456",
	}, nil)

	// ホスト側のpollが起きるのを待つ
	wg.Wait()

	if waitElapsed > 2*time.Second {
		t.Errorf("poll の起床が遅すぎる: %v", waitElapsed)
	}
	hasReady := false
	for _, e := range waitResult.Events {
		if e.Type == "opponent_ready" {
			hasReady = true
			break
		}
	}
	if !hasReady {
		t.Errorf("opponent_ready イベントが含まれない: %v", waitResult.Events)
	}
}

func TestIntegration_Online_poll_roomNotFound(t *testing.T) {
	srv := newTestServer(t)
	r, _ := http.Get(srv.URL + "/api/online/poll?code=NOTFOUND&token=any&since=0")
	assertErrorResponse(t, r, 404, CodeRoomNotFound)
}

func TestIntegration_Online_poll_invalidToken(t *testing.T) {
	srv := newTestServer(t)
	var host createResponse
	postJSON(t, srv.URL+"/api/online/create", map[string]string{"name": "A"}, &host)

	r, _ := http.Get(fmt.Sprintf("%s/api/online/poll?code=%s&token=invalid&since=0",
		srv.URL, host.Code))
	assertErrorResponse(t, r, 401, CodeUnauthorized)
}

// =====================================================
// 各エンドポイントの HTTP メソッド違反 + JSON parse 失敗
// (構造は cpu_handler と同じだが、リグレッション検出のため網羅)
// =====================================================

func TestIntegration_Online_endpoints_invalidMethod(t *testing.T) {
	srv := newTestServer(t)
	endpoints := []string{
		"/api/online/create",
		"/api/online/join",
		"/api/online/secret",
		"/api/online/guess",
	}
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			r, _ := http.Get(srv.URL + ep)
			assertErrorResponse(t, r, 405, CodeMethodNotAllowed)
		})
	}
}

func TestIntegration_Online_endpoints_badJSON(t *testing.T) {
	srv := newTestServer(t)
	endpoints := []string{
		"/api/online/create",
		"/api/online/join",
		"/api/online/secret",
		"/api/online/guess",
	}
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			r, _ := http.Post(srv.URL+ep, "application/json",
				strings.NewReader("not valid json"))
			assertErrorResponse(t, r, 400, CodeBadRequest)
		})
	}
}

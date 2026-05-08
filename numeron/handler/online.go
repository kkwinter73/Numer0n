package handler

import (
	"encoding/json"
	"net/http"
	"numeron/game"
	"numeron/store"
	"strconv"
	"strings"
	"time"
)

// OnlineHandler はオンライン対戦の各エンドポイントをまとめたハンドラ
type OnlineHandler struct {
	rooms *store.RoomStore
}

func NewOnlineHandler(rs *store.RoomStore) *OnlineHandler {
	return &OnlineHandler{rooms: rs}
}

// 共通ユーティリティ ===========================================

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if len(name) > 16 {
		name = name[:16]
	}
	if name == "" {
		name = "PLAYER"
	}
	return name
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// ============ POST /api/online/create ============

type createReq struct {
	Name string `json:"name"`
}
type createRes struct {
	Code  string `json:"code"`
	Token string `json:"token"`
	Slot  int    `json:"slot"`
}

func (h *OnlineHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	room, player, err := h.rooms.CreateRoom(sanitizeName(req.Name))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, createRes{
		Code:  room.Code,
		Token: player.Token,
		Slot:  player.Slot,
	})
}

// ============ POST /api/online/join ============

type joinReq struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
type joinRes struct {
	Code    string `json:"code"`
	Token   string `json:"token"`
	Slot    int    `json:"slot"`
	OppName string `json:"opp_name"`
}

func (h *OnlineHandler) HandleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req joinReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	code := normalizeCode(req.Code)
	if code == "" {
		writeError(w, http.StatusBadRequest, "ルームコードを入力してください")
		return
	}
	room, player, err := h.rooms.JoinRoom(code, sanitizeName(req.Name))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	res := joinRes{
		Code:  room.Code,
		Token: player.Token,
		Slot:  player.Slot,
	}
	if room.Players[0] != nil {
		res.OppName = room.Players[0].Name
	}
	writeJSON(w, http.StatusOK, res)
}

// ============ GET /api/online/state ============

func (h *OnlineHandler) HandleState(w http.ResponseWriter, r *http.Request) {
	code := normalizeCode(r.URL.Query().Get("code"))
	token := r.URL.Query().Get("token")
	room, ok := h.rooms.GetRoom(code)
	if !ok {
		writeError(w, http.StatusNotFound, "ルームが見つかりません")
		return
	}
	snap, err := room.Snapshot(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// ============ POST /api/online/secret ============

type secretReq struct {
	Code   string `json:"code"`
	Token  string `json:"token"`
	Secret string `json:"secret"`
}

func (h *OnlineHandler) HandleSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req secretReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	secret, err := game.ParseInput(req.Secret)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, ok := h.rooms.GetRoom(normalizeCode(req.Code))
	if !ok {
		writeError(w, http.StatusNotFound, "ルームが見つかりません")
		return
	}
	if err := room.SetSecret(req.Token, secret); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	snap, _ := room.Snapshot(req.Token)
	writeJSON(w, http.StatusOK, snap)
}

// ============ POST /api/online/guess ============

type guessReq struct {
	Code  string `json:"code"`
	Token string `json:"token"`
	Guess string `json:"guess"`
}

func (h *OnlineHandler) HandleGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var req guessReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	guess, err := game.ParseInput(req.Guess)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	room, ok := h.rooms.GetRoom(normalizeCode(req.Code))
	if !ok {
		writeError(w, http.StatusNotFound, "ルームが見つかりません")
		return
	}
	if err := room.SubmitGuess(req.Token, guess); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	snap, _ := room.Snapshot(req.Token)
	writeJSON(w, http.StatusOK, snap)
}

// ============ GET /api/online/poll ============

type pollRes struct {
	Events  []store.OnlineEvent  `json:"events"`
	State   *store.RoomSnapshot  `json:"state"`
}

func (h *OnlineHandler) HandlePoll(w http.ResponseWriter, r *http.Request) {
	code := normalizeCode(r.URL.Query().Get("code"))
	token := r.URL.Query().Get("token")
	since, _ := strconv.Atoi(r.URL.Query().Get("since"))

	room, ok := h.rooms.GetRoom(code)
	if !ok {
		writeError(w, http.StatusNotFound, "ルームが見つかりません")
		return
	}

	// 最大 25秒待機（クライアントは30秒タイムアウトより短く設定）
	events, err := room.WaitEvents(token, since, 25*time.Second)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	// 同じレスポンスに最新状態スナップショットも同梱（クライアントが楽になる）
	snap, _ := room.Snapshot(token)
	if events == nil {
		events = []store.OnlineEvent{}
	}
	writeJSON(w, http.StatusOK, pollRes{
		Events: events,
		State:  snap,
	})
}

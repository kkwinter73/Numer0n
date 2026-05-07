package handler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"numeron/game"
	"numeron/store"
)

// APIHandler はHTTPリクエストを受け取り、ゲームの進行を管理
type APIHandler struct {
	store *store.SessionStore
}

func NewAPIHandler(store *store.SessionStore) *APIHandler {
	return &APIHandler{store: store}
}

// StartRequest はゲーム開始時のリクエスト形式
type StartRequest struct {
	PlayerSecret string `json:"player_secret"`
}

// GuessRequest は予想送信時のリクエスト形式
type GuessRequest struct {
	SessionID string `json:"session_id"`
	Guess     string `json:"guess"`
}

// HandleStart は新しいゲームの初期化を行います (/api/start)
func (h *APIHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// リクエスト(JSON)の読み取り
	var req StartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 入力値の検証
	secret, err := game.ParseInput(req.PlayerSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 新しいセッション(ゲーム状態)の作成
	session := &game.Session{
		ID:            game.GenerateID(),
		PlayerSecret:  secret,
		CpuSecret:     game.GenerateRandomSecret(),
		CpuCandidates: game.GenerateAllCandidates(), // 720通りの初期候補
		Turn:          1,
		Status:        "playing",
	}

	// セッションを保存し、クライアントに返す
	h.store.Save(session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// HandleGuess は1ターンごとの予想と判定処理を行います (/api/guess)
func (h *APIHandler) HandleGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// リクエスト(JSON)の読み取り
	var req GuessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// セッションの取得と状態確認
	session, exists := h.store.Get(req.SessionID)
	if !exists || session.Status != "playing" {
		http.Error(w, "Session not found or game is over", http.StatusBadRequest)
		return
	}

	// プレイヤー入力の検証
	playerGuess, err := game.ParseInput(req.Guess)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// プレイヤーの予想を判定
	pEat, pBite := game.CheckEatBite(session.CpuSecret, playerGuess)

	// CPUのターン（候補からランダムに予想）
	guessIndex := rand.Intn(len(session.CpuCandidates))
	cpuGuess := session.CpuCandidates[guessIndex]
	cEat, cBite := game.CheckEatBite(session.PlayerSecret, cpuGuess)

	// 今回の結果を元にCPUの候補を絞り込む
	session.CpuCandidates = game.FilterCandidates(session.CpuCandidates, cpuGuess, cEat, cBite)

	// ターンの結果を履歴に追加
	session.Logs = append(session.Logs, game.TurnLog{
		Turn:        session.Turn,
		PlayerGuess: req.Guess,
		PlayerEat:   pEat,
		PlayerBite:  pBite,
		CpuGuess:    fmt.Sprintf("%d%d%d", cpuGuess[0], cpuGuess[1], cpuGuess[2]),
		CpuEat:      cEat,
		CpuBite:     cBite,
	})

	// 勝敗判定とターン更新
	if pEat == 3 && cEat == 3 {
		session.Status = "draw"
	} else if pEat == 3 {
		session.Status = "player_win"
	} else if cEat == 3 {
		session.Status = "cpu_win"
	} else {
		session.Turn++
	}

	// 試合終了時はコードを開示する
	if session.Status != "playing" {
		session.RevealedCpu = game.FormatSecret(session.CpuSecret)
		session.RevealedYou = game.FormatSecret(session.PlayerSecret)
	}

	// 状態を更新して保存し、クライアントに返す
	h.store.Save(session)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

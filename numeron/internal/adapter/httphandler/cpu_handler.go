package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/numeron/numeron/internal/usecase"
)

// CPUHandler は CPU対戦HTTPエンドポイントを提供します。
// ビジネスロジックは usecase に委譲し、ここでは
// JSON 変換と HTTP ステータスコードへのマッピングのみを行います。
type CPUHandler struct {
	uc *usecase.CPUUsecase
}

func NewCPUHandler(uc *usecase.CPUUsecase) *CPUHandler {
	return &CPUHandler{uc: uc}
}

type startRequest struct {
	PlayerSecret string `json:"player_secret"`
}

type guessRequest struct {
	SessionID string `json:"session_id"`
	Guess     string `json:"guess"`
}

// HandleStart (POST /api/start)
func (h *CPUHandler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req startRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid JSON")
		return
	}

	session, err := h.uc.StartGame(req.PlayerSecret)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, session)
}

// HandleGuess (POST /api/guess)
func (h *CPUHandler) HandleGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req guessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid JSON")
		return
	}

	session, err := h.uc.MakeGuess(req.SessionID, req.Guess)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, session)
}

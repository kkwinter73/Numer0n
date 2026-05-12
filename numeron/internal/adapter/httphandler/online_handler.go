package httphandler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/usecase"
)

// OnlineHandler はオンライン対戦HTTPエンドポイントを提供します。
type OnlineHandler struct {
	uc *usecase.OnlineUsecase
}

func NewOnlineHandler(uc *usecase.OnlineUsecase) *OnlineHandler {
	return &OnlineHandler{uc: uc}
}

// ---------- POST /api/online/create ----------

type createReq struct {
	Name string `json:"name"`
}

func (h *OnlineHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid JSON")
		return
	}
	res, err := h.uc.CreateRoom(req.Name)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"code":  res.Code,
		"token": res.Token,
		"slot":  res.Slot,
	})
}

// ---------- POST /api/online/join ----------

type joinReq struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (h *OnlineHandler) HandleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	var req joinReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid JSON")
		return
	}
	res, err := h.uc.JoinRoom(req.Code, req.Name)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"code":     res.Code,
		"token":    res.Token,
		"slot":     res.Slot,
		"opp_name": res.OppName,
	})
}

// ---------- GET /api/online/state ----------

func (h *OnlineHandler) HandleState(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token := r.URL.Query().Get("token")
	snap, err := h.uc.GetSnapshot(code, token)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, snap)
}

// ---------- POST /api/online/secret ----------

type secretReq struct {
	Code   string `json:"code"`
	Token  string `json:"token"`
	Secret string `json:"secret"`
}

func (h *OnlineHandler) HandleSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	var req secretReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid JSON")
		return
	}
	snap, err := h.uc.SubmitSecret(req.Code, req.Token, req.Secret)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, snap)
}

// ---------- POST /api/online/guess ----------

type guessReqOnline struct {
	Code  string `json:"code"`
	Token string `json:"token"`
	Guess string `json:"guess"`
}

func (h *OnlineHandler) HandleGuess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	var req guessReqOnline
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBadRequest(w, "Invalid JSON")
		return
	}
	snap, err := h.uc.SubmitGuess(req.Code, req.Token, req.Guess)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, snap)
}

// ---------- GET /api/online/poll ----------

// pollResponse はクライアントとの互換性のために、events/state のキー名を維持します。
type pollResponse struct {
	Events []domain.OnlineEvent `json:"events"`
	State  *domain.RoomSnapshot `json:"state"`
}

func (h *OnlineHandler) HandlePoll(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token := r.URL.Query().Get("token")
	since, _ := strconv.Atoi(r.URL.Query().Get("since"))

	res, err := h.uc.Poll(code, token, since)
	if err != nil {
		writeUsecaseError(r.Context(), w, err)
		return
	}
	WriteJSON(w, http.StatusOK, pollResponse{
		Events: res.Events,
		State:  res.State,
	})
}

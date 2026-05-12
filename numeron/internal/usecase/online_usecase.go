package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/port"
)

// OnlineUsecase はオンライン対戦のビジネスフローを提供します。
type OnlineUsecase struct {
	rooms port.RoomRepository
}

func NewOnlineUsecase(rooms port.RoomRepository) *OnlineUsecase {
	return &OnlineUsecase{rooms: rooms}
}

// CreateRoomResult はルーム作成結果をまとめたDTOです。
type CreateRoomResult struct {
	Code  string
	Token string
	Slot  int
}

// JoinRoomResult はルーム参加結果をまとめたDTOです。
type JoinRoomResult struct {
	Code    string
	Token   string
	Slot    int
	OppName string
}

// PollResult は long-poll の結果をまとめたDTOです。
type PollResult struct {
	Events []domain.OnlineEvent
	State  *domain.RoomSnapshot
}

// pollTimeout は long-poll の最大待機時間です。
// クライアント側タイムアウト (30秒) より短く設定します。
const pollTimeout = 25 * time.Second

// ---------- 入力の正規化 ----------

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

// ---------- CreateRoom ----------

// CreateRoom は新ルームを作成し、作成者をホストとして登録します。
func (u *OnlineUsecase) CreateRoom(name string) (*CreateRoomResult, error) {
	room, player, err := u.rooms.CreateRoom(sanitizeName(name))
	if err != nil {
		return nil, fmt.Errorf("ルーム作成失敗: %w", err)
	}
	return &CreateRoomResult{
		Code:  room.Code,
		Token: player.Token,
		Slot:  player.Slot,
	}, nil
}

// ---------- JoinRoom ----------

// JoinRoom はゲストとして既存ルームに参加します。
//
// エラー:
//   - ErrInvalidInput: コードが空
//   - ErrConflict: ルームが満員、対戦中、見つからない 等
//     (現状の RoomRepository は満員・未発見を同じく "見つからない" 系のエラーで返すため
//     ここでは細分化していない。フェーズ1.5で構造化エラー導入時に整理予定)
func (u *OnlineUsecase) JoinRoom(code, name string) (*JoinRoomResult, error) {
	code = normalizeCode(code)
	if code == "" {
		return nil, fmt.Errorf("%w: ルームコードを入力してください", ErrInvalidInput)
	}
	room, player, err := u.rooms.JoinRoom(code, sanitizeName(name))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	res := &JoinRoomResult{
		Code:  room.Code,
		Token: player.Token,
		Slot:  player.Slot,
	}
	if room.Players[0] != nil {
		res.OppName = room.Players[0].Name
	}
	return res, nil
}

// ---------- GetSnapshot ----------

// GetSnapshot は指定プレイヤー視点のルーム状態を返します。
func (u *OnlineUsecase) GetSnapshot(code, token string) (*domain.RoomSnapshot, error) {
	room, err := u.getRoom(code)
	if err != nil {
		return nil, err
	}
	snap, err := room.Snapshot(token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}
	return snap, nil
}

// ---------- SubmitSecret ----------

// SubmitSecret はプレイヤーの暗証番号を設定します。
// 両者が設定完了したら、内部で自動的に Play フェーズに遷移します。
func (u *OnlineUsecase) SubmitSecret(code, token, secretStr string) (*domain.RoomSnapshot, error) {
	secret, err := domain.ParseSecret(secretStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	room, err := u.getRoom(code)
	if err != nil {
		return nil, err
	}
	if err := room.SetSecret(token, secret); err != nil {
		// SetSecret は認証エラーと業務エラー両方を含む。現状はメッセージで判定するしかない。
		// フェーズ1.5 で domain 層のエラーも構造化する予定。
		if strings.Contains(err.Error(), "認証") {
			return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	snap, _ := room.Snapshot(token)
	return snap, nil
}

// ---------- SubmitGuess ----------

// SubmitGuess は予想を提出します。両者揃ったら自動的にターン進行・勝敗判定が走ります。
func (u *OnlineUsecase) SubmitGuess(code, token, guessStr string) (*domain.RoomSnapshot, error) {
	guess, err := domain.ParseSecret(guessStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	room, err := u.getRoom(code)
	if err != nil {
		return nil, err
	}
	if err := room.SubmitGuess(token, guess); err != nil {
		if strings.Contains(err.Error(), "認証") {
			return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrConflict, err)
	}
	snap, _ := room.Snapshot(token)
	return snap, nil
}

// ---------- Poll (long-poll) ----------

// Poll は新しいイベントが来るまで最大25秒待機し、イベントと最新状態を返します。
// タイムアウト時は events が空配列で正常終了します (エラーではない)。
func (u *OnlineUsecase) Poll(code, token string, sinceEventID int) (*PollResult, error) {
	room, err := u.getRoom(code)
	if err != nil {
		return nil, err
	}
	events, err := room.WaitEvents(token, sinceEventID, pollTimeout)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnauthorized, err)
	}
	snap, _ := room.Snapshot(token)
	if events == nil {
		events = []domain.OnlineEvent{}
	}
	return &PollResult{Events: events, State: snap}, nil
}

// ---------- internal helpers ----------

// getRoom は GetRoom の結果を usecase 規約のエラーに変換します。
func (u *OnlineUsecase) getRoom(code string) (*domain.Room, error) {
	code = normalizeCode(code)
	room, ok, err := u.rooms.GetRoom(code)
	if err != nil {
		return nil, fmt.Errorf("ルーム取得失敗: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("%w: 該当のルームが見つかりません", ErrRoomNotFound)
	}
	return room, nil
}

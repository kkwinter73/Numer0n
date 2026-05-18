package usecase

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/port"
)

// CPUUsecase はCPU対戦のビジネスフローを提供します。
// HTTP・CLI・WebSocket等、複数の入り口から再利用可能です。
type CPUUsecase struct {
	sessions port.SessionRepository
}

func NewCPUUsecase(sessions port.SessionRepository) *CPUUsecase {
	return &CPUUsecase{sessions: sessions}
}

// StartGame は新しいCPU対戦セッションを開始します。
//
// 入力: プレイヤーの暗証番号 (3桁・重複なし)
// 戻り値: 作成されたセッション、またはエラー
//
// エラー:
//   - ErrInvalidInput: 暗証番号が不正
//   - その他: ストレージ I/O エラー (DBダウン等)
func (u *CPUUsecase) StartGame(ctx context.Context, playerSecretStr string) (*domain.Session, error) {
	secret, err := domain.ParseSecret(playerSecretStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	session := domain.NewSession(secret)
	if err := u.sessions.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("セッション保存失敗: %w", err)
	}
	return session, nil
}

// MakeGuess は1ターン分の予想を処理し、ターン進行と勝敗判定を行います。
//
// 入力: セッションID、プレイヤーの予想 (3桁・重複なし)
// 戻り値: 更新後のセッション、またはエラー
//
// エラー:
//   - ErrSessionNotFound: セッションが見つからない、または既に終了している
//   - ErrInvalidInput: 予想が不正
//   - その他: ストレージ I/O エラー
func (u *CPUUsecase) MakeGuess(ctx context.Context, sessionID, guessStr string) (*domain.Session, error) {
	session, ok, err := u.sessions.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("セッション取得失敗: %w", err)
	}
	if !ok || session.IsOver() {
		return nil, fmt.Errorf("%w: セッションが見つからないか、既に終了しています", ErrSessionNotFound)
	}

	playerGuess, err := domain.ParseSecret(guessStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// プレイヤー側の判定
	pEat, pBite := domain.CheckEatBite(session.CpuSecret, playerGuess)

	// CPU側のターン (候補からランダムに選択)
	cpuGuess := session.CpuCandidates[rand.Intn(len(session.CpuCandidates))]
	cEat, cBite := domain.CheckEatBite(session.PlayerSecret, cpuGuess)

	// 候補の絞り込み
	session.CpuCandidates = domain.FilterCandidates(session.CpuCandidates, cpuGuess, cEat, cBite)

	// ターンログ追加
	session.Logs = append(session.Logs, domain.TurnLog{
		Turn:        session.Turn,
		PlayerGuess: guessStr,
		PlayerEat:   pEat,
		PlayerBite:  pBite,
		CpuGuess:    cpuGuess.String(),
		CpuEat:      cEat,
		CpuBite:     cBite,
	})

	// 勝敗判定
	switch {
	case pEat == 3 && cEat == 3:
		session.Status = domain.SessionDraw
	case pEat == 3:
		session.Status = domain.SessionPlayerWin
	case cEat == 3:
		session.Status = domain.SessionCpuWin
	default:
		session.Turn++
	}

	if session.IsOver() {
		session.FinalizeReveal()
	}

	if err := u.sessions.Save(ctx, session); err != nil {
		return nil, fmt.Errorf("セッション保存失敗: %w", err)
	}
	return session, nil
}

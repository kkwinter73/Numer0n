package domain

// SessionStatus はCPU対戦の状態を表します。
type SessionStatus string

const (
	SessionPlaying   SessionStatus = "playing"
	SessionPlayerWin SessionStatus = "player_win"
	SessionCpuWin    SessionStatus = "cpu_win"
	SessionDraw      SessionStatus = "draw"
)

// Session は1つのCPU対戦の状態を保持します。
// JSONタグは既存のフロントエンドAPIとの互換性のために維持しています。
type Session struct {
	ID            string        `json:"id"`
	PlayerSecret  Secret        `json:"-"`
	CpuSecret     Secret        `json:"-"`
	CpuCandidates []Secret      `json:"-"`
	Turn          int           `json:"turn"`
	Status        SessionStatus `json:"status"`
	Logs          []TurnLog     `json:"logs"`

	// 試合終了時のみクライアントに開示するフィールド (omitempty でプレイ中は出ない)
	RevealedCpu string `json:"revealed_cpu,omitempty"`
	RevealedYou string `json:"revealed_you,omitempty"`
}

// NewSession は新規セッションを生成します。
// PlayerSecret はクライアントから受け取った値、CpuSecret と CpuCandidates は内部で生成。
func NewSession(playerSecret Secret) *Session {
	return &Session{
		ID:            GenerateID(),
		PlayerSecret:  playerSecret,
		CpuSecret:     GenerateRandomSecret(),
		CpuCandidates: GenerateAllCandidates(),
		Turn:          1,
		Status:        SessionPlaying,
	}
}

// IsOver はゲームが終了しているかを返します。
func (s *Session) IsOver() bool {
	return s.Status != SessionPlaying
}

// FinalizeReveal は試合終了時に双方のSecretを開示用フィールドへセットします。
func (s *Session) FinalizeReveal() {
	s.RevealedCpu = s.CpuSecret.String()
	s.RevealedYou = s.PlayerSecret.String()
}

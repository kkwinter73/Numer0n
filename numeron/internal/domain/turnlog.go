package domain

// TurnLog は対戦における1ターン分の結果を表します。
// CPU対戦・オンライン対戦の両方で使われます。
//
// 注意: フィールド名は歴史的経緯で player_* / cpu_* となっていますが、
// オンライン対戦では「slot 0 が player_*、slot 1 が cpu_*」という慣習で
// 使われます。フロントエンドが視点に応じて再ラベリングします。
// (フェーズ1の整理対象。後で side0_*/side1_* にリネーム予定)
type TurnLog struct {
	Turn        int    `json:"turn"`
	PlayerGuess string `json:"player_guess"`
	PlayerEat   int    `json:"player_eat"`
	PlayerBite  int    `json:"player_bite"`
	CpuGuess    string `json:"cpu_guess"`
	CpuEat      int    `json:"cpu_eat"`
	CpuBite     int    `json:"cpu_bite"`
}

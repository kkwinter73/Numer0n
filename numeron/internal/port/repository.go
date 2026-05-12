// Package port は domain と adapter の境界となるインターフェースを定義します。
//
// このパッケージの役割:
//   - 上位層 (usecase / handler) が依存する「契約」を定義する
//   - 下位層 (persistence) はこの契約を満たす実装を提供する
//
// これにより、ストレージ実装を差し替えても上位層は影響を受けません:
//   - 現状: メモリ実装 (persistence.MemorySessionStore)
//   - 将来: PostgreSQL実装 (persistence.PostgresSessionStore)
//   - テスト: モック実装
//
// インターフェース設計の指針:
//   - 「使う側のニーズ」で設計する (consumer-driven interfaces)
//   - メソッドは最小限に。「全部入り」インターフェースは作らない
//   - エラーは Go 慣習に従い最後の戻り値で返す
package port

import "github.com/numeron/numeron/internal/domain"

// SessionRepository はCPU対戦セッションの永続化を抽象化します。
//
// 現状はメモリ実装のため Save に error は無いが、将来のDB実装で
// I/O エラーが起きうるため、最初から error を返す設計にします。
// (フェーズ2のDB移行時にシグネチャを変えなくて済む)
type SessionRepository interface {
	// Save はセッションを保存します。既存IDなら上書き。
	Save(session *domain.Session) error

	// Get はセッションを取得します。存在しなければ (nil, false, nil)。
	// エラーは I/O 失敗等の異常系のみ (見つからないことはエラーではない)。
	Get(id string) (*domain.Session, bool, error)
}

// RoomRepository はオンライン対戦ルームの永続化を抽象化します。
type RoomRepository interface {
	// CreateRoom は新ルームを作成し、ホストプレイヤーを登録します。
	CreateRoom(name string) (*domain.Room, *domain.OnlinePlayer, error)

	// JoinRoom はゲストとして既存ルームに参加します。
	// ルームが見つからない、満員、対戦中の場合はエラー。
	JoinRoom(code, name string) (*domain.Room, *domain.OnlinePlayer, error)

	// GetRoom はコードでルームを取得します。
	// 存在しなければ (nil, false, nil)。エラーは I/O 失敗等の異常系のみ。
	GetRoom(code string) (*domain.Room, bool, error)
}

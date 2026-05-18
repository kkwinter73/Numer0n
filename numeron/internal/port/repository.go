// Package port は domain と adapter の境界となるインターフェースを定義します。
//
// このパッケージの役割:
//   - 上位層 (usecase / handler) が依存する「契約」を定義する
//   - 下位層 (persistence) はこの契約を満たす実装を提供する
//
// これにより、ストレージ実装を差し替えても上位層は影響を受けません:
//   - メモリ実装 (persistence.MemorySessionStore)
//   - PostgreSQL実装 (persistence.PostgresSessionStore)
//   - テスト用モック実装
//
// インターフェース設計の指針:
//   - 「使う側のニーズ」で設計する (consumer-driven interfaces)
//   - メソッドは最小限に。「全部入り」インターフェースは作らない
//   - エラーは Go 慣習に従い最後の戻り値で返す
//   - DB操作の可能性がある場合は context.Context を第一引数で受ける
package port

import (
	"context"

	"github.com/numeron/numeron/internal/domain"
)

// SessionRepository はCPU対戦セッションの永続化を抽象化します。
//
// 設計のポイント:
//   - 第一引数に context.Context を取る: DB クエリのキャンセル・タイムアウト・トレース伝播のため
//   - メモリ実装でもシグネチャを揃えることで、上位層は context を意識せずに済む
//
// フェーズ2.4 でメモリ実装 + PostgreSQL実装の2系統が並立し、
// `DATABASE_URL` の有無で main.go が選択します。
type SessionRepository interface {
	// Save はセッションを保存します。既存IDなら上書き。
	Save(ctx context.Context, session *domain.Session) error

	// Get はセッションを取得します。存在しなければ (nil, false, nil)。
	// エラーは I/O 失敗等の異常系のみ (見つからないことはエラーではない)。
	Get(ctx context.Context, id string) (*domain.Session, bool, error)
}

// RoomRepository はオンライン対戦ルームの永続化を抽象化します。
//
// 注意: フェーズ2.4 ではオンライン対戦は **メモリ実装のまま**残します。
// 理由:
//   - `domain.Room` は sync.Cond で long-poll を起こす機構を持っており、
//     プロセスローカルでないと機能しない
//   - 未ログインユーザーのルームを永続化するメリットが薄い
//   - フェーズ3 (WebSocket) でhub に移し、フェーズ4 (認証) で
//     試合結果のみDBに永続化する方が自然
//
// よって RoomRepository は当面 context を受けず、メモリ専用のインターフェースとして維持します。
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

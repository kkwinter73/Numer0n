// Package usecase はアプリケーション固有のビジネスフローを実装します。
//
// 責務:
//   - 入力の正規化と検証
//   - 複数の domain 操作を組み合わせたフローの実装
//   - ストレージ (port) との対話
//
// 依存:
//   - domain (ドメインモデルとルール)
//   - port (リポジトリインターフェース)
//
// 依存しないもの:
//   - HTTP / WebSocket 等の通信プロトコル
//   - JSON / Protocol Buffers 等のシリアライゼーション
//   - 具体的なストレージ実装 (PostgreSQL, Redis, メモリ等)
//
// これにより、同じ usecase を HTTP API・WebSocket・CLI 等の異なる入口から呼び出せます。
package usecase

import "errors"

// 業務エラー (handler 側で errors.Is でチェックして HTTP ステータスにマップする)
var (
	// ErrInvalidInput はユーザー入力が不正なときに返します。
	// 元のエラーメッセージ (例: "3桁で入力してください") を %w で wrap して使います。
	// → handler では 400 Bad Request にマッピング
	ErrInvalidInput = errors.New("invalid input")

	// ErrSessionNotFound はCPU対戦セッションが見つからないか、既に終了しているときに返します。
	// → handler では 400 Bad Request にマッピング (404 でなく 400 なのは旧APIとの互換性のため)
	ErrSessionNotFound = errors.New("session not found or game over")

	// ErrRoomNotFound はオンライン対戦ルームが見つからないときに返します。
	// → handler では 404 Not Found にマッピング
	ErrRoomNotFound = errors.New("room not found")

	// ErrUnauthorized はトークン認証に失敗したときに返します。
	// → handler では 401 Unauthorized にマッピング
	ErrUnauthorized = errors.New("unauthorized")

	// ErrConflict は業務ルール上の競合 (満員、既に提出済み等) で返します。
	// → handler では 400 Bad Request にマッピング
	ErrConflict = errors.New("conflict")
)

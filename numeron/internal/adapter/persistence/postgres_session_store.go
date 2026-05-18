// このファイルはビルドタグ `postgres` 付きでのみコンパイルされます。
// お手元で sqlc generate 実行後、`go build -tags postgres ./cmd/server` でビルドします。
//
// 通常ビルド (`go build ./cmd/server`) では PostgresSessionStore は使えず、
// MemorySessionStore のみが利用可能です。
// これは sqlc 生成コードや github.com/google/uuid 等の外部依存を持つため、
// 「依存を入れていない環境でもビルドできる」状態を維持するためのトレードオフです。
//
// 将来 (フェーズ4 で認証実装後等)、これらが標準依存になった時点でビルドタグを外す予定。
//go:build postgres

package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/numeron/numeron/internal/domain"
	"github.com/numeron/numeron/internal/port"

	// sqlcgen は sqlc generate で自動生成されるパッケージ。
	// お手元で `sqlc generate` を実行すると internal/adapter/persistence/sqlc/ に
	// .go ファイルが生成され、このimportが解決されます。
	sqlcgen "github.com/numeron/numeron/internal/adapter/persistence/sqlc"
)

// PostgresSessionStore はCPU対戦セッションを PostgreSQL で永続化します。
//
// 設計:
//   - sqlc 生成の Queries を内部に持ち、sql文を直接書かない (型安全のため)
//   - ドメインモデル (domain.Session) とDB行 (sqlcgen.CpuSession) の相互変換を担当
//   - cpu_candidates は JSONB として保存・取得 (詰め替えはこの層の責務)
type PostgresSessionStore struct {
	db      *sql.DB
	queries *sqlcgen.Queries
}

var _ port.SessionRepository = (*PostgresSessionStore)(nil)

// NewPostgresSessionStore は PostgreSQL バックエンドのストアを生成します。
func NewPostgresSessionStore(db *sql.DB) *PostgresSessionStore {
	return &PostgresSessionStore{
		db:      db,
		queries: sqlcgen.New(db),
	}
}

// Save はセッションを保存します。
//
// 動作:
//   - 既存セッション (ID が DB に存在) なら UpdateCPUSessionAfterTurn で更新
//   - 新規セッション (ID が DB に無い) なら CreateCPUSession で挿入
//   - その後、最新ターンの記録があれば cpu_session_turns に追加 (新規ターン分のみ)
//
// 注意: 完全に冪等な実装にするには「全ターンを書き直す」のが安全ですが、
// 書き込み量が爆発するので「新ターンの差分のみ追加」とします。
// これは「Save は新規 + 1ターン進めた直後」しか呼ばれない前提に依存します。
// (usecase 層がこの前提を守る)
func (s *PostgresSessionStore) Save(ctx context.Context, session *domain.Session) error {
	id, err := uuid.Parse(session.ID)
	if err != nil {
		// 既存メモリ実装では16桁hex のID を使っているため、UUIDでない可能性あり。
		// DB保存時のみUUIDに変換が必要 → domain.NewSession を UUID生成に変更する別案もある
		// 当面: hex ID なら UUID v5 (固定名前空間)で写像することで決定論的にUUIDを得る
		id = uuid.NewSHA1(uuid.NameSpaceOID, []byte(session.ID))
	}

	// 既存セッションか確認
	existing, err := s.queries.GetCPUSessionByID(ctx, id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("get session: %w", err)
	}
	isNew := errors.Is(err, sql.ErrNoRows)

	candidatesJSON, err := encodeCandidates(session.CpuCandidates)
	if err != nil {
		return fmt.Errorf("encode candidates: %w", err)
	}

	if isNew {
		// 新規作成
		_, err := s.queries.CreateCPUSession(ctx, sqlcgen.CreateCPUSessionParams{
			ID:             id,
			UserID:         uuid.NullUUID{}, // 未ログイン時。フェーズ4 で認証ユーザーIDをセット
			PlayerSecret:   session.PlayerSecret.String(),
			CpuSecret:      session.CpuSecret.String(),
			CpuCandidates:  candidatesJSON,
		})
		if err != nil {
			return fmt.Errorf("create session: %w", err)
		}
	} else {
		// 既存更新 (ターン進行後の状態)
		_, err := s.queries.UpdateCPUSessionAfterTurn(ctx, sqlcgen.UpdateCPUSessionAfterTurnParams{
			ID:            id,
			Turn:          int32(session.Turn),
			Status:        string(session.Status),
			CpuCandidates: candidatesJSON,
		})
		if err != nil {
			return fmt.Errorf("update session: %w", err)
		}

		// 新規ターン分のログを追加 (existing.Turn と session.Turn の差分)
		// session.Logs の最後の1件が「いま追加されたターン」と仮定
		if len(session.Logs) > 0 {
			latest := session.Logs[len(session.Logs)-1]
			// 既に同じターン番号で記録されていれば追加しない (冪等性)
			if int32(latest.Turn) > existing.Turn || (int32(latest.Turn) == existing.Turn && session.IsOver()) {
				if err := s.insertTurn(ctx, id, latest); err != nil {
					return fmt.Errorf("insert turn: %w", err)
				}
			}
		}
	}

	return nil
}

// Get はセッションをDB から取得し、ドメインモデルに変換して返します。
// 存在しない場合は (nil, false, nil)。
func (s *PostgresSessionStore) Get(ctx context.Context, id string) (*domain.Session, bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		// 旧式の hex IDなら UUID v5 写像で取得 (Save と対称)
		uid = uuid.NewSHA1(uuid.NameSpaceOID, []byte(id))
	}

	row, err := s.queries.GetCPUSessionByID(ctx, uid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get session: %w", err)
	}

	// ターンログも取得
	turnRows, err := s.queries.ListCPUSessionTurns(ctx, uid)
	if err != nil {
		return nil, false, fmt.Errorf("list turns: %w", err)
	}

	session, err := rowToSession(row, turnRows, id)
	if err != nil {
		return nil, false, fmt.Errorf("convert to domain: %w", err)
	}
	return session, true, nil
}

// insertTurn は1ターン分の記録をDB に追加します (内部ヘルパー)。
func (s *PostgresSessionStore) insertTurn(ctx context.Context, sessionID uuid.UUID, log domain.TurnLog) error {
	_, err := s.queries.InsertCPUSessionTurn(ctx, sqlcgen.InsertCPUSessionTurnParams{
		SessionID:   sessionID,
		Turn:        int32(log.Turn),
		PlayerGuess: log.PlayerGuess,
		PlayerEat:   int16(log.PlayerEat),
		PlayerBite:  int16(log.PlayerBite),
		CpuGuess:    log.CpuGuess,
		CpuEat:      int16(log.CpuEat),
		CpuBite:     int16(log.CpuBite),
	})
	return err
}

// =====================================================
// ドメインモデル ⇔ DB行 の変換
// =====================================================

// rowToSession は DB から取得した行をドメインモデルに変換します。
// originalID は呼び出し側が指定した元のID (UUIDでない場合の表示用)。
func rowToSession(row sqlcgen.CpuSession, turns []sqlcgen.CpuSessionTurn, originalID string) (*domain.Session, error) {
	playerSecret, err := domain.ParseSecret(row.PlayerSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid player_secret: %w", err)
	}
	cpuSecret, err := domain.ParseSecret(row.CpuSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid cpu_secret: %w", err)
	}
	candidates, err := decodeCandidates(row.CpuCandidates)
	if err != nil {
		return nil, fmt.Errorf("decode candidates: %w", err)
	}

	session := &domain.Session{
		ID:            originalID,
		PlayerSecret:  playerSecret,
		CpuSecret:     cpuSecret,
		CpuCandidates: candidates,
		Turn:          int(row.Turn),
		Status:        domain.SessionStatus(row.Status),
		Logs:          make([]domain.TurnLog, 0, len(turns)),
	}

	for _, t := range turns {
		session.Logs = append(session.Logs, domain.TurnLog{
			Turn:        int(t.Turn),
			PlayerGuess: t.PlayerGuess,
			PlayerEat:   int(t.PlayerEat),
			PlayerBite:  int(t.PlayerBite),
			CpuGuess:    t.CpuGuess,
			CpuEat:      int(t.CpuEat),
			CpuBite:     int(t.CpuBite),
		})
	}

	if session.IsOver() {
		session.FinalizeReveal()
	}
	return session, nil
}

// encodeCandidates は []domain.Secret を JSONB に変換します。
// 形式: [[1,2,3],[4,5,6],...]
func encodeCandidates(candidates []domain.Secret) ([]byte, error) {
	// domain.Secret は []int のエイリアスなので、そのまま json.Marshal できる
	return json.Marshal(candidates)
}

// decodeCandidates は JSONB を []domain.Secret に戻します。
func decodeCandidates(data []byte) ([]domain.Secret, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var raw [][]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := make([]domain.Secret, len(raw))
	for i, r := range raw {
		out[i] = domain.Secret(r)
	}
	return out, nil
}

// _ = time.Now  // (linter satisfier: time imported for future use)
var _ = time.Now

/**
 * サーバーAPIのレスポンス型定義。
 *
 * サーバー側 (Go) のドメインモデルと対応:
 *   - internal/domain/session.go (Session)
 *   - internal/domain/room.go    (RoomSnapshot, OnlineEvent)
 *   - internal/domain/turnlog.go (TurnLog)
 *
 * 注意: 手動でメンテナンスするため、サーバー側を変更したら同期が必要。
 *      将来 OpenAPI / tRPC 等で自動生成する余地あり。
 */

// ----- 共通 -----

/** ターンログ (CPU・オンライン両方で使用) */
export type TurnLog = {
  turn: number;
  player_guess: string;
  player_eat: number;
  player_bite: number;
  cpu_guess: string;
  cpu_eat: number;
  cpu_bite: number;
};

// ----- CPU対戦 -----

export type SessionStatus = "playing" | "player_win" | "cpu_win" | "draw";

export type Session = {
  id: string;
  turn: number;
  status: SessionStatus;
  logs: TurnLog[];
  /** 試合終了時のみ存在 */
  revealed_cpu?: string;
  revealed_you?: string;
};

// ----- オンライン対戦 -----

export type Phase = "lobby" | "setup" | "play" | "ended";

export type EndStatus = "p0_win" | "p1_win" | "draw";

export type RoomSnapshot = {
  code: string;
  phase: Phase;
  your_slot: number;
  your_name: string;
  opp_name: string;
  your_secret_set: boolean;
  opp_secret_set: boolean;
  your_guess_ready: boolean;
  opp_guess_ready: boolean;
  turn: number;
  logs: TurnLog[];
  end_status?: EndStatus;
  /** 試合終了時のみ存在 */
  your_secret?: string;
  opp_secret?: string;
  last_event_id: number;
};

export type OnlineEventType =
  | "opponent_joined"
  | "opponent_ready"
  | "game_started"
  | "guess_pending"
  | "turn_resolved"
  | "game_over"
  | "opponent_left"
  | "opponent_return";

export type OnlineEvent = {
  id: number;
  type: OnlineEventType;
  data?: Record<string, unknown>;
};

export type PollResponse = {
  events: OnlineEvent[];
  state: RoomSnapshot;
};

export type CreateRoomResponse = {
  code: string;
  token: string;
  slot: number;
};

export type JoinRoomResponse = {
  code: string;
  token: string;
  slot: number;
  opp_name: string;
};

// ----- ヘルスチェック -----

export type HealthResponse = {
  status: "ok" | "degraded";
  dependencies?: Array<{
    name: string;
    status: "ok" | "fail";
    error?: string;
  }>;
};

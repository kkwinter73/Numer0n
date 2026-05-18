import { apiGet, apiPost } from "./client";
import type {
  CreateRoomResponse,
  HealthResponse,
  JoinRoomResponse,
  PollResponse,
  RoomSnapshot,
  Session,
} from "./types";

/**
 * Numeron API の型付きクライアント関数群。
 *
 * 使い方:
 *   const session = await numeronApi.startGame("123");
 *   const result = await numeronApi.makeGuess(session.id, "456");
 *
 * TanStack Query との組み合わせ例:
 *   const { data, error } = useQuery({
 *     queryKey: ["health"],
 *     queryFn: () => numeronApi.health(),
 *   });
 */

export const numeronApi = {
  // ----- ヘルスチェック -----

  health(): Promise<HealthResponse> {
    return apiGet<HealthResponse>("/api/health");
  },

  // ----- CPU対戦 -----

  startGame(playerSecret: string): Promise<Session> {
    return apiPost<Session>("/api/start", { player_secret: playerSecret });
  },

  makeGuess(sessionId: string, guess: string): Promise<Session> {
    return apiPost<Session>("/api/guess", {
      session_id: sessionId,
      guess,
    });
  },

  // ----- オンライン対戦 -----

  createRoom(name: string): Promise<CreateRoomResponse> {
    return apiPost<CreateRoomResponse>("/api/online/create", { name });
  },

  joinRoom(code: string, name: string): Promise<JoinRoomResponse> {
    return apiPost<JoinRoomResponse>("/api/online/join", { code, name });
  },

  getState(code: string, token: string): Promise<RoomSnapshot> {
    const params = new URLSearchParams({ code, token });
    return apiGet<RoomSnapshot>(`/api/online/state?${params.toString()}`);
  },

  submitSecret(code: string, token: string, secret: string): Promise<RoomSnapshot> {
    return apiPost<RoomSnapshot>("/api/online/secret", { code, token, secret });
  },

  submitGuess(code: string, token: string, guess: string): Promise<RoomSnapshot> {
    return apiPost<RoomSnapshot>("/api/online/guess", { code, token, guess });
  },

  /**
   * ロングポーリング。
   * since にこれまで受信した最大イベントIDを渡すと、新イベントが来るまで最大25秒待機する。
   * AbortController でキャンセル可能。
   */
  poll(code: string, token: string, since: number, signal?: AbortSignal): Promise<PollResponse> {
    const params = new URLSearchParams({
      code,
      token,
      since: String(since),
    });
    return apiGet<PollResponse>(`/api/online/poll?${params.toString()}`, { signal });
  },
};

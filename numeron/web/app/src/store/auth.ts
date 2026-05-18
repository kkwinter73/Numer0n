import { create } from "zustand";

/**
 * 認証状態の Zustand ストア。
 *
 * 用途:
 *   - 現在ログイン中のユーザー情報
 *   - サーバー認証セッション(Cookie で持つので state は補助情報)
 *
 * 注意:
 *   - 「サーバーから取得した最新ユーザー情報」は TanStack Query で管理する
 *     (キャッシュ・再フェッチ・stale time を自動化したいため)
 *   - Zustand は「クライアント側だけで持つ一時状態」(ログイン中ユーザーのキャッシュ、UI設定等) に使う
 *
 * フェーズ4 で本格実装。現状は足場のみ。
 */

export type User = {
  id: string;
  username: string;
  display_name: string;
  rating: number;
};

type AuthState = {
  user: User | null;
  setUser: (user: User | null) => void;
  clear: () => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  setUser: (user) => set({ user }),
  clear: () => set({ user: null }),
}));

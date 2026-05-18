import { ApiError, ApiErrorCode, type ApiErrorResponse } from "./error";

/**
 * HTTPメソッド
 */
type HttpMethod = "GET" | "POST" | "PUT" | "DELETE" | "PATCH";

/**
 * リクエストオプション
 */
type RequestOptions = {
  method?: HttpMethod;
  body?: unknown;
  signal?: AbortSignal;
  headers?: Record<string, string>;
};

/**
 * APIリクエストを送信する汎用関数。
 *
 * - エラー時は ApiError を throw (HTTP 4xx/5xx + サーバー側構造化エラー)
 * - 成功時は JSON をパースした結果を返す
 * - 開発時は Vite proxy 経由で /api → http://localhost:8080
 *
 * @param path 例: '/api/start'
 * @param options method/body 等
 * @returns パース済みのレスポンスJSON
 */
export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, signal, headers = {} } = options;

  const init: RequestInit = {
    method,
    signal,
    headers: {
      Accept: "application/json",
      ...headers,
    },
  };

  if (body !== undefined) {
    init.headers = {
      ...init.headers,
      "Content-Type": "application/json",
    };
    init.body = JSON.stringify(body);
  }

  const response = await fetch(path, init);
  const text = await response.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      // not JSON
    }
  }

  if (!response.ok) {
    // 構造化エラー形式 (新)
    if (data && typeof data === "object" && "error" in data) {
      const err = (data as ApiErrorResponse).error;
      if (err && typeof err === "object") {
        throw new ApiError(err.code, err.message, response.status, err.details ?? null);
      }
    }
    // 古い形式や JSONでない場合
    const msg = (text || `HTTP ${response.status}`).trim();
    throw new ApiError(ApiErrorCode.Unknown, msg, response.status);
  }

  return data as T;
}

/**
 * 略記用ヘルパー: GET
 */
export function apiGet<T>(path: string, options?: Omit<RequestOptions, "method" | "body">): Promise<T> {
  return apiRequest<T>(path, { ...options, method: "GET" });
}

/**
 * 略記用ヘルパー: POST
 */
export function apiPost<T>(path: string, body: unknown, options?: Omit<RequestOptions, "method" | "body">): Promise<T> {
  return apiRequest<T>(path, { ...options, method: "POST", body });
}

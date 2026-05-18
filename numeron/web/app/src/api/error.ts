/**
 * サーバーから返される構造化エラーレスポンス。
 * 形式: { error: { code: string, message: string, details?: object } }
 *
 * サーバー側の定義: internal/adapter/httphandler/api_error.go
 */
export type ApiErrorResponse = {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
};

/**
 * エラーコード定数。
 * サーバーの httphandler/api_error.go と一致させる。
 */
export const ApiErrorCode = {
  InvalidInput: "INVALID_INPUT",
  SessionNotFound: "SESSION_NOT_FOUND",
  RoomNotFound: "ROOM_NOT_FOUND",
  Unauthorized: "UNAUTHORIZED",
  Conflict: "CONFLICT",
  MethodNotAllowed: "METHOD_NOT_ALLOWED",
  BadRequest: "BAD_REQUEST",
  InternalError: "INTERNAL_ERROR",
  Unknown: "UNKNOWN",
} as const;

export type ApiErrorCodeType = (typeof ApiErrorCode)[keyof typeof ApiErrorCode];

/**
 * API呼び出しでエラーが発生した時に投げられるカスタムエラー。
 * `err.code === ApiErrorCode.InvalidInput` のような型安全な分岐ができる。
 */
export class ApiError extends Error {
  readonly code: string;
  readonly details: Record<string, unknown> | null;
  readonly status: number;

  constructor(code: string, message: string, status: number, details: Record<string, unknown> | null = null) {
    super(message || `HTTP ${status}`);
    this.name = "ApiError";
    this.code = code;
    this.details = details;
    this.status = status;
  }

  /** ユーザー入力エラーか */
  isInvalidInput(): boolean {
    return this.code === ApiErrorCode.InvalidInput;
  }

  /** 認証エラーか (再ログインを促すべき) */
  isUnauthorized(): boolean {
    return this.code === ApiErrorCode.Unauthorized;
  }

  /** リソース未発見 */
  isNotFound(): boolean {
    return this.code === ApiErrorCode.SessionNotFound || this.code === ApiErrorCode.RoomNotFound;
  }
}

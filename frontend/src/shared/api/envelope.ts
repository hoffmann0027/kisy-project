// Mirrors the response envelope defined in docs/spec/09-api-contracts.md
// and produced by backend/pkg/httpresponse.
export interface ApiErrorBody {
  code: string;
  message: string;
  details?: unknown;
}

export interface ApiEnvelope<T> {
  success: boolean;
  data?: T;
  error?: ApiErrorBody;
  requestId: string;
  timestamp: string;
}

/**
 * True only when the server itself refused the session.
 *
 * The distinction that matters: a request can fail because the server said
 * "no", or because it was never asked. Only the first ends a session — the
 * second is a phone with no signal yet, and treating it as a sign-out throws
 * away a session that is still valid.
 */
export function isAuthFailure(err: unknown): boolean {
  return err instanceof ApiError && (err.status === 401 || err.status === 403);
}

export class ApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly requestId: string,
    public readonly status: number = 0,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

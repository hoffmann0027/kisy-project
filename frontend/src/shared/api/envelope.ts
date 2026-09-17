// Mirrors the response envelope defined in docs/spec/09-api-contracts.md
// and produced by backend/pkg/httpresponse.
import { UserFacingError } from "@shared/lib/errors";

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

/**
 * The text to show for a failed request: the server's own wording when it
 * refused on a limit the user can act on (a storage quota, the posting rate —
 * audit A-07), otherwise the caller's generic fallback. Everything else keeps
 * the fallback, so internal messages never reach the screen.
 */
export function userFacingError(err: unknown, fallback: string): string {
  if (err instanceof UserFacingError && err.message) return err.message;
  if (err instanceof ApiError && USER_FACING_CODES.has(err.code) && err.message) return err.message;
  return fallback;
}

// Server refusals whose message is written for the user: a limit they can act
// on (A-07) and a private chat that takes only encrypted text (A-10).
const USER_FACING_CODES = new Set(["QUOTA_EXCEEDED", "E2EE_REQUIRED"]);

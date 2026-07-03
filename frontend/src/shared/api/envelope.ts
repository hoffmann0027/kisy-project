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

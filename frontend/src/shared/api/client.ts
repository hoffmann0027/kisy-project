import { ApiError, type ApiEnvelope } from "./envelope";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });

  // 204 or empty bodies still return a valid (empty) result.
  const text = await response.text();
  const envelope: ApiEnvelope<T> = text ? JSON.parse(text) : ({ success: response.ok } as ApiEnvelope<T>);

  if (!envelope.success || !response.ok) {
    const error = envelope.error;
    throw new ApiError(
      error?.code ?? "INTERNAL_ERROR",
      error?.message ?? "Unexpected error",
      envelope.requestId ?? "",
      response.status,
    );
  }

  return envelope.data as T;
}

function body(b?: unknown): string | undefined {
  return b === undefined ? undefined : JSON.stringify(b);
}

export const apiClient = {
  get: <T>(path: string) => request<T>(path, { method: "GET" }),
  post: <T>(path: string, b?: unknown) => request<T>(path, { method: "POST", body: body(b) }),
  put: <T>(path: string, b?: unknown) => request<T>(path, { method: "PUT", body: body(b) }),
  patch: <T>(path: string, b?: unknown) => request<T>(path, { method: "PATCH", body: body(b) }),
  del: <T>(path: string, b?: unknown) => request<T>(path, { method: "DELETE", body: body(b) }),
};

// Runtime differences between the web app and the packaged mobile app.
//
// In the browser the SPA is served from the same origin as the API, so cookies
// (HttpOnly + SameSite=Strict) carry the session and relative URLs work. Inside
// the Android/iOS shell the bundle is served from the WebView's own origin
// (https://localhost), which makes the API cross-site: those cookies are never
// sent. Native builds therefore talk to an absolute API origin and authenticate
// with a Bearer token, which the backend already accepts.

const TOKEN_KEY = "kisy.native.tokens";

export interface NativeTokens {
  accessToken: string;
  accessExpiresAt: string;
  refreshToken: string;
  refreshExpires: string;
}

/** True when running inside the Capacitor shell rather than a browser tab. */
export function isNative(): boolean {
  const cap = (window as unknown as { Capacitor?: { isNativePlatform?: () => boolean } }).Capacitor;
  return Boolean(cap?.isNativePlatform?.());
}

/**
 * Absolute origin of the API for native builds (VITE_NATIVE_API_ORIGIN, baked
 * in at build time). Empty on the web, where relative paths are correct.
 */
export function apiOrigin(): string {
  if (!isNative()) return "";
  return (import.meta.env.VITE_NATIVE_API_ORIGIN as string | undefined)?.replace(/\/$/, "") ?? "";
}

// Tokens live in localStorage inside the WebView. That store is private to the
// app sandbox — unlike a browser tab there is no cross-site attacker and no
// other origin can read it. (A dedicated secure-storage plugin can replace this
// later without touching call sites.)
export function loadTokens(): NativeTokens | null {
  try {
    const raw = localStorage.getItem(TOKEN_KEY);
    return raw ? (JSON.parse(raw) as NativeTokens) : null;
  } catch {
    return null;
  }
}

export function saveTokens(t: NativeTokens | null): void {
  try {
    if (t) localStorage.setItem(TOKEN_KEY, JSON.stringify(t));
    else localStorage.removeItem(TOKEN_KEY);
  } catch {
    // A full or disabled store must not break sign-in.
  }
}

/** Headers that mark a native client and carry its bearer token. */
export function nativeAuthHeaders(): Record<string, string> {
  if (!isNative()) return {};
  // The header tells the backend to include the tokens in the response body;
  // browsers keep getting cookie-only responses.
  const headers: Record<string, string> = { "X-Kisy-Client": "native" };
  const tokens = loadTokens();
  if (tokens?.accessToken) headers.Authorization = `Bearer ${tokens.accessToken}`;
  return headers;
}

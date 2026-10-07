// Content-Security-Policy for the Capacitor shell (audit A-45).
//
// On the web the server sends the policy as a header
// (backend/internal/platform/security/headers.go). Inside the app the bundle
// is served by the WebView itself from https://localhost: no server, no
// header, no policy at all — so the build bakes one into index.html as a
// <meta> tag. It is the web policy with the API's origin added where the app
// reaches it: requests and the WebSocket (connect-src) and pictures and audio
// fetched under authorization into blob: URLs (img-src, media-src; see
// shared/lib/mediaSrc.ts). frame-ancestors has no effect in a <meta> tag and
// is left out. 'wasm-unsafe-eval' is for libsodium, which is WebAssembly with
// no fallback (see headers.go) — it allows compiling wasm, never eval.

/** The policy for a native build whose API lives at apiOrigin (https://host). */
export function nativeContentSecurityPolicy(apiOrigin: string): string {
  const api = new URL(apiOrigin);
  if (api.protocol !== "https:" || api.pathname !== "/" || api.search || api.hash || api.username) {
    throw new Error(`VITE_NATIVE_API_ORIGIN must be a bare https origin, got ${apiOrigin}`);
  }
  const origin = api.origin;
  const socket = `wss://${api.host}`;
  return [
    "default-src 'self'",
    "script-src 'self' 'wasm-unsafe-eval' https://challenges.cloudflare.com",
    "frame-src https://challenges.cloudflare.com",
    "style-src 'self' 'unsafe-inline'",
    `img-src 'self' data: blob: ${origin}`,
    `media-src 'self' blob: ${origin}`,
    "font-src 'self'",
    `connect-src 'self' ${origin} ${socket}`,
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ].join("; ");
}

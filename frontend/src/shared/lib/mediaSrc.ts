import { useEffect, useState } from "react";
import { apiOrigin, isNative, nativeAuthHeaders } from "@shared/lib/native";
import { refreshSession } from "@shared/api/client";

// Attachments, voice notes and avatars are served by the API behind
// authentication, and every component points an <img>/<audio> straight at the
// relative URL the DTO carries ("/api/v1/attachments/<id>").
//
// In a browser tab that is exactly right: same origin, and the session cookie
// rides along automatically. Inside the Capacitor shell both halves break —
// the bundle is served from https://localhost, so the relative URL resolves to
// the WebView's own origin where no API exists, and even pointed at the real
// origin an <img> tag sends no Authorization header and no cross-site cookie.
// The result on the phone: images collapse to their file name, voice notes
// never load, avatars stay blank.
//
// So on native the bytes are fetched with the same authenticated client the
// rest of the app uses and handed to the tag as a blob: URL. On the web
// nothing changes — the URL is returned untouched and no fetch happens.

/** True for the API-served paths that need the treatment above. */
function isApiAsset(url: string): boolean {
  return url.startsWith("/api/");
}

async function fetchAsset(url: string, allowRefresh = true): Promise<string> {
  const response = await fetch(`${apiOrigin()}${url}`, {
    credentials: "omit",
    headers: nativeAuthHeaders(),
  });
  if (response.status === 401 && allowRefresh && (await refreshSession())) {
    return fetchAsset(url, false);
  }
  if (!response.ok) throw new Error(`media: ${response.status}`);
  return URL.createObjectURL(await response.blob());
}

/**
 * Resolves an API asset URL to something a media tag can load.
 *
 * Returns the URL unchanged on the web (and for anything not served by the
 * API); on native it returns null until the authenticated fetch resolves, so
 * callers should treat null as "still loading" rather than "missing".
 */
export function useMediaSrc(url: string | null | undefined): string | null {
  const [src, setSrc] = useState<string | null>(
    !url || !isNative() || !isApiAsset(url) ? (url ?? null) : null,
  );

  useEffect(() => {
    if (!url || !isNative() || !isApiAsset(url)) {
      setSrc(url ?? null);
      return;
    }

    let objectUrl: string | null = null;
    let cancelled = false;

    void fetchAsset(url)
      .then((resolved) => {
        if (cancelled) {
          URL.revokeObjectURL(resolved);
          return;
        }
        objectUrl = resolved;
        setSrc(resolved);
      })
      .catch(() => {
        if (!cancelled) setSrc(null);
      });

    return () => {
      cancelled = true;
      // Blob URLs pin their bytes in memory until revoked, and a chat scrolls
      // through a lot of them.
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [url]);

  return src;
}

/**
 * The URL to use when no authenticated fetch is required (web, or anything
 * the API does not serve), otherwise null.
 *
 * Callers that must stay inside the user-gesture task use this first: a
 * browser only honours audio.play() while the activation is live, so
 * deferring playback to a promise callback can have it rejected outright.
 */
export function mediaSrcSync(url: string): string | null {
  return isNative() && isApiAsset(url) ? null : url;
}

/**
 * One-shot variant for imperative callers (the audio element the voice player
 * drives, downloads). Resolves to a URL that is usable right away; the caller
 * owns revoking it when native.
 */
export async function resolveMediaSrc(url: string): Promise<string> {
  if (!isNative() || !isApiAsset(url)) return url;
  return fetchAsset(url);
}

/** True when resolveMediaSrc handed out a blob: URL the caller must revoke. */
export function isObjectUrl(url: string): boolean {
  return url.startsWith("blob:");
}

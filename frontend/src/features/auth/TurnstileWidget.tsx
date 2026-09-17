import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";

// Cloudflare Turnstile on the sign-up form. The server refuses a sign-up
// without a token this widget produced (403 CAPTCHA_FAILED), so the page
// waits for one before submitting.
//
// Explicit rendering: the script is loaded once, on the first screen that
// needs it, and the widget is drawn into our own element. "interaction-only"
// keeps it invisible for most people; it shows the checkbox only when
// Cloudflare wants someone to click. CSP allows challenges.cloudflare.com for
// script-src and frame-src (backend security headers and deploy/nginx).

const SCRIPT_SRC = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";

interface TurnstileRenderOptions {
  sitekey: string;
  theme: "light" | "dark";
  appearance: "always" | "execute" | "interaction-only";
  language: string;
  callback: (token: string) => void;
  "expired-callback": () => void;
  "error-callback": () => void;
}

export interface TurnstileApi {
  render: (el: HTMLElement, options: TurnstileRenderOptions) => string;
  reset: (widgetId: string) => void;
  remove: (widgetId: string) => void;
}

declare global {
  interface Window {
    turnstile?: TurnstileApi;
  }
}

let loading: Promise<TurnstileApi> | null = null;

function loadTurnstile(): Promise<TurnstileApi> {
  if (window.turnstile) return Promise.resolve(window.turnstile);
  if (!loading) {
    loading = new Promise<TurnstileApi>((resolve, reject) => {
      const script = document.createElement("script");
      script.src = SCRIPT_SRC;
      script.async = true;
      script.onload = () => (window.turnstile ? resolve(window.turnstile) : reject(new Error("turnstile missing")));
      script.onerror = () => reject(new Error("turnstile script failed to load"));
      document.head.appendChild(script);
    }).catch((err: unknown) => {
      // Let the next attempt (a remount, a retry) try the network again.
      loading = null;
      throw err;
    });
  }
  return loading;
}

// The widget follows the active theme's light/dark scheme rather than the
// system one: every KISY theme declares color-scheme on <html>.
function currentScheme(): "light" | "dark" {
  return getComputedStyle(document.documentElement).colorScheme.includes("dark") ? "dark" : "light";
}

export interface TurnstileHandle {
  // A token is single use: after a refused sign-up the widget has to issue a
  // fresh one before the form can be sent again.
  reset: () => void;
}

interface Props {
  siteKey: string;
  // null when the token expired or the check failed — the form must not be
  // sent with a stale one.
  onToken: (token: string | null) => void;
  onError: () => void;
}

export const TurnstileWidget = forwardRef<TurnstileHandle, Props>(function TurnstileWidget(
  { siteKey, onToken, onError },
  ref,
) {
  const container = useRef<HTMLDivElement>(null);
  const widget = useRef<{ api: TurnstileApi; id: string } | null>(null);
  // Callbacks change identity on every render of the page; the widget is
  // rendered once, so it reads the latest ones through a ref.
  const handlers = useRef({ onToken, onError });
  handlers.current = { onToken, onError };

  useImperativeHandle(ref, () => ({
    reset: () => {
      handlers.current.onToken(null);
      if (widget.current) widget.current.api.reset(widget.current.id);
    },
  }));

  useEffect(() => {
    let dropped = false;
    loadTurnstile()
      .then((api) => {
        if (dropped || !container.current) return;
        const id = api.render(container.current, {
          sitekey: siteKey,
          theme: currentScheme(),
          appearance: "interaction-only",
          language: "ru",
          callback: (token) => handlers.current.onToken(token),
          "expired-callback": () => handlers.current.onToken(null),
          "error-callback": () => {
            handlers.current.onToken(null);
            handlers.current.onError();
          },
        });
        widget.current = { api, id };
      })
      .catch(() => {
        if (!dropped) handlers.current.onError();
      });
    return () => {
      dropped = true;
      if (widget.current) {
        widget.current.api.remove(widget.current.id);
        widget.current = null;
      }
    };
  }, [siteKey]);

  return <div ref={container} className="auth-captcha" data-testid="turnstile" />;
});

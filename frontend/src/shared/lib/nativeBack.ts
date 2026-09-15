import { App } from "@capacitor/app";
import { isNative } from "@shared/lib/native";
import { handleBackPress } from "@shared/lib/backStack";

// Wires the Android back button and edge-swipe gesture to the app's own idea
// of "back" (see backStack.ts). A no-op in the browser, where the platform
// already owns the gesture.

/**
 * The router's position in its own history. React Router's data router keeps
 * it in history.state; index 0 is the entry the app opened on, and there is
 * nothing behind it.
 */
function historyIndex(): number {
  const state = window.history.state as { idx?: number } | null;
  return typeof state?.idx === "number" ? state.idx : 0;
}

/**
 * Starts listening. `goBack` steps the router back one entry.
 *
 * Registered once, outside the router, because the gesture can arrive during a
 * cold start — the same reason the push-notification handler lives there.
 */
export function initAndroidBack(goBack: () => void): () => void {
  if (!isNative()) return () => {};

  let remove: (() => void) | null = null;
  let dropped = false;

  void App.addListener("backButton", () => {
    handleBackPress({
      historyIndex: historyIndex(),
      goBack,
      // Not App.exitApp(): closing throws away the session, the socket and the
      // open conversation, and getting back in means a cold start.
      minimize: () => void App.minimizeApp(),
    });
  })
    .then((handle) => {
      if (dropped) void handle.remove();
      else remove = () => void handle.remove();
    })
    .catch(() => {
      // An older shell without the plugin keeps the platform default.
    });

  return () => {
    dropped = true;
    remove?.();
  };
}

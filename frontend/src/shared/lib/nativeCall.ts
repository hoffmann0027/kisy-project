import { registerPlugin } from "@capacitor/core";
import type { PluginListenerHandle } from "@capacitor/core";
import { isNative } from "@shared/lib/native";
import { callLog } from "@shared/lib/callLog";

// The web half of the native ringing screen (android/app/src/main/java/com/
// kisy/messenger/calls).
//
// A killed app runs no JavaScript, so a call push cannot reach this layer at
// all: the native side rings, shows the call over the lock screen and records
// what the user tapped. This module collects that decision once the WebView is
// alive and hands it to useCall, which owns the actual call.

export type NativeCallAction = "accept" | "reject";

export interface NativeCallDecision {
  action: NativeCallAction;
  callId: string;
}

interface KisyCallPlugin {
  getPendingAction(): Promise<{ action: string | null; callId: string | null }>;
  stopRinging(): Promise<void>;
  canUseFullScreenIntent(): Promise<{ granted: boolean }>;
  openFullScreenIntentSettings(): Promise<void>;
  addListener(
    event: "callAction",
    fn: (data: { action: string | null; callId: string | null }) => void,
  ): Promise<PluginListenerHandle>;
}

const KisyCall = registerPlugin<KisyCallPlugin>("KisyCall");

function decision(raw: { action: string | null; callId: string | null }): NativeCallDecision | null {
  if (!raw.callId) return null;
  if (raw.action !== "accept" && raw.action !== "reject") return null;
  return { action: raw.action, callId: raw.callId };
}

/**
 * Takes the decision made on the native ringing screen, if there is one.
 *
 * Reading it clears it on the native side: replaying an old "reject" would
 * hang up whatever call happens to be ringing now.
 */
export async function takeNativeCallDecision(): Promise<NativeCallDecision | null> {
  if (!isNative()) return null;
  try {
    const found = decision(await KisyCall.getPendingAction());
    if (found) callLog("native decision picked up", found);
    return found;
  } catch (err) {
    // An older build without the plugin: calls still work through the socket.
    callLog("plugin unavailable", err);
    return null;
  }
}

/** Subscribes to decisions taken while the app is running. Returns an unsubscribe. */
export function onNativeCallDecision(fn: (d: NativeCallDecision) => void): () => void {
  if (!isNative()) return () => {};
  let handle: PluginListenerHandle | null = null;
  let dropped = false;
  void KisyCall.addListener("callAction", (raw) => {
    const found = decision(raw);
    if (!found) return;
    callLog("native decision while running", found);
    fn(found);
  })
    .then((h) => {
      if (dropped) void h.remove();
      else handle = h;
    })
    .catch(() => {});
  return () => {
    dropped = true;
    void handle?.remove();
  };
}

/** Clears the call notification and stops the ringtone. */
export async function stopNativeRinging(): Promise<void> {
  if (!isNative()) return;
  await KisyCall.stopRinging().catch(() => {});
}

/**
 * Logs whether Android will let this build take over a locked screen for a
 * call. Called once at startup: when the answer is no, a call on a locked
 * phone arrives as a banner plus a ringtone instead of a call screen, and that
 * is the first thing to check when someone says the phone never rang.
 */
export async function reportFullScreenIntent(): Promise<boolean> {
  if (!isNative()) return false;
  try {
    const { granted } = await KisyCall.canUseFullScreenIntent();
    callLog(granted ? "full-screen intent allowed" : "full-screen intent DENIED — calls show as a banner");
    return granted;
  } catch {
    return false;
  }
}

/** Opens the system page where the full-screen-intent permission is granted. */
export async function openFullScreenIntentSettings(): Promise<void> {
  if (!isNative()) return;
  await KisyCall.openFullScreenIntentSettings().catch(() => {});
}

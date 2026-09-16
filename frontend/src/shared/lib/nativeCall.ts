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

/** Where a call's sound goes. "wired" and "bluetooth" are headsets. */
export type AudioRoute = "earpiece" | "speaker" | "wired" | "bluetooth";

export interface AudioRouteState {
  route: AudioRoute;
  /** What the loudspeaker button asked for; a headset can override it. */
  speaker: boolean;
}

/** Capacitor's permission vocabulary; "denied" means only Settings can change it. */
export type NativePermissionState = "granted" | "prompt" | "prompt-with-rationale" | "denied";
export type NativePermission = "notifications" | "microphone" | "fullScreenIntent";

interface KisyCallPlugin {
  getPendingAction(): Promise<{ action: string | null; callId: string | null }>;
  startCallAudio(opts: { video: boolean }): Promise<{ route: AudioRoute | null }>;
  setSpeaker(opts: { on: boolean }): Promise<{ route: AudioRoute | null }>;
  stopCallAudio(): Promise<void>;
  checkAppPermissions(): Promise<Record<NativePermission, NativePermissionState>>;
  requestAppPermission(opts: { name: NativePermission }): Promise<{ state: NativePermissionState }>;
  openPermissionSettings(opts: { name: NativePermission }): Promise<void>;
  addListener(event: "audioRoute", fn: (data: AudioRouteState) => void): Promise<PluginListenerHandle>;
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

// --- call audio ---------------------------------------------------------------

function isRoute(v: unknown): v is AudioRoute {
  return v === "earpiece" || v === "speaker" || v === "wired" || v === "bluetooth";
}

/**
 * Routes the call's sound for a voice call (earpiece, screen blanks at the ear)
 * or a video call (loudspeaker). Safe to call again: it re-asserts the route,
 * which the WebView overrides when remote audio starts playing.
 */
export async function startCallAudio(video: boolean): Promise<void> {
  if (!isNative()) return;
  await KisyCall.startCallAudio({ video }).catch((err) => callLog("call audio unavailable", err));
}

export async function setCallSpeaker(on: boolean): Promise<void> {
  if (!isNative()) return;
  await KisyCall.setSpeaker({ on }).catch((err) => callLog("speaker switch failed", err));
}

/**
 * Normal audio, proximity lock released. Harmless without a call, which is
 * why it also runs when the call layer mounts: a WebView reloaded mid-call
 * must not leave the screen blanking at the ear.
 */
export async function stopCallAudio(): Promise<void> {
  if (!isNative()) return;
  await KisyCall.stopCallAudio().catch(() => {});
}

/** Subscribes to route changes (headset in or out, loudspeaker). Returns an unsubscribe. */
export function onAudioRoute(fn: (state: AudioRouteState) => void): () => void {
  if (!isNative()) return () => {};
  let handle: PluginListenerHandle | null = null;
  let dropped = false;
  void KisyCall.addListener("audioRoute", (raw) => {
    if (!isRoute(raw.route)) return;
    fn({ route: raw.route, speaker: raw.speaker === true });
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

// --- permissions ---------------------------------------------------------------

export async function checkNativePermissions(): Promise<Record<NativePermission, NativePermissionState>> {
  return KisyCall.checkAppPermissions();
}

export async function requestNativePermission(name: NativePermission): Promise<NativePermissionState> {
  return (await KisyCall.requestAppPermission({ name })).state;
}

export async function openNativePermissionSettings(name: NativePermission): Promise<void> {
  await KisyCall.openPermissionSettings({ name });
}

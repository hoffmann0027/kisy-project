/**
 * One tagged line per step of a call.
 *
 * A call that fails to ring fails on a phone, with no devtools attached, so
 * the only trail is logcat — Capacitor forwards console output there. Every
 * line is prefixed so the whole path is one filter:
 *
 *     adb logcat | grep -E "call-push|KisyCall"
 *
 * `KisyCall` is the native half (push received, ringing, what was tapped) and
 * `call-push` is this half (what the app did with it). Together they show
 * exactly where a silent phone stopped.
 */
export function callLog(step: string, detail?: unknown): void {
  if (detail === undefined) {
    console.info(`[call-push] ${step}`);
    return;
  }
  console.info(`[call-push] ${step}`, detail);
}

// The permission onboarding as data: which screens a person sees, in what
// order, and what each screen offers. No platform calls here, so the rules can
// be tested on their own — the platform half lives in platform.ts.

export type AppPermission = "notifications" | "microphone" | "fullScreenIntent";

/**
 * Capacitor's vocabulary, plus "unsupported" for a browser without the API.
 * "denied" always means the system will not ask again: only Settings can
 * change it.
 */
export type PermissionState = "granted" | "prompt" | "prompt-with-rationale" | "denied" | "unsupported";

export type Platform = "native" | "web";

/**
 * The order is deliberate. Notifications first: without them nobody learns a
 * call is coming at all. Then the microphone the call needs once it is
 * answered. Then the full-screen call screen, which is a settings page rather
 * than a dialog and the most to ask of someone.
 *
 * The camera is not here on purpose: it is asked for when it is first needed.
 */
export const ONBOARDING_ORDER: readonly AppPermission[] = ["notifications", "microphone", "fullScreenIntent"];

/** What a browser can ask for at all. A web page cannot route a call to the lock screen, and the microphone is asked for by the browser when a call starts. */
const WEB_PERMISSIONS: readonly AppPermission[] = ["notifications"];

export function permissionsFor(platform: Platform): AppPermission[] {
  return ONBOARDING_ORDER.filter((p) => platform === "native" || WEB_PERMISSIONS.includes(p));
}

/**
 * The screens to show. Already granted means nothing to explain; unsupported
 * means nothing to ask. What is left keeps the fixed order.
 */
export function onboardingSteps(platform: Platform, states: Partial<Record<AppPermission, PermissionState>>): AppPermission[] {
  return permissionsFor(platform).filter((p) => {
    const state = states[p];
    return state !== undefined && state !== "granted" && state !== "unsupported";
  });
}

/**
 * What the main button on a screen does.
 *
 * - "request": show the system dialog.
 * - "settings": the system will not show it again (or never had one — the
 *   full-screen call screen is a settings toggle), so the button opens
 *   Settings instead. Asking again here would only bounce straight back.
 * - "instructions": the same situation in a browser, which cannot open its
 *   own settings page — the screen explains where to go.
 * - "done": nothing left to do on this screen.
 */
export type StepAction = "request" | "settings" | "instructions" | "done";

export function actionFor(permission: AppPermission, state: PermissionState, platform: Platform): StepAction {
  if (state === "granted" || state === "unsupported") return "done";
  // Android has no dialog for this one: it is always a trip to Settings.
  if (permission === "fullScreenIntent") return platform === "native" ? "settings" : "done";
  if (state === "denied") return platform === "native" ? "settings" : "instructions";
  return "request";
}

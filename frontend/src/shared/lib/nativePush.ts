import { PushNotifications } from "@capacitor/push-notifications";
import type { PluginListenerHandle } from "@capacitor/core";
import { pushApi } from "@shared/api/endpoints";
import { isNative } from "@shared/lib/native";

// Push notifications for the packaged mobile app.
//
// The browser build subscribes through the service worker (see push.ts); a
// WebView has no Push API, so the app receives notifications through Firebase
// Cloud Messaging instead. The plugin hands us a registration token, the
// backend stores it against the user and sends to it (backend/internal/push).

// Must match push.AndroidChannelID on the backend: FCM addresses this channel
// by name, and a mismatch would silently downgrade every notification to the
// low-importance fallback channel.
const CHANNEL_ID = "kisy_messages";

// The last token we registered, kept so sign-out can tell the server to forget
// this device even after the plugin has been unregistered.
const TOKEN_KEY = "kisy.native.pushToken";

/** True when running inside the mobile shell, where FCM is the transport. */
export function nativePushSupported(): boolean {
  return isNative();
}

function storedToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

function rememberToken(token: string | null): void {
  try {
    if (token) localStorage.setItem(TOKEN_KEY, token);
    else localStorage.removeItem(TOKEN_KEY);
  } catch {
    // A full or disabled store only costs us the sign-out cleanup.
  }
}

/**
 * Runs register() and resolves with the token Firebase answers with.
 *
 * The plugin reports the result through events rather than the register()
 * promise, so this bridges the two and gives up rather than hanging forever if
 * Google Play services never answers.
 */
async function requestToken(timeoutMs = 15_000): Promise<string> {
  const handles: PluginListenerHandle[] = [];
  try {
    return await new Promise<string>((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error("push: registration timed out")), timeoutMs);
      const settle = (fn: () => void) => {
        clearTimeout(timer);
        fn();
      };
      void PushNotifications.addListener("registration", (token) => {
        settle(() => resolve(token.value));
      }).then((h) => handles.push(h));
      void PushNotifications.addListener("registrationError", (err) => {
        settle(() => reject(new Error(`push: registration failed: ${JSON.stringify(err)}`)));
      }).then((h) => handles.push(h));
      void PushNotifications.register().catch((err: unknown) => {
        settle(() => reject(err instanceof Error ? err : new Error(String(err))));
      });
    });
  } finally {
    // Leaving these attached would double-handle the next registration.
    await Promise.all(handles.map((h) => h.remove().catch(() => {})));
  }
}

/**
 * Creates the high-importance channel the backend addresses. Idempotent, and
 * a no-op on anything but Android.
 */
async function ensureChannel(): Promise<void> {
  try {
    await PushNotifications.createChannel({
      id: CHANNEL_ID,
      name: "Сообщения",
      description: "Новые сообщения и упоминания",
      importance: 5, // heads-up, with sound
      visibility: 1, // shown on the lock screen, without the message body
    });
  } catch {
    // iOS has no channels, and a refusal here only affects presentation.
  }
}

/**
 * Asks for the notification permission, registers with Firebase and hands the
 * token to the backend. Returns false when the user declines or the server has
 * no Firebase credentials configured.
 */
export async function enableNativePush(): Promise<boolean> {
  if (!nativePushSupported()) return false;

  const { mobileEnabled } = await pushApi.vapidKey();
  // Nothing on the server could deliver a push: do not spend the one
  // permission prompt Android gives us.
  if (!mobileEnabled) return false;

  let status = await PushNotifications.checkPermissions();
  if (status.receive === "prompt" || status.receive === "prompt-with-rationale") {
    status = await PushNotifications.requestPermissions();
  }
  if (status.receive !== "granted") return false;

  await ensureChannel();
  const token = await requestToken();
  await pushApi.registerDevice(token, "android");
  rememberToken(token);
  return true;
}

/** Stops delivery to this device and forgets it server-side. */
export async function disableNativePush(): Promise<void> {
  if (!nativePushSupported()) return;
  const token = storedToken();
  if (token) await pushApi.unregisterDevice(token).catch(() => {});
  rememberToken(null);
  await PushNotifications.unregister().catch(() => {});
}

/** Reports whether this device currently receives pushes. */
export async function nativePushEnabled(): Promise<boolean> {
  if (!nativePushSupported()) return false;
  if (!storedToken()) return false;
  const status = await PushNotifications.checkPermissions();
  return status.receive === "granted";
}

/**
 * Re-registers a device that already opted in, on every sign-in. Two reasons
 * this is not skippable: FCM rotates tokens (restore to a new phone, cleared
 * app data, or its own schedule) and a stale token stops delivering silently,
 * and signing out unbinds the device server-side, so the binding has to be
 * re-established for whoever signs in next.
 */
export async function refreshNativePushToken(): Promise<void> {
  if (!(await nativePushEnabled())) return;
  try {
    await ensureChannel();
    const token = await requestToken();
    await pushApi.registerDevice(token, "android");
    rememberToken(token);
  } catch {
    // Best effort: a failed refresh leaves the previous token in place.
  }
}

/**
 * Unbinds this device from the account being signed out, without giving up the
 * notification permission — the next sign-in re-registers silently.
 *
 * Called before the session is torn down, since the request needs it.
 */
export async function forgetNativePushDevice(): Promise<void> {
  if (!nativePushSupported()) return;
  const token = storedToken();
  if (!token) return;
  await pushApi.unregisterDevice(token).catch(() => {
    // Worst case the server keeps a token it will prune on the first failed
    // send; blocking sign-out over it would be worse.
  });
}

// Guards against a second listener — a remount (StrictMode in development)
// would otherwise navigate twice per tap.
let navigationWired = false;

/**
 * Wires the notification tap. Called once at startup; `navigate` receives the
 * in-app path the backend attached to the push.
 */
export function initNativePushNavigation(navigate: (path: string) => void): void {
  if (!nativePushSupported() || navigationWired) return;
  navigationWired = true;
  void PushNotifications.addListener("pushNotificationActionPerformed", (action) => {
    const url = action.notification.data?.url;
    // Only in-app paths. "//host" is excluded too: it looks relative but
    // resolves to another origin.
    if (typeof url === "string" && url.startsWith("/") && !url.startsWith("//")) navigate(url);
  });
}

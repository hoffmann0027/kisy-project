import { pushApi } from "@shared/api/endpoints";
import {
  disableNativePush,
  enableNativePush,
  nativePushEnabled,
  nativePushSupported,
} from "@shared/lib/nativePush";

// Push helpers, one façade over two transports. In a browser the service
// worker (registered in production) shows the notifications and we manage a
// Web Push subscription; inside the mobile shell there is no Push API, so the
// same calls route to Firebase (see nativePush.ts). Callers — the profile
// toggle — do not care which one is in play.

export function pushSupported(): boolean {
  if (nativePushSupported()) return true;
  return "serviceWorker" in navigator && "PushManager" in window && "Notification" in window;
}

function urlBase64ToUint8Array(base64: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const b64 = (base64 + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(b64);
  const arr = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i++) arr[i] = raw.charCodeAt(i);
  return arr;
}

// enablePush requests permission and subscribes this browser. Returns true on
// success. Throws/returns false if unsupported, denied, or push is disabled
// server-side (no VAPID key).
export async function enablePush(): Promise<boolean> {
  if (nativePushSupported()) return enableNativePush();
  if (!pushSupported()) return false;
  const { publicKey, enabled } = await pushApi.vapidKey();
  if (!enabled || !publicKey) return false;

  const permission = await Notification.requestPermission();
  if (permission !== "granted") return false;

  const reg = await navigator.serviceWorker.ready;
  const existing = await reg.pushManager.getSubscription();
  const sub =
    existing ??
    (await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(publicKey),
    }));

  const json = sub.toJSON() as { endpoint?: string; keys?: { p256dh?: string; auth?: string } };
  if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) return false;
  await pushApi.subscribe({ endpoint: json.endpoint, keys: { p256dh: json.keys.p256dh, auth: json.keys.auth } });
  return true;
}

export async function disablePush(): Promise<void> {
  if (nativePushSupported()) return disableNativePush();
  if (!pushSupported()) return;
  const reg = await navigator.serviceWorker.ready;
  const sub = await reg.pushManager.getSubscription();
  if (!sub) return;
  await pushApi.unsubscribe(sub.endpoint).catch(() => {});
  await sub.unsubscribe().catch(() => {});
}

// pushEnabled reports whether this browser currently has an active subscription.
export async function pushEnabled(): Promise<boolean> {
  if (nativePushSupported()) return nativePushEnabled();
  if (!pushSupported() || Notification.permission !== "granted") return false;
  const reg = await navigator.serviceWorker.ready;
  return (await reg.pushManager.getSubscription()) != null;
}

import { Preferences } from "@capacitor/preferences";
import { isNative } from "@shared/lib/native";
import {
  checkNativePermissions,
  openNativePermissionSettings,
  requestNativePermission,
} from "@shared/lib/nativeCall";
import type { AppPermission, PermissionState, Platform } from "./sequence";

// Reading, requesting and fixing permissions on whichever platform this is.
// Native goes through the KisyCall plugin; a browser has only the Notification
// API, and the onboarding asks it for nothing else.

export function currentPlatform(): Platform {
  return isNative() ? "native" : "web";
}

function webNotificationState(): PermissionState {
  if (typeof window === "undefined" || !("Notification" in window)) return "unsupported";
  switch (Notification.permission) {
    case "granted":
      return "granted";
    case "denied":
      return "denied";
    default:
      return "prompt";
  }
}

export async function readPermissions(): Promise<Partial<Record<AppPermission, PermissionState>>> {
  if (currentPlatform() === "web") return { notifications: webNotificationState() };
  return checkNativePermissions();
}

export async function requestPermission(permission: AppPermission): Promise<PermissionState> {
  if (currentPlatform() === "web") {
    if (permission !== "notifications" || webNotificationState() === "unsupported") return "unsupported";
    await Notification.requestPermission();
    return webNotificationState();
  }
  return requestNativePermission(permission);
}

export async function openPermissionSettings(permission: AppPermission): Promise<void> {
  if (currentPlatform() === "native") await openNativePermissionSettings(permission);
}

// Once per device, not per account: the permissions belong to the phone.
// Preferences rather than localStorage because on Android it is the app's own
// SharedPreferences — it survives the WebView's storage being cleared, which
// would otherwise replay the whole onboarding.
const DONE_KEY = "kisy.permissionsOnboarding.done";

export async function onboardingDone(): Promise<boolean> {
  try {
    return (await Preferences.get({ key: DONE_KEY })).value === "1";
  } catch {
    // Unreadable storage: better to skip than to show it on every start.
    return true;
  }
}

export async function markOnboardingDone(): Promise<void> {
  await Preferences.set({ key: DONE_KEY, value: "1" }).catch(() => {});
}

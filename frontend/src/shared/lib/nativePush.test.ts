import { beforeEach, describe, expect, it, vi } from "vitest";

// The plugin is native code; here it is a scriptable stand-in that lets each
// test decide what Firebase and the OS answer.
const plugin = vi.hoisted(() => {
  type Listener = (payload: unknown) => void;
  const listeners = new Map<string, Set<Listener>>();
  return {
    listeners,
    permission: { receive: "granted" as string },
    registerBehaviour: "registration" as "registration" | "error" | "silent",
    token: "fcm-token-1",
    removed: 0,
    emit(event: string, payload: unknown) {
      for (const fn of listeners.get(event) ?? []) fn(payload);
    },
    api: {
      checkPermissions: vi.fn(async () => ({ receive: plugin.permission.receive })),
      requestPermissions: vi.fn(async () => ({ receive: plugin.permission.receive })),
      createChannel: vi.fn(async () => {}),
      unregister: vi.fn(async () => {}),
      register: vi.fn(async () => {
        // The real plugin answers through events, not this promise.
        if (plugin.registerBehaviour === "registration") plugin.emit("registration", { value: plugin.token });
        if (plugin.registerBehaviour === "error") plugin.emit("registrationError", { error: "no play services" });
      }),
      addListener: vi.fn(async (event: string, fn: Listener) => {
        if (!listeners.has(event)) listeners.set(event, new Set());
        listeners.get(event)!.add(fn);
        return {
          remove: async () => {
            plugin.removed++;
            listeners.get(event)!.delete(fn);
          },
        };
      }),
    },
  };
});

const api = vi.hoisted(() => ({
  vapidKey: vi.fn(),
  registerDevice: vi.fn(),
  unregisterDevice: vi.fn(),
}));

const nativeFlag = vi.hoisted(() => ({ value: true }));

vi.mock("@capacitor/push-notifications", () => ({ PushNotifications: plugin.api }));
vi.mock("@shared/api/endpoints", () => ({ pushApi: api }));
vi.mock("@shared/lib/native", () => ({ isNative: () => nativeFlag.value }));

import {
  disableNativePush,
  enableNativePush,
  forgetNativePushDevice,
  initNativePushNavigation,
  nativePushEnabled,
  refreshNativePushToken,
} from "./nativePush";

describe("native push", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    plugin.listeners.clear();
    plugin.permission.receive = "granted";
    plugin.registerBehaviour = "registration";
    plugin.token = "fcm-token-1";
    plugin.removed = 0;
    nativeFlag.value = true;
    api.vapidKey.mockResolvedValue({ publicKey: "", enabled: false, mobileEnabled: true });
    api.registerDevice.mockResolvedValue({ registered: true });
    api.unregisterDevice.mockResolvedValue({ unregistered: true });
  });

  it("registers the Firebase token with the backend", async () => {
    await expect(enableNativePush()).resolves.toBe(true);
    expect(api.registerDevice).toHaveBeenCalledWith("fcm-token-1", "android");
    // The high-importance channel must exist, or pushes arrive silently.
    expect(plugin.api.createChannel).toHaveBeenCalledWith(expect.objectContaining({ id: "kisy_messages" }));
    await expect(nativePushEnabled()).resolves.toBe(true);
    // Both registration listeners are detached again.
    expect(plugin.removed).toBe(2);
  });

  it("does not prompt when the server cannot send pushes", async () => {
    api.vapidKey.mockResolvedValue({ publicKey: "", enabled: false, mobileEnabled: false });
    await expect(enableNativePush()).resolves.toBe(false);
    expect(plugin.api.requestPermissions).not.toHaveBeenCalled();
    expect(api.registerDevice).not.toHaveBeenCalled();
  });

  it("gives up when the user denies the notification permission", async () => {
    plugin.permission.receive = "denied";
    await expect(enableNativePush()).resolves.toBe(false);
    expect(plugin.api.register).not.toHaveBeenCalled();
    expect(api.registerDevice).not.toHaveBeenCalled();
  });

  it("surfaces a registration failure instead of hanging", async () => {
    plugin.registerBehaviour = "error";
    await expect(enableNativePush()).rejects.toThrow(/registration failed/);
    expect(api.registerDevice).not.toHaveBeenCalled();
    expect(plugin.removed).toBe(2);
  });

  it("does nothing at all in a browser", async () => {
    nativeFlag.value = false;
    await expect(enableNativePush()).resolves.toBe(false);
    await expect(nativePushEnabled()).resolves.toBe(false);
    expect(api.vapidKey).not.toHaveBeenCalled();
  });

  it("re-registers a rotated token on sign-in", async () => {
    await enableNativePush();
    api.registerDevice.mockClear();

    plugin.token = "fcm-token-2";
    await refreshNativePushToken();

    expect(api.registerDevice).toHaveBeenCalledWith("fcm-token-2", "android");
    expect(localStorage.getItem("kisy.native.pushToken")).toBe("fcm-token-2");
  });

  it("re-registers on sign-in even when the token has not changed", async () => {
    // Sign-out unbinds the device server-side, so an unchanged token still has
    // to be re-sent or the phone stays silent.
    await enableNativePush();
    api.registerDevice.mockClear();

    await refreshNativePushToken();

    expect(api.registerDevice).toHaveBeenCalledWith("fcm-token-1", "android");
  });

  it("skips the refresh for a device that never opted in", async () => {
    await refreshNativePushToken();
    expect(plugin.api.register).not.toHaveBeenCalled();
    expect(api.registerDevice).not.toHaveBeenCalled();
  });

  it("unbinds the device on sign-out but keeps the permission", async () => {
    await enableNativePush();
    await forgetNativePushDevice();

    expect(api.unregisterDevice).toHaveBeenCalledWith("fcm-token-1");
    // The token survives, so signing back in re-registers without a prompt.
    expect(localStorage.getItem("kisy.native.pushToken")).toBe("fcm-token-1");
    expect(plugin.api.unregister).not.toHaveBeenCalled();
  });

  it("turning push off forgets the device for good", async () => {
    await enableNativePush();
    await disableNativePush();

    expect(api.unregisterDevice).toHaveBeenCalledWith("fcm-token-1");
    expect(localStorage.getItem("kisy.native.pushToken")).toBeNull();
    expect(plugin.api.unregister).toHaveBeenCalled();
    await expect(nativePushEnabled()).resolves.toBe(false);
  });

  it("opens the chat a tapped notification points at", async () => {
    const navigate = vi.fn();
    initNativePushNavigation(navigate);
    // addListener is async; let the registration settle.
    await Promise.resolve();

    plugin.emit("pushNotificationActionPerformed", { notification: { data: { url: "/chat/42" } } });
    expect(navigate).toHaveBeenCalledWith("/chat/42");

    // A payload that is not an in-app path must not drive navigation —
    // including "//host", which looks relative but is not.
    navigate.mockClear();
    plugin.emit("pushNotificationActionPerformed", { notification: { data: { url: "https://evil.example" } } });
    plugin.emit("pushNotificationActionPerformed", { notification: { data: { url: "//evil.example/x" } } });
    plugin.emit("pushNotificationActionPerformed", { notification: { data: {} } });
    expect(navigate).not.toHaveBeenCalled();
  });
});

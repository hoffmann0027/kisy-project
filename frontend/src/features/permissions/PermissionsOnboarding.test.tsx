import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AppPermission, PermissionState, Platform } from "./sequence";

// The walkthrough on top of the sequence rules: one screen at a time, the
// system dialog only when there is one to show, Settings when there is not,
// and never again once finished.

const env = vi.hoisted(() => ({
  platform: "native" as Platform,
  done: false,
  states: {} as Partial<Record<AppPermission, PermissionState>>,
  answers: {} as Partial<Record<AppPermission, PermissionState>>,
}));

const calls = vi.hoisted(() => ({
  request: vi.fn(),
  settings: vi.fn(),
  markDone: vi.fn(),
  enablePush: vi.fn(async () => true),
}));

vi.mock("./platform", () => ({
  currentPlatform: () => env.platform,
  readPermissions: vi.fn(async () => ({ ...env.states })),
  requestPermission: vi.fn(async (p: AppPermission) => {
    calls.request(p);
    const answer = env.answers[p] ?? "denied";
    env.states[p] = answer;
    return answer;
  }),
  openPermissionSettings: vi.fn(async (p: AppPermission) => calls.settings(p)),
  onboardingDone: vi.fn(async () => env.done),
  markOnboardingDone: vi.fn(async () => {
    calls.markDone();
    env.done = true;
  }),
}));

vi.mock("@shared/lib/push", () => ({ enablePush: calls.enablePush }));

const { PermissionsOnboarding } = await import("./PermissionsOnboarding");

beforeEach(() => {
  vi.clearAllMocks();
  env.platform = "native";
  env.done = false;
  env.states = {};
  env.answers = {};
});

describe("the permission onboarding", () => {
  it("walks a phone through notifications, microphone and the call screen in that order", async () => {
    env.states = { notifications: "prompt", microphone: "prompt", fullScreenIntent: "denied" };
    env.answers = { notifications: "granted", microphone: "granted" };
    render(<PermissionsOnboarding />);

    expect(await screen.findByText("Уведомления")).toBeTruthy();
    fireEvent.click(screen.getByText("Разрешить"));
    // A granted notification permission is followed by the push registration.
    await waitFor(() => expect(calls.enablePush).toHaveBeenCalled());

    expect(await screen.findByText("Микрофон")).toBeTruthy();
    fireEvent.click(screen.getByText("Разрешить"));

    expect(await screen.findByText("Экран входящего звонка")).toBeTruthy();
    // No dialog exists for it on Android: the only way is Settings.
    expect(screen.queryByText("Разрешить")).toBeNull();
    fireEvent.click(screen.getByText("Открыть настройки"));
    expect(calls.settings).toHaveBeenCalledWith("fullScreenIntent");

    expect(calls.request.mock.calls.map(([p]) => p)).toEqual(["notifications", "microphone"]);
  });

  it("offers Settings instead of asking again once refused for good", async () => {
    env.states = { notifications: "granted", microphone: "prompt", fullScreenIntent: "granted" };
    env.answers = { microphone: "denied" };
    render(<PermissionsOnboarding />);

    fireEvent.click(await screen.findByText("Разрешить"));
    expect(await screen.findByText("Открыть настройки")).toBeTruthy();
    expect(screen.queryByText("Разрешить")).toBeNull();

    fireEvent.click(screen.getByText("Открыть настройки"));
    expect(calls.settings).toHaveBeenCalledWith("microphone");
    expect(calls.request).toHaveBeenCalledTimes(1);
  });

  it("moves on by itself when the permission was granted in Settings", async () => {
    env.states = { notifications: "granted", microphone: "denied", fullScreenIntent: "denied" };
    render(<PermissionsOnboarding />);
    expect(await screen.findByText("Микрофон")).toBeTruthy();

    // Back from Settings with the switch turned on.
    env.states.microphone = "granted";
    await act(async () => {
      window.dispatchEvent(new Event("focus"));
    });
    expect(await screen.findByText("Экран входящего звонка")).toBeTruthy();
  });

  it("asks a browser only about notifications, through the Notification API", async () => {
    env.platform = "web";
    env.states = { notifications: "prompt" };
    env.answers = { notifications: "granted" };
    render(<PermissionsOnboarding />);

    expect(await screen.findByText("Уведомления")).toBeTruthy();
    expect(screen.getByText("Готово")).toBeTruthy(); // the only screen
    fireEvent.click(screen.getByText("Разрешить"));
    await waitFor(() => expect(calls.markDone).toHaveBeenCalled());
    expect(screen.queryByText("Микрофон")).toBeNull();
  });

  it("explains where to go in a browser that refused, with nothing to press but Готово", async () => {
    env.platform = "web";
    env.states = { notifications: "denied" };
    render(<PermissionsOnboarding />);

    expect(await screen.findByText(/настройках браузера/)).toBeTruthy();
    expect(screen.queryByText("Разрешить")).toBeNull();
    expect(screen.queryByText("Открыть настройки")).toBeNull();
  });

  it("is marked done when skipped, and not shown again", async () => {
    env.states = { notifications: "prompt", microphone: "granted", fullScreenIntent: "granted" };
    const first = render(<PermissionsOnboarding />);
    fireEvent.click(await screen.findByText("Готово"));
    expect(calls.markDone).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(screen.queryByText("Уведомления")).toBeNull());
    first.unmount();

    render(<PermissionsOnboarding />);
    await new Promise((r) => setTimeout(r, 20));
    expect(screen.queryByText("Уведомления")).toBeNull();
    expect(calls.request).not.toHaveBeenCalled();
  });

  it("shows nothing and marks itself done on a phone that already has everything", async () => {
    env.states = { notifications: "granted", microphone: "granted", fullScreenIntent: "granted" };
    const { container } = render(<PermissionsOnboarding />);
    await waitFor(() => expect(calls.markDone).toHaveBeenCalled());
    expect(container.textContent).toBe("");
  });
});

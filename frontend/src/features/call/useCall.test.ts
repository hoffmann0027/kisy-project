import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The path from "the phone rang while the app was dead" to "the call is
// connected" runs through native code that no test can drive: a push, a
// notification, a lock screen, a tap. What CAN be pinned down is the half that
// runs here — what the app does once it is handed that tap — and it is the
// half where a mistake is invisible until someone is standing there with a
// ringing phone.

const native = vi.hoisted(() => ({
  decision: null as { action: "accept" | "reject"; callId: string } | null,
  listeners: new Set<(d: { action: "accept" | "reject"; callId: string }) => void>(),
}));

vi.mock("@shared/lib/nativeCall", () => ({
  takeNativeCallDecision: vi.fn(async () => native.decision),
  onNativeCallDecision: vi.fn((fn: (d: { action: "accept" | "reject"; callId: string }) => void) => {
    native.listeners.add(fn);
    return () => native.listeners.delete(fn);
  }),
  stopNativeRinging: vi.fn(async () => {}),
  reportFullScreenIntent: vi.fn(async () => true),
}));

vi.mock("@shared/lib/nativePush", () => ({ CALL_PUSH_EVENT: "kisy:call-push" }));

const ws = vi.hoisted(() => ({ send: vi.fn(), subscribe: vi.fn(() => () => {}) }));
vi.mock("@shared/ws/client", () => ({ wsClient: ws }));

const api = vi.hoisted(() => ({
  pending: vi.fn(async () => ({ call: null as unknown })),
  reject: vi.fn(async () => ({ rejected: true })),
  iceConfig: vi.fn(async () => ({ iceServers: [] })),
}));
vi.mock("@shared/api/endpoints", () => ({ callsApi: api }));

vi.mock("./ringtone", () => ({ ringtone: { incoming: vi.fn(), outgoing: vi.fn(), stop: vi.fn() } }));

const mic = vi.fn(async () => ({ getTracks: () => [], getAudioTracks: () => [] }));

class FakePeerConnection {
  onicecandidate: unknown = null;
  ontrack: unknown = null;
  onconnectionstatechange: unknown = null;
  connectionState = "new";
  addTrack = vi.fn();
  close = vi.fn();
  setRemoteDescription = vi.fn(async () => {});
  addIceCandidate = vi.fn(async () => {});
  createAnswer = vi.fn(async () => ({ type: "answer", sdp: "answer-sdp" }));
  setLocalDescription = vi.fn(async () => {});
}

const ringingCall = {
  callId: "call-1",
  callerId: "user-2",
  callerName: "Пётр",
  chatId: "chat-1",
  offer: "offer-sdp",
};

// Imported after the mocks are in place.
const { useCall } = await import("./useCall");

beforeEach(() => {
  vi.clearAllMocks();
  native.decision = null;
  native.listeners.clear();
  api.pending.mockResolvedValue({ call: null });
  Object.defineProperty(globalThis, "RTCPeerConnection", { value: FakePeerConnection, configurable: true });
  Object.defineProperty(globalThis.navigator, "mediaDevices", {
    value: { getUserMedia: mic },
    configurable: true,
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

const answers = () => ws.send.mock.calls.filter(([f]) => (f as { type: string }).type === "call.answer");

describe("a call answered on the native screen", () => {
  it("is picked up and connected when the app starts", async () => {
    native.decision = { action: "accept", callId: ringingCall.callId };
    api.pending.mockResolvedValue({ call: ringingCall });

    renderHook(() => useCall());

    // No second tap in the app: the user already answered on the lock screen.
    await waitFor(() => expect(answers()).toHaveLength(1));
    expect(answers()[0][0]).toMatchObject({ data: { callId: ringingCall.callId, sdp: "answer-sdp" } });
  });

  it("is answered once even when the tap arrives twice", async () => {
    // The plugin announces the tap to a running app AND leaves it on disk for
    // a cold start; both reach us, and answering twice would open a second
    // microphone and a second connection.
    native.decision = { action: "accept", callId: ringingCall.callId };
    api.pending.mockResolvedValue({ call: ringingCall });

    renderHook(() => useCall());
    await waitFor(() => expect(answers()).toHaveLength(1));

    for (const fn of native.listeners) fn({ action: "accept", callId: ringingCall.callId });
    document.dispatchEvent(new Event("visibilitychange"));

    await new Promise((r) => setTimeout(r, 20));
    expect(answers()).toHaveLength(1);
    expect(mic).toHaveBeenCalledTimes(1);
  });

  it("does nothing when the caller has already given up", async () => {
    native.decision = { action: "accept", callId: ringingCall.callId };
    api.pending.mockResolvedValue({ call: null });

    renderHook(() => useCall());

    await new Promise((r) => setTimeout(r, 20));
    expect(answers()).toHaveLength(0);
    expect(mic).not.toHaveBeenCalled();
  });
});

describe("a call declined on the native screen", () => {
  it("is declined over REST, because the socket may not be up yet", async () => {
    native.decision = { action: "reject", callId: ringingCall.callId };

    renderHook(() => useCall());

    await waitFor(() => expect(api.reject).toHaveBeenCalledWith(ringingCall.callId));
    expect(ws.send).not.toHaveBeenCalled();
  });

  it("is declined once, not once per way the tap reached us", async () => {
    native.decision = { action: "reject", callId: ringingCall.callId };

    renderHook(() => useCall());
    await waitFor(() => expect(api.reject).toHaveBeenCalledTimes(1));

    for (const fn of native.listeners) fn({ action: "reject", callId: ringingCall.callId });
    await new Promise((r) => setTimeout(r, 20));
    expect(api.reject).toHaveBeenCalledTimes(1);
  });
});

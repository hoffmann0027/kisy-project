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

const audio = vi.hoisted(() => ({
  start: vi.fn(async (_video: boolean) => {}),
  setSpeaker: vi.fn(async (_on: boolean) => {}),
  stop: vi.fn(async () => {}),
  listeners: new Set<(s: { route: string; speaker: boolean }) => void>(),
}));

vi.mock("@shared/lib/nativeCall", () => ({
  takeNativeCallDecision: vi.fn(async () => native.decision),
  onNativeCallDecision: vi.fn((fn: (d: { action: "accept" | "reject"; callId: string }) => void) => {
    native.listeners.add(fn);
    return () => native.listeners.delete(fn);
  }),
  stopNativeRinging: vi.fn(async () => {}),
  reportFullScreenIntent: vi.fn(async () => true),
  startCallAudio: audio.start,
  setCallSpeaker: audio.setSpeaker,
  stopCallAudio: audio.stop,
  onAudioRoute: vi.fn((fn: (s: { route: string; speaker: boolean }) => void) => {
    audio.listeners.add(fn);
    return () => audio.listeners.delete(fn);
  }),
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
  mic.mockResolvedValue({ getTracks: () => [], getAudioTracks: () => [] });
  Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
  native.decision = null;
  native.listeners.clear();
  audio.listeners.clear();
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

function setVisibility(state: "visible" | "hidden") {
  Object.defineProperty(document, "visibilityState", { value: state, configurable: true });
}

describe("answering from a locked screen", () => {
  it("waits for the app to be on screen instead of answering behind the lock", async () => {
    // Android does not give the microphone to an app nobody can see, and the
    // app is exactly there while the phone is still unlocking.
    setVisibility("hidden");
    native.decision = { action: "accept", callId: ringingCall.callId };
    api.pending.mockResolvedValue({ call: ringingCall });

    renderHook(() => useCall());
    await new Promise((r) => setTimeout(r, 20));

    expect(answers()).toHaveLength(0);
    expect(mic).not.toHaveBeenCalled();
    // The call is kept, not refused.
    expect(api.reject).not.toHaveBeenCalled();

    setVisibility("visible");
    document.dispatchEvent(new Event("visibilitychange"));
    await waitFor(() => expect(answers()).toHaveLength(1));
  });

  it("keeps the call ringing when the microphone is not available yet", async () => {
    // The tap said "answer". Turning a refused microphone into a hang-up would
    // answer the user's "Ответить" with a rejected call.
    setVisibility("visible");
    native.decision = { action: "accept", callId: ringingCall.callId };
    api.pending.mockResolvedValue({ call: ringingCall });
    mic.mockRejectedValueOnce(Object.assign(new Error("denied"), { name: "NotAllowedError" }));

    renderHook(() => useCall());
    await waitFor(() => expect(mic).toHaveBeenCalled());
    await new Promise((r) => setTimeout(r, 20));

    const rejects = ws.send.mock.calls.filter(([f]) => (f as { type: string }).type === "call.reject");
    expect(rejects).toHaveLength(0);
    expect(answers()).toHaveLength(0);
  });

  it("still hangs up when the user answers in the app and has no microphone", async () => {
    // Only the automatic answer is forgiving: a person who pressed the button
    // is looking at the screen and deserves to be told.
    setVisibility("visible");
    api.pending.mockResolvedValue({ call: ringingCall });
    mic.mockRejectedValueOnce(Object.assign(new Error("denied"), { name: "NotAllowedError" }));

    const { result } = renderHook(() => useCall());
    await waitFor(() => expect(result.current.view.phase).toBe("incoming"));
    await result.current.accept();

    const rejects = ws.send.mock.calls.filter(([f]) => (f as { type: string }).type === "call.reject");
    expect(rejects).toHaveLength(1);
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

describe("call audio", () => {
  it("routes a voice call to the earpiece while it has media, and releases it when it ends", async () => {
    // Mounting with no call clears anything a reloaded WebView left behind.
    api.pending.mockResolvedValue({ call: ringingCall });
    const { result } = renderHook(() => useCall());
    await waitFor(() => expect(audio.stop).toHaveBeenCalled());
    expect(audio.start).not.toHaveBeenCalled();

    // Ringing is not a call yet: the ringtone belongs on the loudspeaker.
    await waitFor(() => expect(result.current.view.phase).toBe("incoming"));
    expect(audio.start).not.toHaveBeenCalled();

    await result.current.accept();
    await waitFor(() => expect(audio.start).toHaveBeenCalledWith(false));
    expect(result.current.view.speaker).toBe(false);

    audio.stop.mockClear();
    result.current.hangup();
    await waitFor(() => expect(audio.stop).toHaveBeenCalledTimes(1));
  });

  it("keeps the loudspeaker choice in the call state and applies it through the plugin", async () => {
    api.pending.mockResolvedValue({ call: ringingCall });
    const { result } = renderHook(() => useCall());
    await waitFor(() => expect(result.current.view.phase).toBe("incoming"));
    await result.current.accept();
    await waitFor(() => expect(result.current.view.phase).toBe("connecting"));

    result.current.toggleSpeaker();
    await waitFor(() => expect(result.current.view.speaker).toBe(true));
    expect(audio.setSpeaker).toHaveBeenLastCalledWith(true);

    result.current.toggleSpeaker();
    await waitFor(() => expect(result.current.view.speaker).toBe(false));
    expect(audio.setSpeaker).toHaveBeenLastCalledWith(false);
  });

  it("shows the route the phone reports, so a headset replaces the loudspeaker button", async () => {
    api.pending.mockResolvedValue({ call: ringingCall });
    const { result } = renderHook(() => useCall());
    await waitFor(() => expect(result.current.view.phase).toBe("incoming"));
    await result.current.accept();
    await waitFor(() => expect(result.current.view.phase).toBe("connecting"));

    for (const fn of audio.listeners) fn({ route: "bluetooth", speaker: false });
    await waitFor(() => expect(result.current.view.audioRoute).toBe("bluetooth"));
  });

  it("ignores route reports when there is no call", async () => {
    const { result } = renderHook(() => useCall());
    await waitFor(() => expect(audio.stop).toHaveBeenCalled());
    for (const fn of audio.listeners) fn({ route: "wired", speaker: false });
    await new Promise((r) => setTimeout(r, 10));
    expect(result.current.view.audioRoute).toBeNull();
  });
});

describe("ringing after the call connects", () => {
  // Answering first and hearing the ring afterwards is the regression this
  // guards: whatever rang — the in-app tone or the native ringer woken by a
  // late call push — has to be silenced by the call connecting, and again on
  // every route change, not only at the moment of answering.
  async function connectedCall() {
    api.pending.mockResolvedValue({ call: ringingCall });
    const hook = renderHook(() => useCall());
    await waitFor(() => expect(hook.result.current.view.phase).toBe("incoming"));
    const created: FakePeerConnection[] = [];
    const Tracking = class extends FakePeerConnection {
      constructor() {
        super();
        created.push(this);
      }
    };
    Object.defineProperty(globalThis, "RTCPeerConnection", { value: Tracking, configurable: true });
    await hook.result.current.accept();
    await waitFor(() => expect(created).toHaveLength(1));
    return { ...hook, pc: created[0] };
  }

  it("stops both ringers when the connection reaches connected", async () => {
    const { result, pc } = await connectedCall();
    const { ringtone } = await import("./ringtone");
    const { stopNativeRinging } = await import("@shared/lib/nativeCall");
    vi.mocked(ringtone.stop).mockClear();
    vi.mocked(stopNativeRinging).mockClear();

    pc.connectionState = "connected";
    (pc.onconnectionstatechange as () => void)();

    await waitFor(() => expect(result.current.view.phase).toBe("active"));
    expect(ringtone.stop).toHaveBeenCalled();
    expect(stopNativeRinging).toHaveBeenCalled();
  });

  it("leaves the ringback alone while still dialing out", async () => {
    const Dialing = class extends FakePeerConnection {
      createOffer = vi.fn(async () => ({ type: "offer", sdp: "offer-sdp" }));
    };
    Object.defineProperty(globalThis, "RTCPeerConnection", { value: Dialing, configurable: true });
    const { result } = renderHook(() => useCall());
    await result.current.startCall({ id: "user-2", displayName: "Пётр", avatarUrl: null }, "chat-1");
    await waitFor(() => expect(result.current.view.phase).toBe("outgoing"));
    const { ringtone } = await import("./ringtone");
    vi.mocked(ringtone.stop).mockClear();
    for (const fn of audio.listeners) fn({ route: "speaker", speaker: true });
    await waitFor(() => expect(result.current.view.audioRoute).toBe("speaker"));
    expect(ringtone.stop).not.toHaveBeenCalled();
  });

  it("silences them again on every route change during the call", async () => {
    const { result, pc } = await connectedCall();
    pc.connectionState = "connected";
    (pc.onconnectionstatechange as () => void)();
    await waitFor(() => expect(result.current.view.phase).toBe("active"));

    const { ringtone } = await import("./ringtone");
    const { stopNativeRinging } = await import("@shared/lib/nativeCall");
    for (const round of [1, 2]) {
      vi.mocked(ringtone.stop).mockClear();
      vi.mocked(stopNativeRinging).mockClear();
      for (const fn of audio.listeners) fn({ route: round === 1 ? "speaker" : "earpiece", speaker: round === 1 });
      await waitFor(() => expect(stopNativeRinging).toHaveBeenCalled());
      expect(ringtone.stop).toHaveBeenCalled();
    }
  });
});

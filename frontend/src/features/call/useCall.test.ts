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

const ws = vi.hoisted(() => {
  const handlers = new Set<(e: unknown) => void>();
  return {
    handlers,
    send: vi.fn(),
    subscribe: vi.fn((fn: (e: unknown) => void) => {
      handlers.add(fn);
      return () => handlers.delete(fn);
    }),
  };
});
/** Deliver a server event to the hook, as the socket would. */
const serverSends = (e: unknown) => ws.handlers.forEach((fn) => fn(e));
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
  static last: FakePeerConnection | null = null;
  onicecandidate: unknown = null;
  ontrack: unknown = null;
  onconnectionstatechange: (() => void) | null = null;
  connectionState = "new";
  signalingState = "stable";
  constructor() {
    FakePeerConnection.last = this;
  }
  getConfiguration = vi.fn(() => ({ iceServers: [] }));
  sender = {
    track: { kind: "audio" },
    getParameters: vi.fn(() => ({ encodings: [{}] as RTCRtpEncodingParameters[] })),
    setParameters: vi.fn(async (_p: { encodings: RTCRtpEncodingParameters[] }) => {}),
  };
  getSenders = vi.fn(() => [this.sender]);
  /** What the network delivered so far, as getStats() reports it. */
  stats: Array<Record<string, unknown>> = [];
  getStats = vi.fn(async () => ({ forEach: (fn: (e: unknown) => void) => this.stats.forEach(fn) }));
  setConfiguration = vi.fn();
  /** The network moved the connection to a new state. */
  becomes(state: string) {
    this.connectionState = state;
    this.onconnectionstatechange?.();
  }
  addTrack = vi.fn();
  close = vi.fn();
  setRemoteDescription = vi.fn(async () => {});
  addIceCandidate = vi.fn(async () => {});
  createAnswer = vi.fn(async () => ({ type: "answer", sdp: "answer-sdp" }));
  createOffer = vi.fn(async (_opts?: RTCOfferOptions) => ({ type: "offer", sdp: "offer-sdp" }));
  setLocalDescription = vi.fn(async (d: { type: string }) => {
    this.signalingState = d.type === "offer" ? "have-local-offer" : "stable";
  });
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
  ws.handlers.clear();
  FakePeerConnection.last = null;
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

// Audit A-28: relay credentials are issued for one call. They used to be
// fetched once and cached for the life of the tab, so a tab left open past
// their expiry placed every later call without a relay.
describe("relay credentials", () => {
  it("are asked for the partner before ringing, and fresh for every call", async () => {
    const { result } = renderHook(() => useCall());
    const peer = { id: "user-2", displayName: "Пётр", avatarUrl: null };

    await result.current.startCall(peer, "chat-1");
    expect(api.iceConfig).toHaveBeenLastCalledWith({ chatId: "chat-1", peerId: "user-2" });
    result.current.hangup();

    await waitFor(() => expect(result.current.view.phase).not.toBe("outgoing"));
    await result.current.startCall(peer, "chat-1");
    expect(api.iceConfig).toHaveBeenCalledTimes(2);
  });

  it("are asked for the call itself when answering", async () => {
    native.decision = { action: "accept", callId: ringingCall.callId };
    api.pending.mockResolvedValue({ call: ringingCall });

    renderHook(() => useCall());

    await waitFor(() => expect(answers()).toHaveLength(1));
    expect(api.iceConfig).toHaveBeenCalledWith({ callId: ringingCall.callId });
  });
});

// A call whose path drops mid-conversation (Wi-Fi to mobile data, a lapsed
// relay) used to end on the spot with "Сбой соединения". Now the caller
// restarts ICE over the call's own signaling and the call carries on.
describe("a call that loses its connection", () => {
  const peer = { id: "user-2", displayName: "Пётр", avatarUrl: null };
  const sent = (type: string) => ws.send.mock.calls.map(([f]) => f as { type: string; data: Record<string, unknown> }).filter((f) => f.type === type);

  async function callerInCall() {
    const hook = renderHook(() => useCall());
    await hook.result.current.startCall(peer, "chat-1");
    const callId = sent("call.invite")[0].data.callId as string;
    serverSends({ event: "call.answered", data: { callId, sdp: "answer-sdp" } });
    const pc = FakePeerConnection.last!;
    await waitFor(() => expect(hook.result.current.view.phase).toBe("connecting"));
    pc.becomes("connected");
    await waitFor(() => expect(hook.result.current.view.phase).toBe("active"));
    return { ...hook, pc, callId };
  }

  it("is restarted by the caller with fresh relay credentials, and recovers", async () => {
    const { result, pc, callId } = await callerInCall();

    pc.becomes("failed");
    await waitFor(() => expect(sent("call.renegotiate")).toHaveLength(1));
    expect(api.iceConfig).toHaveBeenLastCalledWith({ callId });
    expect(pc.createOffer).toHaveBeenLastCalledWith({ iceRestart: true });
    expect(sent("call.renegotiate")[0].data).toMatchObject({ callId, kind: "offer" });
    expect(result.current.view.reconnecting).toBe(true);
    expect(result.current.view.phase).toBe("active");

    serverSends({ event: "call.renegotiate", data: { callId, kind: "answer", sdp: "restart-answer" } });
    await waitFor(() => expect(pc.setRemoteDescription).toHaveBeenLastCalledWith({ type: "answer", sdp: "restart-answer" }));
    pc.becomes("connected");
    await waitFor(() => expect(result.current.view.reconnecting).toBe(false));
    expect(result.current.view.phase).toBe("active");
    expect(sent("call.hangup")).toHaveLength(0);
  });

  it("is answered by the callee when the caller restarts it", async () => {
    native.decision = { action: "accept", callId: ringingCall.callId };
    api.pending.mockResolvedValue({ call: ringingCall });
    renderHook(() => useCall());
    await waitFor(() => expect(answers()).toHaveLength(1));
    const pc = FakePeerConnection.last!;
    pc.becomes("connected");

    serverSends({ event: "call.renegotiate", data: { callId: ringingCall.callId, kind: "offer", sdp: "restart-offer" } });
    await waitFor(() => expect(sent("call.renegotiate")).toHaveLength(1));
    expect(pc.setRemoteDescription).toHaveBeenLastCalledWith({ type: "offer", sdp: "restart-offer" });
    expect(sent("call.renegotiate")[0].data).toMatchObject({ callId: ringingCall.callId, kind: "answer", sdp: "answer-sdp" });
    expect(api.iceConfig).toHaveBeenLastCalledWith({ callId: ringingCall.callId });
  });

  it("ends when no path comes back", async () => {
    const { result, pc } = await callerInCall();
    vi.useFakeTimers({ shouldAdvanceTime: true });
    try {
      pc.becomes("disconnected");
      await vi.advanceTimersByTimeAsync(33_000); // the grace period, then the give-up
      expect(sent("call.hangup")).toHaveLength(1);
      expect(result.current.view.endedReason).toBe("Связь потеряна");
    } finally {
      vi.useRealTimers();
    }
  });

  it("still fails at once when it never connected", async () => {
    const { result } = renderHook(() => useCall());
    await result.current.startCall(peer, "chat-1");
    FakePeerConnection.last!.becomes("failed");
    await waitFor(() => expect(result.current.view.endedReason).toBe("Сбой соединения"));
    expect(sent("call.renegotiate")).toHaveLength(0);
  });
});

describe("call audio quality", () => {
  const peer = { id: "user-2", displayName: "Пётр", avatarUrl: null };

  async function connectedCall() {
    const hook = renderHook(() => useCall());
    await hook.result.current.startCall(peer, "chat-1");
    const callId = (ws.send.mock.calls[0][0] as { data: { callId: string } }).data.callId;
    serverSends({ event: "call.answered", data: { callId, sdp: "answer-sdp" } });
    const pc = FakePeerConnection.last!;
    await waitFor(() => expect(hook.result.current.view.phase).toBe("connecting"));
    pc.becomes("connected");
    await waitFor(() => expect(hook.result.current.view.phase).toBe("active"));
    return { ...hook, pc };
  }

  it("asks for the microphone as a voice call wants it", async () => {
    renderHook(() => useCall()).result.current.startCall(peer, "chat-1");
    await waitFor(() => expect(mic).toHaveBeenCalled());
    expect(mic).toHaveBeenCalledWith({
      audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true, channelCount: 1 },
    });
  });

  it("marks its audio as voice for the network once connected", async () => {
    const { pc } = await connectedCall();
    await waitFor(() => expect(pc.sender.setParameters).toHaveBeenCalledTimes(1));
    expect(pc.sender.setParameters.mock.calls[0][0].encodings[0]).toMatchObject({
      priority: "high",
      networkPriority: "high",
    });
  });

  it("says when the network is what makes it sound bad", async () => {
    const { result, pc } = await connectedCall();
    pc.stats = [{ type: "inbound-rtp", id: "a", kind: "audio", packetsReceived: 80, packetsLost: 20, jitter: 0.01 }];
    await waitFor(() => expect(result.current.view.quality).toBe("poor"), { timeout: 3_000 });
  });
});

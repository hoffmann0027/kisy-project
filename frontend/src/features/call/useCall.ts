import { useCallback, useEffect, useRef, useState } from "react";
import { wsClient } from "@shared/ws/client";
import { CALL_PUSH_EVENT } from "@shared/lib/nativePush";
import { callLog } from "@shared/lib/callLog";
import {
  onNativeCallDecision,
  reportFullScreenIntent,
  stopNativeRinging,
  takeNativeCallDecision,
  type NativeCallDecision,
} from "@shared/lib/nativeCall";
import type { CallIncomingData, ServerEvent } from "@shared/ws/events";
import { callsApi } from "@shared/api/endpoints";
import { ringtone } from "./ringtone";

export interface CallPeer {
  id: string;
  displayName: string;
  avatarUrl: string | null;
}

export type CallPhase = "idle" | "outgoing" | "incoming" | "connecting" | "active" | "ended";

export interface CallView {
  phase: CallPhase;
  peer: CallPeer | null;
  role: "caller" | "callee" | null;
  muted: boolean;
  conn: RTCPeerConnectionState | null;
  error: string | null;
  endedReason: string | null;
  startedAt: number | null;
}

const idleView: CallView = {
  phase: "idle",
  peer: null,
  role: null,
  muted: false,
  conn: null,
  error: null,
  endedReason: null,
  startedAt: null,
};

interface Session {
  callId: string;
  role: "caller" | "callee";
  peer: CallPeer;
  chatId: string;
  pc: RTCPeerConnection | null;
  localStream: MediaStream | null;
  remoteSet: boolean;
  pendingIce: RTCIceCandidateInit[];
  offerSdp?: string; // callee: remote offer held until the call is accepted
}

// useCall encapsulates the whole 1:1 audio call lifecycle: a single active
// session, its RTCPeerConnection, media, ringtone and signaling over the shared
// WebSocket. It is instantiated once by CallProvider.
export function useCall() {
  const [view, setView] = useState<CallView>(idleView);
  const session = useRef<Session | null>(null);
  const remoteAudio = useRef<HTMLAudioElement | null>(null);
  const iceCache = useRef<RTCConfiguration | null>(null);
  const endTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  if (!remoteAudio.current && typeof window !== "undefined") {
    remoteAudio.current = new Audio();
    remoteAudio.current.autoplay = true;
  }

  const fetchIce = useCallback(async (): Promise<RTCConfiguration> => {
    if (iceCache.current) return iceCache.current;
    try {
      const cfg = await callsApi.iceConfig();
      iceCache.current = { iceServers: cfg.iceServers };
    } catch {
      iceCache.current = { iceServers: [] };
    }
    return iceCache.current;
  }, []);

  const cleanupMedia = useCallback(() => {
    ringtone.stop();
    // The native notification rings on its own timer; whatever ended the call
    // here has to silence it too.
    void stopNativeRinging();
    const s = session.current;
    if (s?.pc) {
      s.pc.onicecandidate = null;
      s.pc.ontrack = null;
      s.pc.onconnectionstatechange = null;
      try {
        s.pc.close();
      } catch {
        /* already closed */
      }
    }
    s?.localStream?.getTracks().forEach((t) => t.stop());
    if (remoteAudio.current) remoteAudio.current.srcObject = null;
  }, []);

  // teardown ends the session immediately and returns the UI to idle.
  const teardown = useCallback(() => {
    cleanupMedia();
    session.current = null;
    setView(idleView);
  }, [cleanupMedia]);

  // finishWith shows a brief terminal state (e.g. "Нет ответа") then idles.
  const finishWith = useCallback(
    (message: string, isError = false) => {
      cleanupMedia();
      session.current = null;
      setView({ ...idleView, phase: "ended", endedReason: message, error: isError ? message : null });
      if (endTimer.current) clearTimeout(endTimer.current);
      endTimer.current = setTimeout(() => {
        setView((v) => (v.phase === "ended" ? idleView : v));
      }, 2500);
    },
    [cleanupMedia],
  );

  const flushIce = useCallback((s: Session) => {
    if (!s.pc) return;
    for (const c of s.pendingIce) void s.pc.addIceCandidate(c).catch(() => {});
    s.pendingIce = [];
  }, []);

  const makePc = useCallback((config: RTCConfiguration, callId: string): RTCPeerConnection => {
    const pc = new RTCPeerConnection(config);
    pc.onicecandidate = (ev) => {
      if (ev.candidate) wsClient.send({ type: "call.ice", data: { callId, candidate: ev.candidate.toJSON() } });
    };
    pc.ontrack = (ev) => {
      if (remoteAudio.current) {
        remoteAudio.current.srcObject = ev.streams[0] ?? null;
        void remoteAudio.current.play().catch(() => {});
      }
    };
    pc.onconnectionstatechange = () => {
      const st = pc.connectionState;
      setView((v) => (v.phase === "idle" || v.phase === "ended" ? v : { ...v, conn: st }));
      if (st === "connected") {
        ringtone.stop();
        setView((v) =>
          v.phase === "idle" || v.phase === "ended"
            ? v
            : { ...v, phase: "active", startedAt: v.startedAt ?? Date.now() },
        );
      } else if (st === "failed") {
        finishWith("Сбой соединения", true);
      }
    };
    return pc;
  }, [finishWith]);

  // getMic acquires the microphone, mapping denial to a friendly message.
  const getMic = useCallback(async (): Promise<MediaStream> => {
    try {
      return await navigator.mediaDevices.getUserMedia({ audio: true });
    } catch (err) {
      const name = (err as DOMException)?.name;
      if (name === "NotAllowedError" || name === "SecurityError") {
        throw new Error("Доступ к микрофону запрещён");
      }
      if (name === "NotFoundError" || name === "DevicesNotFoundError") {
        throw new Error("Микрофон не найден");
      }
      throw new Error("Не удалось получить доступ к микрофону");
    }
  }, []);

  // --- public actions ---

  const startCall = useCallback(
    async (peer: CallPeer, chatId: string) => {
      if (session.current) return; // one call at a time
      const callId = crypto.randomUUID();
      setView({ ...idleView, phase: "outgoing", peer, role: "caller" });
      let localStream: MediaStream;
      try {
        localStream = await getMic();
      } catch (e) {
        finishWith((e as Error).message, true);
        return;
      }
      const config = await fetchIce();
      const pc = makePc(config, callId);
      localStream.getTracks().forEach((t) => pc.addTrack(t, localStream));
      session.current = { callId, role: "caller", peer, chatId, pc, localStream, remoteSet: false, pendingIce: [] };

      ringtone.outgoing();
      try {
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        wsClient.send({ type: "call.invite", data: { callId, toUserId: peer.id, chatId, sdp: offer.sdp ?? "" } });
      } catch {
        finishWith("Не удалось начать звонок", true);
      }
    },
    [fetchIce, finishWith, getMic, makePc],
  );

  const accept = useCallback(async () => {
    const s = session.current;
    if (!s || s.role !== "callee") return;
    callLog("answering", s.callId);
    ringtone.stop();
    void stopNativeRinging();
    let localStream: MediaStream;
    try {
      localStream = await getMic();
    } catch (e) {
      wsClient.send({ type: "call.reject", data: { callId: s.callId } });
      finishWith((e as Error).message, true);
      return;
    }
    const config = await fetchIce();
    const pc = makePc(config, s.callId);
    localStream.getTracks().forEach((t) => pc.addTrack(t, localStream));
    s.pc = pc;
    s.localStream = localStream;
    setView((v) => ({ ...v, phase: "connecting" }));
    try {
      await pc.setRemoteDescription({ type: "offer", sdp: s.offerSdp });
      s.remoteSet = true;
      flushIce(s);
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      wsClient.send({ type: "call.answer", data: { callId: s.callId, sdp: answer.sdp ?? "" } });
    } catch {
      wsClient.send({ type: "call.hangup", data: { callId: s.callId } });
      finishWith("Не удалось установить соединение", true);
    }
  }, [fetchIce, finishWith, flushIce, getMic, makePc]);

  const reject = useCallback(() => {
    const s = session.current;
    if (!s) return;
    callLog("declining in the app", s.callId);
    wsClient.send({ type: "call.reject", data: { callId: s.callId } });
    teardown();
  }, [teardown]);

  const hangup = useCallback(() => {
    const s = session.current;
    if (!s) return;
    // Before answer the caller "cancels"; otherwise either party "hangs up".
    if (view.phase === "outgoing") {
      wsClient.send({ type: "call.cancel", data: { callId: s.callId } });
    } else {
      wsClient.send({ type: "call.hangup", data: { callId: s.callId } });
    }
    teardown();
  }, [teardown, view.phase]);

  const toggleMute = useCallback(() => {
    const track = session.current?.localStream?.getAudioTracks()[0];
    if (!track) return;
    track.enabled = !track.enabled;
    setView((v) => ({ ...v, muted: !track.enabled }));
  }, []);

  // --- inbound signaling ---

  const onIncoming = useCallback((data: CallIncomingData) => {
    if (session.current) {
      // Already busy on this client — decline the new call politely.
      wsClient.send({ type: "call.reject", data: { callId: data.callId } });
      return;
    }
    session.current = {
      callId: data.callId,
      role: "callee",
      peer: data.from,
      chatId: data.chatId,
      pc: null,
      localStream: null,
      remoteSet: false,
      pendingIce: [],
      offerSdp: data.sdp,
    };
    setView({ ...idleView, phase: "incoming", peer: data.from, role: "callee" });
    ringtone.incoming();
    callLog("ringing in the app", data.callId);
    // Only when the app is actually on screen. A backgrounded app ringing
    // through a hidden WebView is a phone that looks silent — there the
    // native notification is the ring, and taking it away would hide the call.
    if (document.visibilityState === "visible") void stopNativeRinging();
  }, []);

  /**
   * Pick up a call that started while this client had no socket.
   *
   * A phone woken by a call push missed the call.incoming frame — it was
   * published to a connection that did not exist — so the invite is fetched
   * instead. Also runs on mount and on returning to the foreground: waking to
   * a ringing phone and finding a silent app is exactly the bug this closes.
   */
  const resumePending = useCallback(async () => {
    if (session.current) return; // already ringing or talking here
    const { call } = await callsApi.pending().catch((err) => {
      callLog("/calls/pending failed", err);
      return { call: null };
    });
    if (!call) {
      callLog("no call waiting");
      return;
    }
    if (session.current) return;
    callLog("call waiting on the server", call.callId);
    onIncoming({
      callId: call.callId,
      from: { id: call.callerId, displayName: call.callerName, avatarUrl: null },
      chatId: call.chatId,
      sdp: call.offer,
    });
  }, [onIncoming]);

  /**
   * Carry out what the user tapped on the native ringing screen.
   *
   * That screen is plain Android: it can wake the phone and take the answer,
   * but it has no session and no WebRTC, so the tap only becomes a real answer
   * here (see android/.../calls/KisyCallPlugin.java).
   */
  const applyNativeDecision = useCallback(
    async (d: NativeCallDecision) => {
      if (d.action === "reject") {
        // Over REST, not the socket: the app was just started by the
        // notification and the socket may not be up yet.
        callLog("declining on the server", d.callId);
        await callsApi.reject(d.callId).catch((err) => callLog("decline failed", err));
        if (session.current?.callId === d.callId) teardown();
        return;
      }
      await resumePending();
      if (session.current?.callId !== d.callId) {
        // Answered too late: the caller hung up, or the ring timed out while
        // the app was starting.
        callLog("answered a call that is no longer ringing", d.callId);
        return;
      }
      await accept();
    },
    [accept, resumePending, teardown],
  );

  useEffect(() => {
    void reportFullScreenIntent();

    // One entry point for every way a call can turn up: a cold start from the
    // notification, a push, coming back to the foreground.
    const pickUp = async () => {
      await resumePending();
      const decided = await takeNativeCallDecision();
      if (decided) await applyNativeDecision(decided);
    };

    void pickUp();

    const onPush = () => {
      callLog("push woke the app");
      void pickUp();
    };
    const onVisible = () => {
      if (document.visibilityState === "visible") void pickUp();
    };
    const offNative = onNativeCallDecision((d) => void applyNativeDecision(d));
    window.addEventListener(CALL_PUSH_EVENT, onPush);
    document.addEventListener("visibilitychange", onVisible);
    return () => {
      offNative();
      window.removeEventListener(CALL_PUSH_EVENT, onPush);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [applyNativeDecision, resumePending]);

  useEffect(() => {
    const unsub = wsClient.subscribe((e: ServerEvent) => {
      switch (e.event) {
        case "call.incoming":
          onIncoming(e.data);
          break;
        case "call.answered": {
          const s = session.current;
          if (!s || s.callId !== e.data.callId || s.role !== "caller" || !s.pc) return;
          ringtone.stop();
          void s.pc
            .setRemoteDescription({ type: "answer", sdp: e.data.sdp })
            .then(() => {
              s.remoteSet = true;
              flushIce(s);
              setView((v) => ({ ...v, phase: "connecting" }));
            })
            .catch(() => finishWith("Сбой соединения", true));
          break;
        }
        case "call.ice": {
          const s = session.current;
          if (!s || s.callId !== e.data.callId) return;
          if (s.pc && s.remoteSet) void s.pc.addIceCandidate(e.data.candidate).catch(() => {});
          else s.pendingIce.push(e.data.candidate);
          break;
        }
        case "call.rejected":
          if (session.current?.callId === e.data.callId) finishWith("Звонок отклонён");
          break;
        case "call.canceled":
          if (session.current?.callId === e.data.callId) finishWith("Вызов отменён");
          break;
        case "call.busy":
          if (session.current?.callId === e.data.callId) finishWith("Абонент занят");
          break;
        case "call.timeout":
          if (session.current?.callId === e.data.callId) {
            finishWith(session.current?.role === "caller" ? "Нет ответа" : "Пропущенный звонок");
          }
          break;
        case "call.ended":
          if (session.current?.callId === e.data.callId) finishWith("Звонок завершён");
          break;
      }
    });
    return () => {
      unsub();
      cleanupMedia();
      if (endTimer.current) clearTimeout(endTimer.current);
    };
  }, [cleanupMedia, finishWith, flushIce, onIncoming]);

  return { view, startCall, accept, reject, hangup, toggleMute };
}

// Voice note recorder: idle → recording → processing. Wraps MediaRecorder
// (opus/webm where available), tracks elapsed time for the composer timer
// and produces the blob + duration + peak envelope on stop.
import { useCallback, useEffect, useRef, useState } from "react";
import { computePeaks } from "./waveform";

export interface VoiceNote {
  blob: Blob;
  durationMs: number;
  waveform: Uint8Array;
}

export type RecorderState = "idle" | "recording" | "processing";

// Preference order MediaRecorder actually supports across browsers.
const MIME_CANDIDATES = ["audio/webm;codecs=opus", "audio/webm", "audio/mp4"];

function pickMime(): string | undefined {
  if (typeof MediaRecorder === "undefined") return undefined;
  return MIME_CANDIDATES.find((m) => MediaRecorder.isTypeSupported(m));
}

/** Decode the recording and build its peak envelope; empty on failure —
 * a voice note without bars still plays fine. */
async function buildWaveform(blob: Blob): Promise<Uint8Array> {
  try {
    const ctx = new AudioContext();
    try {
      const decoded = await ctx.decodeAudioData(await blob.arrayBuffer());
      return computePeaks(decoded.getChannelData(0));
    } finally {
      void ctx.close();
    }
  } catch {
    return new Uint8Array(0);
  }
}

export function useVoiceRecorder() {
  const [state, setState] = useState<RecorderState>("idle");
  const [elapsedMs, setElapsedMs] = useState(0);

  const recorder = useRef<MediaRecorder | null>(null);
  const stream = useRef<MediaStream | null>(null);
  const chunks = useRef<Blob[]>([]);
  const startedAt = useRef(0);
  const timer = useRef<number>();
  const cancelled = useRef(false);

  const cleanup = useCallback(() => {
    window.clearInterval(timer.current);
    stream.current?.getTracks().forEach((t) => t.stop());
    stream.current = null;
    recorder.current = null;
    chunks.current = [];
    setElapsedMs(0);
  }, []);

  useEffect(() => cleanup, [cleanup]);

  const start = useCallback(async (): Promise<boolean> => {
    if (state !== "idle") return false;
    try {
      const media = await navigator.mediaDevices.getUserMedia({ audio: true });
      stream.current = media;
      const mime = pickMime();
      const rec = new MediaRecorder(media, mime ? { mimeType: mime } : undefined);
      chunks.current = [];
      cancelled.current = false;
      rec.ondataavailable = (e) => {
        if (e.data.size > 0) chunks.current.push(e.data);
      };
      rec.start(250);
      recorder.current = rec;
      startedAt.current = performance.now();
      setElapsedMs(0);
      timer.current = window.setInterval(() => setElapsedMs(performance.now() - startedAt.current), 200);
      setState("recording");
      return true;
    } catch {
      cleanup();
      return false; // microphone denied/unavailable
    }
  }, [state, cleanup]);

  /** Stop and produce the note; null when cancelled or nothing captured. */
  const stop = useCallback((): Promise<VoiceNote | null> => {
    const rec = recorder.current;
    if (!rec || state !== "recording") return Promise.resolve(null);
    setState("processing");
    return new Promise((resolve) => {
      rec.onstop = async () => {
        const durationMs = Math.round(performance.now() - startedAt.current);
        const blob = new Blob(chunks.current, { type: rec.mimeType || "audio/webm" });
        const wasCancelled = cancelled.current;
        cleanup();
        setState("idle");
        if (wasCancelled || blob.size === 0 || durationMs < 300) {
          resolve(null);
          return;
        }
        resolve({ blob, durationMs, waveform: await buildWaveform(blob) });
      };
      rec.stop();
    });
  }, [state, cleanup]);

  const cancel = useCallback(() => {
    cancelled.current = true;
    void stop();
  }, [stop]);

  return { state, elapsedMs, start, stop, cancel };
}

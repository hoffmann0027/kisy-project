// Shared voice player: ONE HTMLAudioElement for the whole app — starting a
// note stops whatever was playing, exactly like TG/WA. Playback rate and the
// listened-set persist in localStorage (non-secret UI state).
import { create } from "zustand";
import { isObjectUrl, mediaSrcSync, resolveMediaSrc } from "@shared/lib/mediaSrc";

export type PlaybackRate = 1 | 1.5 | 2;

const RATE_KEY = "kisy-voice-rate";
const LISTENED_KEY = "kisy-voice-listened";
const LISTENED_CAP = 500; // oldest ids fall off; the dot is a convenience

function loadRate(): PlaybackRate {
  const v = Number(localStorage.getItem(RATE_KEY));
  return v === 1.5 || v === 2 ? v : 1;
}

function loadListened(): string[] {
  try {
    const raw = JSON.parse(localStorage.getItem(LISTENED_KEY) ?? "[]");
    return Array.isArray(raw) ? raw.filter((x) => typeof x === "string") : [];
  } catch {
    return [];
  }
}

interface VoicePlayerState {
  /** Attachment id of the currently loaded note (playing or paused). */
  activeId: string | null;
  playing: boolean;
  /** 0..1 of the active note. */
  progress: number;
  rate: PlaybackRate;
  listened: string[];
  toggle: (attachmentId: string, url: string) => void;
  seek: (fraction: number) => void;
  cycleRate: () => void;
  isListened: (attachmentId: string) => boolean;
}

// Lazy singleton — created on first playback, reused forever.
let audio: HTMLAudioElement | null = null;
// Blob URL backing the note currently loaded, when the shell had to fetch the
// bytes itself; revoked as soon as another note takes over.
let currentObjectUrl: string | null = null;

function ensureAudio(get: () => VoicePlayerState, set: (p: Partial<VoicePlayerState>) => void): HTMLAudioElement {
  if (audio) return audio;
  audio = new Audio();
  audio.preload = "auto";
  audio.ontimeupdate = () => {
    if (!audio!.duration) return;
    set({ progress: audio!.currentTime / audio!.duration });
  };
  audio.onended = () => set({ playing: false, progress: 1 });
  audio.onpause = () => set({ playing: false });
  audio.onplay = () => set({ playing: true, rate: get().rate });
  return audio;
}

export const useVoicePlayer = create<VoicePlayerState>((set, get) => ({
  activeId: null,
  playing: false,
  progress: 0,
  rate: loadRate(),
  listened: loadListened(),

  toggle: (attachmentId, url) => {
    const a = ensureAudio(get, (p) => set(p));
    const { activeId, playing, listened } = get();

    if (activeId === attachmentId) {
      if (playing) {
        a.pause();
      } else {
        a.playbackRate = get().rate;
        void a.play();
      }
      return;
    }

    // Switching notes: the singleton guarantees the previous one stops.
    // The note lives behind the authenticated API, which an <audio> element
    // cannot reach in the mobile shell, so resolve it first (a no-op on web).
    if (currentObjectUrl) {
      URL.revokeObjectURL(currentObjectUrl);
      currentObjectUrl = null;
    }
    a.playbackRate = get().rate;
    a.currentTime = 0;
    const nextListened = listened.includes(attachmentId)
      ? listened
      : [...listened, attachmentId].slice(-LISTENED_CAP);
    localStorage.setItem(LISTENED_KEY, JSON.stringify(nextListened));
    set({ activeId: attachmentId, progress: 0, listened: nextListened });

    // Web: play in the same task as the click, or the autoplay policy may
    // refuse it. Only the mobile shell, which has to fetch the bytes with the
    // session token, takes the asynchronous path.
    const direct = mediaSrcSync(url);
    if (direct !== null) {
      a.src = direct;
      void a.play();
      return;
    }

    void resolveMediaSrc(url).then((src) => {
      // A newer note may have been started while this one resolved.
      if (get().activeId !== attachmentId) {
        if (isObjectUrl(src)) URL.revokeObjectURL(src);
        return;
      }
      if (isObjectUrl(src)) currentObjectUrl = src;
      a.src = src;
      void a.play();
    });
  },

  seek: (fraction) => {
    if (!audio || !audio.duration || get().activeId === null) return;
    audio.currentTime = Math.max(0, Math.min(1, fraction)) * audio.duration;
    set({ progress: fraction });
  },

  cycleRate: () => {
    const next: PlaybackRate = get().rate === 1 ? 1.5 : get().rate === 1.5 ? 2 : 1;
    localStorage.setItem(RATE_KEY, String(next));
    if (audio) audio.playbackRate = next;
    set({ rate: next });
  },

  isListened: (attachmentId) => get().listened.includes(attachmentId),
}));

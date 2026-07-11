// Waveform helpers: a voice note travels with a compact peak envelope
// (one byte per bar, stage A schema) that the bubble renders as bars.

/** Bars in a recorded envelope — small enough for any bubble width. */
export const WAVEFORM_BARS = 48;

/**
 * Downsample raw PCM into `bars` peak values (0..255). Peaks (not RMS):
 * speech looks livelier and silence stays visibly flat.
 */
export function computePeaks(channel: Float32Array, bars = WAVEFORM_BARS): Uint8Array {
  const out = new Uint8Array(bars);
  if (channel.length === 0) return out;
  const window = Math.max(1, Math.floor(channel.length / bars));
  let max = 0;
  const peaks = new Float32Array(bars);
  for (let b = 0; b < bars; b++) {
    const start = b * window;
    const end = b === bars - 1 ? channel.length : Math.min(start + window, channel.length);
    let peak = 0;
    for (let i = start; i < end; i++) {
      const v = Math.abs(channel[i]);
      if (v > peak) peak = v;
    }
    peaks[b] = peak;
    if (peak > max) max = peak;
  }
  // Normalize to the loudest bar so quiet recordings still show shape.
  if (max > 0) {
    for (let b = 0; b < bars; b++) out[b] = Math.round((peaks[b] / max) * 255);
  }
  return out;
}

export function waveformToBase64(peaks: Uint8Array): string {
  let s = "";
  for (const b of peaks) s += String.fromCharCode(b);
  return btoa(s);
}

export function waveformFromBase64(b64: string): Uint8Array {
  try {
    const s = atob(b64);
    const out = new Uint8Array(s.length);
    for (let i = 0; i < s.length; i++) out[i] = s.charCodeAt(i);
    return out;
  } catch {
    return new Uint8Array(0);
  }
}

/** mm:ss for player timestamps. */
export function formatDuration(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000));
  const m = Math.floor(total / 60);
  const s = total % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

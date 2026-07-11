// @vitest-environment node
import { describe, expect, it } from "vitest";
import { computePeaks, formatDuration, waveformFromBase64, waveformToBase64, WAVEFORM_BARS } from "./waveform";

describe("computePeaks", () => {
  it("produces the requested number of bars, normalized to the loudest", () => {
    // First half silence, second half a 0.5-amplitude tone.
    const pcm = new Float32Array(4800);
    for (let i = 2400; i < 4800; i++) pcm[i] = 0.5 * Math.sin(i / 10);

    const peaks = computePeaks(pcm, 48);
    expect(peaks).toHaveLength(48);
    // Silent half stays near zero; loud half reaches full scale after
    // normalization.
    expect(Math.max(...peaks.slice(0, 20))).toBe(0);
    expect(Math.max(...peaks.slice(28))).toBe(255);
  });

  it("handles empty and tiny inputs without dividing by zero", () => {
    expect(computePeaks(new Float32Array(0))).toHaveLength(WAVEFORM_BARS);
    const tiny = computePeaks(new Float32Array([0.1, -0.9]), 8);
    expect(tiny).toHaveLength(8);
    expect(Math.max(...tiny)).toBe(255);
  });
});

describe("waveform base64 round-trip", () => {
  it("encodes and decodes losslessly", () => {
    const peaks = new Uint8Array([0, 1, 127, 200, 255]);
    expect(Array.from(waveformFromBase64(waveformToBase64(peaks)))).toEqual(Array.from(peaks));
  });

  it("returns empty on malformed input", () => {
    expect(waveformFromBase64("%%%not-base64%%%")).toHaveLength(0);
  });
});

describe("formatDuration", () => {
  it("renders mm:ss", () => {
    expect(formatDuration(0)).toBe("0:00");
    expect(formatDuration(4200)).toBe("0:04");
    expect(formatDuration(61_000)).toBe("1:01");
    expect(formatDuration(600_000)).toBe("10:00");
  });
});

import { describe, expect, it } from "vitest";
import { rateQuality, readSample, type QualitySample } from "./quality";

const report = (entries: Array<Record<string, unknown>>) =>
  ({ forEach: (fn: (e: unknown) => void) => entries.forEach(fn) }) as unknown as RTCStatsReport;

const sample = (s: Partial<QualitySample>): QualitySample => ({
  packetsReceived: 0,
  packetsLost: 0,
  jitter: 0.01,
  rtt: 0.05,
  ...s,
});

describe("readSample", () => {
  it("reads the incoming audio and the path the transport selected", () => {
    const s = readSample(
      report([
        { type: "inbound-rtp", id: "v", kind: "video", packetsReceived: 999, packetsLost: 999 },
        { type: "inbound-rtp", id: "a", kind: "audio", packetsReceived: 500, packetsLost: 7, jitter: 0.02 },
        { type: "transport", id: "t", selectedCandidatePairId: "p2" },
        { type: "candidate-pair", id: "p1", nominated: true, state: "succeeded", currentRoundTripTime: 0.9 },
        { type: "candidate-pair", id: "p2", nominated: true, state: "succeeded", currentRoundTripTime: 0.08 },
      ]),
    );
    expect(s).toEqual({ packetsReceived: 500, packetsLost: 7, jitter: 0.02, rtt: 0.08 });
  });

  it("is null before any audio arrives", () => {
    expect(readSample(report([{ type: "transport", id: "t" }]))).toBeNull();
  });
});

describe("rateQuality", () => {
  const prev = sample({ packetsReceived: 1000, packetsLost: 10 });

  it("rates a clean interval good", () => {
    expect(rateQuality(prev, sample({ packetsReceived: 1100, packetsLost: 10 }))).toBe("good");
  });

  it("judges the interval, not the whole call", () => {
    // 10 lost of the first 1000 is history; 15 of the last 100 is now.
    expect(rateQuality(prev, sample({ packetsReceived: 1085, packetsLost: 25 }))).toBe("poor");
  });

  it("rates moderate loss, delay or jitter fair", () => {
    expect(rateQuality(prev, sample({ packetsReceived: 1096, packetsLost: 14 }))).toBe("fair");
    expect(rateQuality(prev, sample({ packetsReceived: 1100, packetsLost: 10, rtt: 0.4 }))).toBe("fair");
    expect(rateQuality(prev, sample({ packetsReceived: 1100, packetsLost: 10, jitter: 0.06 }))).toBe("fair");
  });

  it("rates a long delay poor", () => {
    expect(rateQuality(prev, sample({ packetsReceived: 1100, packetsLost: 10, rtt: 0.8 }))).toBe("poor");
  });

  it("rates silence on the wire poor", () => {
    expect(rateQuality(prev, sample({ packetsReceived: 1000, packetsLost: 10 }))).toBe("poor");
  });
});

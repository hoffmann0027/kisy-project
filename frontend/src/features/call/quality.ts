// Connection quality of a live call, read from WebRTC statistics.
//
// A call on a bad network sounds broken, and without a hint the person blames
// the app. Every couple of seconds the call samples what actually arrives —
// packets lost against packets received, jitter, round-trip time — and the
// call screen says when the network is the problem.

export type CallQuality = "good" | "fair" | "poor";

/** Cumulative counters as RTCPeerConnection.getStats() reports them. */
export interface QualitySample {
  packetsReceived: number;
  packetsLost: number;
  /** Seconds, of the incoming audio. */
  jitter: number | null;
  /** Seconds, of the selected candidate pair. */
  rtt: number | null;
}

type StatsEntry = Record<string, unknown> & { type: string; id: string };

const num = (v: unknown): number | null => (typeof v === "number" && Number.isFinite(v) ? v : null);

/** The incoming audio stream and the path it takes, or null before media flows. */
export function readSample(report: RTCStatsReport): QualitySample | null {
  const entries: StatsEntry[] = [];
  report.forEach((e) => entries.push(e as StatsEntry));

  const inbound = entries.find((e) => e.type === "inbound-rtp" && e.kind === "audio");
  if (!inbound) return null;

  const transport = entries.find((e) => e.type === "transport" && typeof e.selectedCandidatePairId === "string");
  const pair =
    entries.find((e) => e.type === "candidate-pair" && e.id === transport?.selectedCandidatePairId) ??
    entries.find((e) => e.type === "candidate-pair" && e.nominated === true && e.state === "succeeded");

  return {
    packetsReceived: num(inbound.packetsReceived) ?? 0,
    packetsLost: Math.max(0, num(inbound.packetsLost) ?? 0),
    jitter: num(inbound.jitter),
    rtt: num(pair?.currentRoundTripTime),
  };
}

// Where speech stops being comfortable: Opus with in-band FEC hides a few
// percent of loss, beyond ~10% words go missing; past ~350 ms of round trip
// people start talking over each other, past ~700 ms a conversation breaks.
const POOR = { loss: 0.1, rtt: 0.7, jitter: 0.1 };
const FAIR = { loss: 0.03, rtt: 0.35, jitter: 0.05 };

/** Rates the interval since prev (or the whole call, on the first sample). */
export function rateQuality(prev: QualitySample | null, cur: QualitySample): CallQuality {
  const received = cur.packetsReceived - (prev?.packetsReceived ?? 0);
  const lost = Math.max(0, cur.packetsLost - (prev?.packetsLost ?? 0));
  // Nothing arrived at all: the audio has stopped, whatever the counters say.
  if (received <= 0) return "poor";
  const loss = lost / (lost + received);
  const rtt = cur.rtt ?? 0;
  const jitter = cur.jitter ?? 0;
  if (loss >= POOR.loss || rtt >= POOR.rtt || jitter >= POOR.jitter) return "poor";
  if (loss >= FAIR.loss || rtt >= FAIR.rtt || jitter >= FAIR.jitter) return "fair";
  return "good";
}

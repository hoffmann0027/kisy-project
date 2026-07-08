// A self-contained call ring/ringback generated with the Web Audio API — no
// binary asset to ship, trivially stoppable, and it obeys the browser/OS
// output (muting the tab silences it). Incoming uses a two-tone ring; outgoing
// a single ringback beep. Kept module-global because at most one call rings at
// a time.

let ctx: AudioContext | null = null;
let timer: ReturnType<typeof setInterval> | null = null;

function audioCtx(): AudioContext | null {
  if (typeof window === "undefined") return null;
  const Ctor = window.AudioContext ?? (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctor) return null;
  if (!ctx) ctx = new Ctor();
  // Autoplay policies may leave the context suspended until a user gesture;
  // resume best-effort (the incoming ring may stay silent until interaction).
  if (ctx.state === "suspended") void ctx.resume().catch(() => {});
  return ctx;
}

function beep(c: AudioContext, freq: number, at: number, dur: number) {
  const osc = c.createOscillator();
  const gain = c.createGain();
  osc.type = "sine";
  osc.frequency.value = freq;
  osc.connect(gain);
  gain.connect(c.destination);
  gain.gain.setValueAtTime(0.0001, at);
  gain.gain.exponentialRampToValueAtTime(0.12, at + 0.03);
  gain.gain.exponentialRampToValueAtTime(0.0001, at + dur);
  osc.start(at);
  osc.stop(at + dur + 0.03);
}

function loop(render: (c: AudioContext) => void, everyMs: number) {
  stop();
  const c = audioCtx();
  if (!c) return;
  render(c);
  timer = setInterval(() => {
    const cc = audioCtx();
    if (cc) render(cc);
  }, everyMs);
}

export const ringtone = {
  /** Ringing tone for an incoming call. */
  incoming() {
    loop((c) => {
      const t = c.currentTime;
      beep(c, 520, t, 0.4);
      beep(c, 660, t + 0.5, 0.4);
    }, 2000);
  },
  /** Ringback tone while an outgoing call is dialing. */
  outgoing() {
    loop((c) => beep(c, 440, c.currentTime, 0.8), 2200);
  },
  stop() {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  },
};

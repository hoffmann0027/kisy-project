// Player store behavior with a stubbed Audio element: single shared
// instance, listened-set persistence, rate cycling.
import { beforeEach, describe, expect, it, vi } from "vitest";

class FakeAudio {
  static instances = 0;
  src = "";
  currentTime = 0;
  duration = 10;
  playbackRate = 1;
  preload = "";
  onplay: (() => void) | null = null;
  onpause: (() => void) | null = null;
  onended: (() => void) | null = null;
  ontimeupdate: (() => void) | null = null;
  constructor() {
    FakeAudio.instances++;
  }
  play() {
    this.onplay?.();
    return Promise.resolve();
  }
  pause() {
    this.onpause?.();
  }
}

vi.stubGlobal("Audio", FakeAudio);

import { useVoicePlayer } from "./player";

describe("voice player store", () => {
  beforeEach(() => {
    localStorage.clear();
    FakeAudio.instances = 0;
    useVoicePlayer.setState({ activeId: null, playing: false, progress: 0, listened: [], rate: 1 });
  });

  it("plays, pauses and switches notes on one shared Audio instance", () => {
    const p = useVoicePlayer.getState();
    p.toggle("a1", "/api/v1/attachments/a1");
    expect(useVoicePlayer.getState().activeId).toBe("a1");
    expect(useVoicePlayer.getState().playing).toBe(true);

    // Same note toggles pause.
    useVoicePlayer.getState().toggle("a1", "/api/v1/attachments/a1");
    expect(useVoicePlayer.getState().playing).toBe(false);

    // A different note takes over the same instance.
    useVoicePlayer.getState().toggle("a2", "/api/v1/attachments/a2");
    expect(useVoicePlayer.getState().activeId).toBe("a2");
    expect(FakeAudio.instances).toBe(1);
  });

  it("marks notes as listened and persists the set", () => {
    useVoicePlayer.getState().toggle("a1", "/u1");
    expect(useVoicePlayer.getState().listened).toContain("a1");
    expect(JSON.parse(localStorage.getItem("kisy-voice-listened")!)).toContain("a1");
  });

  it("cycles playback rate 1 → 1.5 → 2 → 1 and persists it", () => {
    const s = useVoicePlayer.getState();
    s.cycleRate();
    expect(useVoicePlayer.getState().rate).toBe(1.5);
    useVoicePlayer.getState().cycleRate();
    expect(useVoicePlayer.getState().rate).toBe(2);
    useVoicePlayer.getState().cycleRate();
    expect(useVoicePlayer.getState().rate).toBe(1);
    expect(localStorage.getItem("kisy-voice-rate")).toBe("1");
  });
});

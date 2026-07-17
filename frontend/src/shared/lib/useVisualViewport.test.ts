import { renderHook } from "@testing-library/react";
import { act } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useVisualViewport } from "./useVisualViewport";

/** Minimal stand-in for window.visualViewport with controllable geometry. */
function fakeViewport(height: number, offsetTop = 0) {
  const listeners: Record<string, Set<() => void>> = { resize: new Set(), scroll: new Set() };
  return {
    height,
    offsetTop,
    addEventListener: (t: string, cb: () => void) => listeners[t]?.add(cb),
    removeEventListener: (t: string, cb: () => void) => listeners[t]?.delete(cb),
    emit(t: string) {
      listeners[t]?.forEach((cb) => cb());
    },
    set(next: { height?: number; offsetTop?: number }) {
      if (next.height !== undefined) this.height = next.height;
      if (next.offsetTop !== undefined) this.offsetTop = next.offsetTop;
    },
    listenerCount: () => listeners.resize.size + listeners.scroll.size,
  };
}

function setViewport(vv: ReturnType<typeof fakeViewport> | undefined) {
  Object.defineProperty(window, "visualViewport", { value: vv, configurable: true, writable: true });
}

function setInnerHeight(h: number) {
  Object.defineProperty(window, "innerHeight", { value: h, configurable: true, writable: true });
}

describe("useVisualViewport", () => {
  afterEach(() => {
    setViewport(undefined);
    vi.restoreAllMocks();
  });

  it("returns 0 when visualViewport is unavailable (desktop)", () => {
    setViewport(undefined);
    const { result } = renderHook(() => useVisualViewport());
    expect(result.current).toBe(0);
  });

  it("reports the keyboard overlap between layout and visual viewport", () => {
    setInnerHeight(800);
    setViewport(fakeViewport(500)); // 800 - 500 - 0 = 300
    const { result } = renderHook(() => useVisualViewport());
    expect(result.current).toBe(300);
  });

  it("subtracts the visual-viewport offsetTop", () => {
    setInnerHeight(800);
    setViewport(fakeViewport(500, 40)); // 800 - 500 - 40 = 260
    const { result } = renderHook(() => useVisualViewport());
    expect(result.current).toBe(260);
  });

  it("treats a sub-threshold shrink (address bar, not keyboard) as 0", () => {
    setInnerHeight(800);
    setViewport(fakeViewport(760)); // overlap 40 < 60px threshold
    const { result } = renderHook(() => useVisualViewport());
    expect(result.current).toBe(0);
  });

  it("updates on a coalesced resize event", () => {
    // Run the rAF the hook schedules synchronously so we can assert the result.
    vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb: FrameRequestCallback) => {
      cb(0);
      return 1;
    });
    setInnerHeight(800);
    const vv = fakeViewport(800);
    setViewport(vv);
    const { result } = renderHook(() => useVisualViewport());
    expect(result.current).toBe(0);

    act(() => {
      vv.set({ height: 500 });
      vv.emit("resize");
    });
    expect(result.current).toBe(300);
  });

  it("removes its listeners on unmount", () => {
    setInnerHeight(800);
    const vv = fakeViewport(500);
    setViewport(vv);
    const { unmount } = renderHook(() => useVisualViewport());
    expect(vv.listenerCount()).toBeGreaterThan(0);
    unmount();
    expect(vv.listenerCount()).toBe(0);
  });
});

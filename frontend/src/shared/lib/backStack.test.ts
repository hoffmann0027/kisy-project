import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import {
  closeTopOverlay,
  handleBackPress,
  openOverlayCount,
  pushBackHandler,
  removeBackHandler,
  useBackHandler,
} from "./backStack";

// Back on Android closed the app from anywhere — session, socket and open chat
// gone, and the way back in was a cold start. These are the rules that replaced
// that: close what is open, then step back, and only then get out of the way.

describe("the overlay stack", () => {
  it("closes the most recently opened overlay first", () => {
    const order: string[] = [];
    const outer = pushBackHandler(() => order.push("outer"));
    const inner = pushBackHandler(() => order.push("inner"));

    expect(closeTopOverlay()).toBe(true);
    removeBackHandler(inner);
    expect(closeTopOverlay()).toBe(true);
    removeBackHandler(outer);

    // A modal opened from inside another modal is the one on screen.
    expect(order).toEqual(["inner", "outer"]);
    expect(openOverlayCount()).toBe(0);
  });

  it("reports that nothing was open", () => {
    expect(closeTopOverlay()).toBe(false);
  });

  it("removes a handler from the middle without disturbing the rest", () => {
    const hit: string[] = [];
    const a = pushBackHandler(() => hit.push("a"));
    const b = pushBackHandler(() => hit.push("b"));
    const c = pushBackHandler(() => hit.push("c"));

    removeBackHandler(b); // closed by its own button rather than by back
    closeTopOverlay();

    expect(hit).toEqual(["c"]);
    removeBackHandler(a);
    removeBackHandler(c);
  });
});

describe("useBackHandler", () => {
  it("claims the gesture only while open, and gives it up on unmount", () => {
    const onBack = vi.fn();
    const { rerender, unmount } = renderHook(({ open }) => useBackHandler(open, onBack), {
      initialProps: { open: false },
    });
    expect(openOverlayCount()).toBe(0);

    rerender({ open: true });
    expect(openOverlayCount()).toBe(1);
    closeTopOverlay();
    expect(onBack).toHaveBeenCalledTimes(1);

    rerender({ open: false });
    expect(openOverlayCount()).toBe(0);
    unmount();
    expect(openOverlayCount()).toBe(0);
  });

  it("does not lose its place when the callback changes", () => {
    // A handler closing over changing state re-renders constantly. Moving it
    // to the top of the stack on every render would steal the gesture from
    // whatever opened after it.
    const { rerender } = renderHook(({ fn }) => useBackHandler(true, fn), {
      initialProps: { fn: vi.fn() },
    });
    const later = pushBackHandler(vi.fn());

    const newest = vi.fn();
    rerender({ fn: newest });

    closeTopOverlay(); // still the one that opened last
    expect(newest).not.toHaveBeenCalled();
    removeBackHandler(later);

    // ...and the latest callback is the one that runs when its turn comes.
    closeTopOverlay();
    expect(newest).toHaveBeenCalledTimes(1);
  });
});

describe("the shared Modal", () => {
  it("is closed by back, without knowing anything about Android", async () => {
    // The wiring lives in one place on purpose: 19 modals render through this
    // component, and a new one gets the behaviour for free.
    const { createElement } = await import("react");
    const { Modal } = await import("@shared/ui/Modal");
    const { render } = await import("@testing-library/react");
    const onClose = vi.fn();

    render(createElement(Modal, { open: true, title: "Профиль", onClose, children: null }));

    expect(closeTopOverlay()).toBe(true);
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

describe("one press of back", () => {
  const deps = () => ({ goBack: vi.fn(), minimize: vi.fn() });

  it("closes an overlay before touching navigation", () => {
    const d = deps();
    const id = pushBackHandler(vi.fn());
    handleBackPress({ historyIndex: 5, ...d });
    expect(d.goBack).not.toHaveBeenCalled();
    expect(d.minimize).not.toHaveBeenCalled();
    removeBackHandler(id);
  });

  it("steps back when there is somewhere to go", () => {
    const d = deps();
    handleBackPress({ historyIndex: 3, ...d });
    expect(d.goBack).toHaveBeenCalledTimes(1);
    expect(d.minimize).not.toHaveBeenCalled();
  });

  it("minimises instead of closing at the first screen", () => {
    // Closing would drop the session and the socket, and re-entry is a cold
    // start — which is how a password screen ended up in front of a call.
    const d = deps();
    handleBackPress({ historyIndex: 0, ...d });
    expect(d.minimize).toHaveBeenCalledTimes(1);
    expect(d.goBack).not.toHaveBeenCalled();
  });
});

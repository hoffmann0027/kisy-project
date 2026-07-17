import { useEffect, useState } from "react";

/** Below this the shrink is browser chrome (the iOS address bar collapsing),
 *  not a keyboard — treating it as one would make the composer jitter. */
const KEYBOARD_MIN_PX = 60;

/**
 * Height in CSS px currently covered by the on-screen keyboard.
 *
 * When the keyboard opens, mobile browsers shrink the *visual* viewport but
 * leave the *layout* viewport (and therefore `100dvh`, `100%` and fixed
 * positioning) untouched — which is why the composer and modal fields end up
 * underneath it. The difference between the two viewports is that overlap.
 *
 * Returns 0 where `visualViewport` is unavailable (desktop), so callers can
 * subtract it unconditionally.
 */
export function useVisualViewport(): number {
  const [inset, setInset] = useState(0);

  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return; // desktop / unsupported → stays 0

    let frame = 0;

    const measure = () => {
      frame = 0;
      // offsetTop covers the page being scrolled *within* the visual viewport,
      // which iOS does to keep the focused field visible.
      const overlap = window.innerHeight - vv.height - vv.offsetTop;
      const next = overlap < KEYBOARD_MIN_PX ? 0 : Math.round(overlap);
      setInset((prev) => (prev === next ? prev : next));
    };

    // resize/scroll fire per animation frame while the keyboard animates;
    // coalesce them so we lay out at most once a frame.
    const schedule = () => {
      if (frame) return;
      frame = requestAnimationFrame(measure);
    };

    measure();
    vv.addEventListener("resize", schedule);
    vv.addEventListener("scroll", schedule);
    return () => {
      if (frame) cancelAnimationFrame(frame);
      vv.removeEventListener("resize", schedule);
      vv.removeEventListener("scroll", schedule);
    };
  }, []);

  return inset;
}

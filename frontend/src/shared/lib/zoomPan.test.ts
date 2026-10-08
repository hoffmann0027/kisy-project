import { describe, expect, it } from "vitest";
import { IDENTITY, MAX_SCALE, clampPan, swipeDirection, toggleZoom, zoomTo } from "./zoomPan";

// A user's suggestion: zooming into pictures in chats was clumsy — a tap
// doubled the size around the centre and nothing else; the edges of the
// picture could not be reached. These are the rules the viewer now follows.

const stage = { w: 400, h: 800 };
const image = { w: 400, h: 300 };

describe("zooming", () => {
  it("keeps the point under the fingers where it is", () => {
    const at = { x: 100, y: -50 };
    const v = zoomTo(IDENTITY, 2, at);
    // The image point that was at `at` (image coords: at / scale) is still there.
    const before = { x: (at.x - IDENTITY.x) / IDENTITY.scale, y: (at.y - IDENTITY.y) / IDENTITY.scale };
    expect(v.x + before.x * v.scale).toBeCloseTo(at.x);
    expect(v.y + before.y * v.scale).toBeCloseTo(at.y);
  });

  it("never shrinks below fitting the screen, nor grows without limit", () => {
    expect(zoomTo(IDENTITY, 0.3, { x: 0, y: 0 }).scale).toBe(1);
    expect(zoomTo(IDENTITY, 40, { x: 0, y: 0 }).scale).toBe(MAX_SCALE);
  });

  it("double tap goes in at the tapped point, and back to fit", () => {
    const zoomed = toggleZoom(IDENTITY, { x: 150, y: 0 });
    expect(zoomed.scale).toBeGreaterThan(2);
    expect(zoomed.x).toBeLessThan(0); // the right side came towards the middle
    expect(toggleZoom(zoomed, { x: 0, y: 0 })).toEqual(IDENTITY);
  });
});

describe("panning", () => {
  it("reaches the edges of a zoomed picture but not past them", () => {
    const v = { scale: 2, x: 5000, y: 5000 };
    const c = clampPan(v, image, stage);
    expect(c.x).toBe(200); // (400*2 - 400) / 2
    expect(c.y).toBe(0); // 300*2 is still shorter than the stage: stays centred
    expect(clampPan({ scale: 2, x: -5000, y: 0 }, image, stage).x).toBe(-200);
  });

  it("does not move an unzoomed picture", () => {
    expect(clampPan({ scale: 1, x: 80, y: 40 }, image, stage)).toEqual({ scale: 1, x: 0, y: 0 });
  });
});

describe("swiping between pictures", () => {
  it("turns on a clear horizontal swipe only", () => {
    expect(swipeDirection(-120, 10)).toBe(1); // to the next
    expect(swipeDirection(120, -10)).toBe(-1); // to the previous
    expect(swipeDirection(30, 0)).toBe(0); // too short
    expect(swipeDirection(-90, 100)).toBe(0); // mostly vertical
  });
});

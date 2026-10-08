import { describe, expect, it } from "vitest";
import { floatingPosition } from "./EmojiPicker";

// A floating picker opens above its anchor when it fits there, below when only
// there it fits, and never leaves the window.
const W = 300;
const H = 320;
const anchor = (top: number, left = 500) => ({ top, bottom: top + 30, left, width: 30 });

describe("floatingPosition", () => {
  it("opens above when there is room", () => {
    expect(floatingPosition(anchor(600), W, H, 1200, 900).top).toBe(600 - 8 - H);
  });

  it("opens below a message at the top of the chat", () => {
    expect(floatingPosition(anchor(120), W, H, 1200, 900).top).toBe(120 + 30 + 8);
  });

  it("stays inside a window too short for either side", () => {
    const p = floatingPosition(anchor(200), W, H, 1200, 400);
    expect(p.top).toBeGreaterThanOrEqual(8);
    expect(p.top + H).toBeLessThanOrEqual(400 - 8);
  });

  it("stays inside the window sideways", () => {
    expect(floatingPosition(anchor(600, 1180), W, H, 1200, 900).left).toBe(1200 - W - 8);
    expect(floatingPosition(anchor(600, 0), W, H, 1200, 900).left).toBe(8);
  });
});

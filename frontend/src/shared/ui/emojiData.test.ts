// @vitest-environment node
import { describe, expect, it } from "vitest";
import { ALL_EMOJIS, EMOJI_CATEGORIES, searchEmojis } from "./emojiData";

describe("emoji data", () => {
  it("has non-empty categories and a flattened list", () => {
    expect(EMOJI_CATEGORIES.length).toBeGreaterThan(0);
    for (const c of EMOJI_CATEGORIES) expect(c.emojis.length).toBeGreaterThan(0);
    expect(ALL_EMOJIS.length).toBe(EMOJI_CATEGORIES.reduce((n, c) => n + c.emojis.length, 0));
  });

  it("searches by russian and english keywords and by char", () => {
    expect(searchEmojis("лайк").map((e) => e.char)).toContain("👍");
    expect(searchEmojis("fire").map((e) => e.char)).toContain("🔥");
    expect(searchEmojis("сердце").length).toBeGreaterThan(1);
    expect(searchEmojis("👍").map((e) => e.char)).toContain("👍");
  });

  it("empty query returns nothing", () => {
    expect(searchEmojis("")).toHaveLength(0);
    expect(searchEmojis("   ")).toHaveLength(0);
  });
});

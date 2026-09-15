import { describe, expect, it } from "vitest";
import { applyReaction } from "./queries";

describe("applyReaction — one reaction per reader", () => {
  it("replaces the reader's previous emoji instead of adding a second one", () => {
    const before = [
      { emoji: "👍", count: 3, mine: true },
      { emoji: "🔥", count: 1, mine: false },
    ];
    expect(applyReaction(before, "🔥", true)).toEqual([
      { emoji: "👍", count: 2, mine: false },
      { emoji: "🔥", count: 2, mine: true },
    ]);
  });

  it("drops a chip whose only reaction was the reader's", () => {
    const before = [{ emoji: "😂", count: 1, mine: true }];
    expect(applyReaction(before, "👏", true)).toEqual([{ emoji: "👏", count: 1, mine: true }]);
  });

  it("takes a reaction back", () => {
    const before = [
      { emoji: "👍", count: 2, mine: true },
      { emoji: "🔥", count: 1, mine: false },
    ];
    expect(applyReaction(before, "👍", false)).toEqual([
      { emoji: "👍", count: 1, mine: false },
      { emoji: "🔥", count: 1, mine: false },
    ]);
  });

  it("never leaves the reader with more than one reaction", () => {
    let reactions = [{ emoji: "❤️", count: 4, mine: false }];
    for (const emoji of ["👍", "🔥", "❤️", "🦄", "👍"]) {
      reactions = applyReaction(reactions, emoji, true);
      expect(reactions.filter((r) => r.mine)).toHaveLength(1);
    }
    expect(reactions).toEqual([
      { emoji: "❤️", count: 4, mine: false },
      { emoji: "👍", count: 1, mine: true },
    ]);
  });
});

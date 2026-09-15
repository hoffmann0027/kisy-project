import { describe, expect, it } from "vitest";
import { capabilitiesOf } from "./useCapabilities";

// One place decides what an account may see, because the alternative is what
// this codebase already lived through: the same rule written separately in the
// drawer, the hub and the tab bar, drifting apart one forgotten edit at a time.
// Here the cost of drift is higher than a duplicated menu item — it is a button
// that leads to a 403.

const invited = (roleLevel: number) => ({ accountKind: "invited", roleLevel });
const basic = { accountKind: "basic", roleLevel: null };

describe("an account that came through an invitation", () => {
  it("keeps everything the hierarchy grants", () => {
    const ceo = capabilitiesOf(invited(1));
    expect(ceo).toMatchObject({
      isInvited: true,
      canSeeRating: true,
      canSeeLevels: true,
      canVoteLevels: true,
      canSeeConditions: true,
      canAdmin: true,
      feedPlacement: "hub",
    });
  });

  it("is not an administrator unless it is the CEO", () => {
    expect(capabilitiesOf(invited(2)).canAdmin).toBe(false);
    expect(capabilitiesOf(invited(10)).canAdmin).toBe(false);
  });

  it("loses the rating board at the bottom level, but nothing else", () => {
    // Level 10 is "not in a clan" — still in the hierarchy, just not on the
    // board.
    const bottom = capabilitiesOf(invited(10));
    expect(bottom.canSeeRating).toBe(false);
    expect(bottom.isInvited).toBe(true);
    expect(bottom.canSeeConditions).toBe(true);
    expect(bottom.feedPlacement).toBe("hub");
  });
});

describe("an account that registered without an invitation", () => {
  it("gets none of the hierarchy screens", () => {
    expect(capabilitiesOf(basic)).toMatchObject({
      isInvited: false,
      canSeeRating: false,
      canSeeLevels: false,
      canVoteLevels: false,
      canSeeConditions: false,
      canAdmin: false,
    });
  });

  it("gets the feed in the tab bar, where the rating board would be", () => {
    expect(capabilitiesOf(basic).feedPlacement).toBe("tab");
  });
});

describe("edge cases that must fail closed", () => {
  it("grants nothing while nobody is signed in", () => {
    expect(capabilitiesOf(null).isInvited).toBe(false);
    expect(capabilitiesOf(null).canAdmin).toBe(false);
  });

  it("does not trust the kind alone", () => {
    // An invited account always has a level. Something claiming to be invited
    // without one is inconsistent, and the safe reading is the smaller one —
    // otherwise the UI offers screens whose every request comes back 403.
    const inconsistent = capabilitiesOf({ accountKind: "invited", roleLevel: null });
    expect(inconsistent.isInvited).toBe(false);
    expect(inconsistent.canAdmin).toBe(false);
    expect(inconsistent.canSeeRating).toBe(false);
  });

  it("does not let a missing level read as the strongest one", () => {
    // roleLevel is compared with <=, and the hierarchy counts downwards, so a
    // null treated as 0 would outrank the CEO. This is the frontend half of
    // the same trap guarded in internal/access.
    expect(capabilitiesOf({ accountKind: "basic", roleLevel: null }).canSeeRating).toBe(false);
  });
});

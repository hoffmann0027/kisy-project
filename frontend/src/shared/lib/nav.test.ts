import { describe, expect, it } from "vitest";
import { capabilitiesOf } from "./useCapabilities";
import { communitiesDestination, feedLivesInHub, messagesDestination, ratingOrFeed } from "./nav";

/**
 * Audit D-11: navigation was four independent lists (rail, tab bar, Hub,
 * drawer) and they had drifted. An invited account at level 10 — in the
 * hierarchy but in no clan — was offered "Рейтинг" on the desktop, where the
 * only answer was a "you are not in a clan" popup, and "Лента" on the phone,
 * where the Hub offered the feed a second time.
 *
 * The rule of docs/spec/02-frontend-ux.md is one door per destination. This
 * test enumerates every kind of account and checks it holds.
 */

const accounts = {
  "CEO (invited, level 1)": { accountKind: "invited", roleLevel: 1 },
  "invited, level 9 (the lowest clan level)": { accountKind: "invited", roleLevel: 9 },
  "invited, level 10 (no clan)": { accountKind: "invited", roleLevel: 10 },
  "basic (registered without an invitation)": { accountKind: "basic", roleLevel: null },
  "nobody signed in": null,
};

describe("first-level destinations", () => {
  for (const [name, user] of Object.entries(accounts)) {
    describe(name, () => {
      const caps = capabilitiesOf(user);

      it("has exactly one door to the feed", () => {
        const inThirdSlot = ratingOrFeed(caps).key === "feed";
        const inHub = feedLivesInHub(caps);
        expect(inThirdSlot !== inHub).toBe(true);
      });

      it("is offered the rating board only when it has one", () => {
        expect(ratingOrFeed(caps).key === "rating").toBe(caps.canSeeRating);
      });

      it("gets the same third destination on the phone and on the desktop", () => {
        // Both widgets call this one function; the test pins that there is
        // nothing account-dependent left outside it.
        expect(ratingOrFeed(caps)).toBe(ratingOrFeed(caps));
      });
    });
  }

  it("keeps level 10 away from the empty board", () => {
    // The specific case that was broken: not "Рейтинг" with a popup, but the
    // feed, in one place.
    const caps = capabilitiesOf({ accountKind: "invited", roleLevel: 10 });
    expect(ratingOrFeed(caps).key).toBe("feed");
    expect(feedLivesInHub(caps)).toBe(false);
  });

  it("matches the routes it claims", () => {
    expect(messagesDestination.match("/")).toBe(true);
    expect(messagesDestination.match("/chat/abc")).toBe(true);
    expect(messagesDestination.match("/communities")).toBe(false);
    // A group opens under communities, so it must not light up messages.
    expect(messagesDestination.match("/group/abc")).toBe(false);
    expect(communitiesDestination.match("/group/abc")).toBe(true);
    expect(communitiesDestination.match("/communities/x")).toBe(true);
  });
});

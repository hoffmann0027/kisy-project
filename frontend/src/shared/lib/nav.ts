import type { ReactElement } from "react";
import { Icon } from "@shared/ui/icons";
import type { Capabilities } from "@shared/lib/useCapabilities";

/**
 * First-level destinations, in one place.
 *
 * There were four independent lists — the desktop rail, the phone tab bar, the
 * Hub and the drawer — and they had already drifted apart: an invited account
 * at level 10 was offered "Рейтинг" on the desktop (a button whose only answer
 * was a "you are not in a clan" popup) and "Лента" on the phone, where the Hub
 * offered it a second time. Two doors to one destination is exactly what
 * docs/spec/02-frontend-ux.md forbids (audit D-11).
 *
 * The lists stay separate — the rail and the tab bar differ in order and in
 * what surrounds them — but WHICH destinations exist, and where the feed
 * lives, is decided here.
 */

export interface NavDestination {
  key: string;
  /** Title on the desktop rail, aria-label on the phone. */
  label: string;
  icon: (p: { size?: number }) => ReactElement;
  to: string;
  /** Whether a pathname lights this destination up. */
  match: (path: string) => boolean;
}

export const messagesDestination: NavDestination = {
  key: "messages",
  label: "Сообщения",
  icon: Icon.Chat,
  to: "/",
  match: (p) => p === "/" || p.startsWith("/chat/"),
};

export const communitiesDestination: NavDestination = {
  key: "communities",
  label: "Сообщества",
  icon: Icon.Community,
  // Groups live under communities, so a group route highlights this one.
  to: "/communities",
  match: (p) => p.startsWith("/communities") || p.startsWith("/group/"),
};

const ratingDestination: NavDestination = {
  key: "rating",
  label: "Рейтинг",
  icon: Icon.Trophy,
  to: "/rating",
  match: (p) => p.startsWith("/rating"),
};

const feedDestination: NavDestination = {
  key: "feed",
  label: "Лента",
  icon: Icon.Board,
  to: "/feed",
  match: (p) => p.startsWith("/feed"),
};

/**
 * The third slot: the rating board for an account that has one, the feed for
 * an account that does not.
 *
 * The test is canSeeRating, not isInvited: an invited account at level 10 is
 * in the hierarchy but in no clan, so the board is empty for it and the server
 * says so. That difference is what the rail and the tab bar disagreed about.
 */
export function ratingOrFeed(caps: Capabilities): NavDestination {
  return caps.canSeeRating ? ratingDestination : feedDestination;
}

/**
 * Whether the Hub shows a "Лента" card. It must not, when the feed already
 * occupies the third slot — that was the duplicate on the phone.
 */
export function feedLivesInHub(caps: Capabilities): boolean {
  return ratingOrFeed(caps).key !== "feed";
}

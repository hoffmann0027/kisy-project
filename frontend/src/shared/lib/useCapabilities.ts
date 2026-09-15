import { useAuthStore } from "@shared/store/auth";

// What this account may see and do — in one place.
//
// KISY now has two kinds of account. An invited one redeemed a CEO invitation
// and holds a clearance level; a basic one registered openly and holds none,
// standing outside the role hierarchy rather than at the bottom of it.
// Everything built on that hierarchy — the rating board, promotion conditions,
// the company vote, administration — does not apply to a basic account.
//
// Scattering `if (user.roleLevel === 1)` and `if (user.roleLevel <= 9)` across
// components is how the drawer, the hub and the tab bar drifted apart before
// (docs/spec/02-frontend-ux.md). The same applies to these rules, with a
// sharper edge: a missed check is not a duplicated menu item, it is an offer
// the server will refuse. The backend enforces all of this independently —
// hiding a button is presentation, never protection.

/** The rating board is the clan board: clearance 1–9. Level 10 is "no clan". */
const RATING_MAX_LEVEL = 9;

export interface Capabilities {
  /** The account came through an invitation and has a place in the hierarchy. */
  isInvited: boolean;
  canSeeRating: boolean;
  /** Whether clearance levels are shown at all (badges, level labels). */
  canSeeLevels: boolean;
  canVoteLevels: boolean;
  canSeeConditions: boolean;
  canAdmin: boolean;
  /**
   * Where the feed lives. A basic account has no rating board, so the feed
   * takes that place in the tab bar; an invited one keeps the rating there and
   * reaches the feed from the hub.
   */
  feedPlacement: "tab" | "hub";
}

const anonymous: Capabilities = {
  isInvited: false,
  canSeeRating: false,
  canSeeLevels: false,
  canVoteLevels: false,
  canSeeConditions: false,
  canAdmin: false,
  feedPlacement: "tab",
};

export function capabilitiesOf(user: { accountKind?: string; roleLevel?: number | null } | null): Capabilities {
  if (!user) return anonymous;

  const level = user.roleLevel ?? null;
  // Both halves are checked, not just the kind: an invited account always has
  // a level, and treating one without a level as invited would show it screens
  // whose every request comes back 403.
  const isInvited = user.accountKind === "invited" && level !== null;

  return {
    isInvited,
    canSeeRating: isInvited && level !== null && level <= RATING_MAX_LEVEL,
    canSeeLevels: isInvited,
    canVoteLevels: isInvited,
    canSeeConditions: isInvited,
    canAdmin: isInvited && level === 1,
    feedPlacement: isInvited ? "hub" : "tab",
  };
}

/** Capabilities of the signed-in account. */
export function useCapabilities(): Capabilities {
  const user = useAuthStore((s) => s.user);
  return capabilitiesOf(user);
}

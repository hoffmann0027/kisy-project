import { create } from "zustand";
import { authApi, usersApi } from "@shared/api/endpoints";
import { isAuthFailure } from "@shared/api/envelope";
import { forgetNativePushDevice } from "@shared/lib/nativePush";
import type { User } from "@shared/api/types";

type Status = "loading" | "authenticated" | "anonymous" | "offline";

// How long to keep trying to reach the server before admitting there is no
// connection. A phone woken by a call has its radio attaching while the app is
// already running, and the first request or two simply do not land.
const RETRY_DELAYS_MS = [400, 800, 1600, 3200];

const delay = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

// The page's memory belongs to one account (audit A-11). The E2EE session and
// its in-flight start, the MLS chat states, the react-query cache with its
// chat list and decrypted messages, the drafts and the presence stores are all
// module state with no owner of their own. Signing out used to clear only
// `user`, so whoever signed in next in the same tab inherited the previous
// account's session — and its E2EE identity, which the server then refused,
// which is exactly the failure that used to fall back to plaintext.
//
// Rather than chase every cache (and the next one someone adds), the page is
// reloaded whenever it stops belonging to the account it was loaded for.
// What survives a reload is on purpose: the per-account encrypted keystore in
// IndexedDB, without which that account's history could never be read again.
let pageOwner: string | null = null;
let reloadPage = () => window.location.reload();

/** Test-only: observe reloads instead of performing them. */
export function setPageReloaderForTests(fn: () => void, owner: string | null = null): void {
  reloadPage = fn;
  pageOwner = owner;
}

/** Claims the page for a signed-in account; false when it must reload first. */
function claimPage(userId: string): boolean {
  if (pageOwner !== null && pageOwner !== userId) {
    reloadPage();
    return false;
  }
  pageOwner = userId;
  return true;
}

interface AuthState {
  user: User | null;
  status: Status;
  /** Fetches the current session on app start (cookie-based). */
  bootstrap: () => Promise<void>;
  login: (username: string, password: string) => Promise<void>;
  register: (inviteToken: string, username: string, displayName: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  status: "loading",

  /**
   * Restores the session on app start.
   *
   * Only the server can end a session. This used to treat every failure as
   * "not signed in", so a phone woken by a call — radio still attaching, first
   * requests going nowhere — was shown the password screen although its
   * session was untouched: the tokens were still in storage, and signing in by
   * hand a moment later worked, because by then the network was up.
   */
  bootstrap: async () => {
    set({ status: "loading" });
    for (let attempt = 0; ; attempt++) {
      try {
        const { user } = await usersApi.me();
        if (!claimPage(user.id)) return;
        set({ user, status: "authenticated" });
        return;
      } catch (err) {
        if (isAuthFailure(err)) {
          set({ user: null, status: "anonymous" });
          return;
        }
        if (attempt >= RETRY_DELAYS_MS.length) {
          // Out of patience, but still not a sign-out: the app says it cannot
          // reach the server and offers to try again.
          set({ user: null, status: "offline" });
          return;
        }
        await delay(RETRY_DELAYS_MS[attempt]);
      }
    }
  },

  login: async (username, password) => {
    const { user } = await authApi.login(username, password);
    // Another account used this page: its memory must go before this one
    // sees anything. The new session's cookies/tokens are already stored, so
    // the reload comes back signed in.
    if (!claimPage(user.id)) return;
    set({ user, status: "authenticated" });
  },

  register: async (inviteToken, username, displayName, password) => {
    const { user } = await authApi.register(inviteToken, username, displayName, password);
    if (!claimPage(user.id)) return;
    set({ user, status: "authenticated" });
  },

  logout: async () => {
    try {
      // A phone must stop showing notifications for an account that is no
      // longer signed in on it. Done first, while the session still authorises
      // the request; the permission itself is kept, so signing back in
      // re-registers without another prompt.
      await forgetNativePushDevice();
      await authApi.logout();
    } finally {
      set({ user: null, status: "anonymous" });
      // Nothing of the signed-out account may stay in memory.
      if (pageOwner !== null) reloadPage();
    }
  },

  setUser: (user) => set({ user }),
}));

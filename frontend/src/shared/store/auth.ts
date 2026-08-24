import { create } from "zustand";
import { authApi, usersApi } from "@shared/api/endpoints";
import { forgetNativePushDevice } from "@shared/lib/nativePush";
import type { User } from "@shared/api/types";

type Status = "loading" | "authenticated" | "anonymous";

interface AuthState {
  user: User | null;
  status: Status;
  /** Fetches the current session on app start (cookie-based). */
  bootstrap: () => Promise<void>;
  login: (username: string, password: string) => Promise<void>;
  register: (inviteToken: string, username: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
  setUser: (user: User) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  status: "loading",

  bootstrap: async () => {
    try {
      const { user } = await usersApi.me();
      set({ user, status: "authenticated" });
    } catch {
      set({ user: null, status: "anonymous" });
    }
  },

  login: async (username, password) => {
    const { user } = await authApi.login(username, password);
    set({ user, status: "authenticated" });
  },

  register: async (inviteToken, username, password) => {
    const { user } = await authApi.register(inviteToken, username, password);
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
    }
  },

  setUser: (user) => set({ user }),
}));

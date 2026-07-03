import { create } from "zustand";
import { authApi, usersApi } from "@shared/api/endpoints";
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
      await authApi.logout();
    } finally {
      set({ user: null, status: "anonymous" });
    }
  },

  setUser: (user) => set({ user }),
}));

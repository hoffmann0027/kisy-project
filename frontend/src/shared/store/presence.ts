import { create } from "zustand";

interface PresenceState {
  online: Set<string>;
  setOnline: (userId: string, isOnline: boolean) => void;
}

// Tracks which user IDs are currently online, fed by WebSocket
// user.online / user.offline events.
export const usePresenceStore = create<PresenceState>((set) => ({
  online: new Set<string>(),
  setOnline: (userId, isOnline) =>
    set((s) => {
      const next = new Set(s.online);
      if (isOnline) next.add(userId);
      else next.delete(userId);
      return { online: next };
    }),
}));

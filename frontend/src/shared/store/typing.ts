import { create } from "zustand";

interface TypingState {
  // chatId -> set of userIds currently typing
  byChat: Record<string, Set<string>>;
  start: (chatId: string, userId: string) => void;
  stop: (chatId: string, userId: string) => void;
}

// Auto-expiry so a dropped typing.stop does not leave a stuck indicator.
const timers = new Map<string, number>();

export const useTypingStore = create<TypingState>((set, get) => ({
  byChat: {},
  start: (chatId, userId) => {
    const key = `${chatId}:${userId}`;
    window.clearTimeout(timers.get(key));
    timers.set(key, window.setTimeout(() => get().stop(chatId, userId), 4000));
    set((s) => {
      const next = new Set(s.byChat[chatId] ?? []);
      next.add(userId);
      return { byChat: { ...s.byChat, [chatId]: next } };
    });
  },
  stop: (chatId, userId) => {
    set((s) => {
      const cur = s.byChat[chatId];
      if (!cur) return s;
      const next = new Set(cur);
      next.delete(userId);
      return { byChat: { ...s.byChat, [chatId]: next } };
    });
  },
}));

import { create } from "zustand";

interface DraftState {
  // Unsent message text per chat id, so switching chats never loses a draft.
  drafts: Record<string, string>;
  setDraft: (chatId: string, text: string) => void;
  clearDraft: (chatId: string) => void;
}

export const useDraftStore = create<DraftState>((set) => ({
  drafts: {},
  setDraft: (chatId, text) =>
    set((s) => {
      if (!text) {
        if (!(chatId in s.drafts)) return s;
        const next = { ...s.drafts };
        delete next[chatId];
        return { drafts: next };
      }
      return { drafts: { ...s.drafts, [chatId]: text } };
    }),
  clearDraft: (chatId) =>
    set((s) => {
      if (!(chatId in s.drafts)) return s;
      const next = { ...s.drafts };
      delete next[chatId];
      return { drafts: next };
    }),
}));

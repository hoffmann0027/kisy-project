import { create } from "zustand";

interface ReadReceiptState {
  // ISO timestamp of the counterpart's last-read position, per private chat.
  // A message of ours is "read" when its createdAt <= this value.
  otherReadAt: Record<string, string>;
  // Seed from the chat listing (Chat.otherLastReadAt) without regressing a
  // fresher live value.
  seed: (chatId: string, iso: string | null) => void;
  // Advance the counterpart's read position from a live message.read event.
  advance: (chatId: string, iso: string) => void;
}

// Tracks how far each chat partner has read, fed by the initial chat DTO and
// by WebSocket message.read events, so the composer can render sent/read ticks.
export const useReadReceiptStore = create<ReadReceiptState>((set) => ({
  otherReadAt: {},
  seed: (chatId, iso) =>
    set((s) => {
      if (!iso) return s;
      const cur = s.otherReadAt[chatId];
      if (cur && cur >= iso) return s;
      return { otherReadAt: { ...s.otherReadAt, [chatId]: iso } };
    }),
  advance: (chatId, iso) =>
    set((s) => {
      const cur = s.otherReadAt[chatId];
      if (cur && cur >= iso) return s;
      return { otherReadAt: { ...s.otherReadAt, [chatId]: iso } };
    }),
}));

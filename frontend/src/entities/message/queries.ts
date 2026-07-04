import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { messagesApi } from "@shared/api/endpoints";
import type { ChatType, Message, MessagePage } from "@shared/api/types";

export const messageKeys = {
  list: (chatType: ChatType, chatId: string) => ["messages", chatType, chatId] as const,
};

// useMessages loads a chat's history newest-first, paging backwards.
export function useMessages(chatType: ChatType, chatId: string | null) {
  return useInfiniteQuery({
    queryKey: chatId ? messageKeys.list(chatType, chatId) : ["messages", "none"],
    enabled: !!chatId,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) => messagesApi.list(chatType, chatId as string, pageParam),
    getNextPageParam: (last: MessagePage) => (last.hasMore ? last.nextCursor : undefined),
  });
}

// flattenMessages merges the pages (each newest-first) into a single
// oldest-first array for rendering.
export function flattenMessages(pages: MessagePage[] | undefined): Message[] {
  if (!pages) return [];
  const all = pages.flatMap((p) => p.items);
  return all.slice().reverse();
}

export function useSendMessage(chatType: ChatType, chatId: string) {
  return useMutation({
    mutationFn: (args: { text: string; replyTo?: string }) =>
      messagesApi.send(chatType, chatId, args.text, args.replyTo),
    // Optimistic insertion is handled by the caller via the cache writer so
    // the pending bubble can be reconciled with the server ack / WS echo.
  });
}

export function useDeleteMessage() {
  return useMutation({ mutationFn: (messageId: string) => messagesApi.remove(messageId) });
}

export function useEditMessage() {
  return useMutation({
    mutationFn: (args: { messageId: string; text: string }) => messagesApi.edit(args.messageId, args.text),
    // The updated message arrives via WebSocket (message.updated) and is
    // patched into the cache there, so we do not double-write here.
  });
}

export function useReaction() {
  return useMutation({
    mutationFn: (args: { messageId: string; emoji: string; remove: boolean }) =>
      args.remove
        ? messagesApi.removeReaction(args.messageId, args.emoji)
        : messagesApi.addReaction(args.messageId, args.emoji),
  });
}

// upsertMessage inserts or replaces a message in the infinite-query cache
// (used by the WebSocket layer for realtime delivery).
export function useMessageCacheWriter() {
  const qc = useQueryClient();

  return {
    insert(msg: Message) {
      qc.setQueryData(messageKeys.list(msg.chatType, msg.chatId), (old: any) => {
        if (!old) return old;
        const pages = old.pages as MessagePage[];
        if (pages.some((p) => p.items.some((m) => m.id === msg.id))) return old;
        const first = { ...pages[0], items: [msg, ...pages[0].items] };
        return { ...old, pages: [first, ...pages.slice(1)] };
      });
    },
    patch(chatType: ChatType, chatId: string, messageId: string, fn: (m: Message) => Message) {
      qc.setQueryData(messageKeys.list(chatType, chatId), (old: any) => {
        if (!old) return old;
        const pages = (old.pages as MessagePage[]).map((p) => ({
          ...p,
          items: p.items.map((m) => (m.id === messageId ? fn(m) : m)),
        }));
        return { ...old, pages };
      });
    },
    // insertPending shows an optimistic bubble immediately; unlike insert it
    // does not bail when the cache is empty, so the first message in a fresh
    // conversation still appears at once.
    insertPending(msg: Message) {
      qc.setQueryData(messageKeys.list(msg.chatType, msg.chatId), (old: any) => {
        if (!old) {
          return { pageParams: [undefined], pages: [{ items: [msg], hasMore: false }] };
        }
        const pages = old.pages as MessagePage[];
        const first = { ...pages[0], items: [msg, ...pages[0].items] };
        return { ...old, pages: [first, ...pages.slice(1)] };
      });
    },
    // resolvePending swaps the optimistic bubble for the server's message,
    // deduping against a WebSocket echo that may already have inserted it.
    resolvePending(chatType: ChatType, chatId: string, tempId: string, real: Message) {
      qc.setQueryData(messageKeys.list(chatType, chatId), (old: any) => {
        if (!old) return old;
        let seenReal = false;
        const pages = (old.pages as MessagePage[]).map((p) => ({
          ...p,
          items: p.items
            .map((m) => (m.id === tempId ? real : m))
            .filter((m) => {
              if (m.id !== real.id) return true;
              if (seenReal) return false; // drop duplicate echo
              seenReal = true;
              return true;
            }),
        }));
        return { ...old, pages };
      });
    },
    remove(chatType: ChatType, chatId: string, messageId: string) {
      qc.setQueryData(messageKeys.list(chatType, chatId), (old: any) => {
        if (!old) return old;
        const pages = (old.pages as MessagePage[]).map((p) => ({
          ...p,
          items: p.items.filter((m) => m.id !== messageId),
        }));
        return { ...old, pages };
      });
    },
  };
}

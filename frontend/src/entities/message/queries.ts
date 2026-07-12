import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { messagesApi, type SendMessageBody } from "@shared/api/endpoints";
import type { ChatType, Message, MessagePage } from "@shared/api/types";
import { cachePlaintext, e2eeSession, encryptForChat, hydrateMessages } from "@entities/e2ee";

export const messageKeys = {
  list: (chatType: ChatType, chatId: string) => ["messages", chatType, chatId] as const,
  pinned: (chatType: ChatType, chatId: string) => ["pinned", chatType, chatId] as const,
  thread: (rootId: string) => ["thread", rootId] as const,
};

// hydratePage swaps ciphertext for locally decrypted text (E2EE chats);
// a missing session leaves messages untouched (plaintext-only mode).
async function hydratePage(page: MessagePage): Promise<MessagePage> {
  const s = e2eeSession();
  if (!s) return page;
  return { ...page, items: await hydrateMessages(s, page.items) };
}

export function usePinnedMessages(chatType: ChatType, chatId: string | null) {
  return useQuery({
    queryKey: chatId ? messageKeys.pinned(chatType, chatId) : ["pinned", "none"],
    enabled: !!chatId,
    queryFn: async () => {
      const { pinned } = await messagesApi.listPinned(chatType, chatId as string);
      const s = e2eeSession();
      return s ? hydrateMessages(s, pinned) : pinned;
    },
  });
}

export function usePinMessage(chatType: ChatType, chatId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { messageId: string; pin: boolean }) =>
      args.pin ? messagesApi.pin(args.messageId) : messagesApi.unpin(args.messageId),
    onSuccess: () => qc.invalidateQueries({ queryKey: messageKeys.pinned(chatType, chatId) }),
  });
}

// useMessages loads a chat's history newest-first, paging backwards.
export function useMessages(chatType: ChatType, chatId: string | null) {
  return useInfiniteQuery({
    queryKey: chatId ? messageKeys.list(chatType, chatId) : ["messages", "none"],
    enabled: !!chatId,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) =>
      messagesApi.list(chatType, chatId as string, pageParam).then(hydratePage),
    getNextPageParam: (last: MessagePage) => (last.hasMore ? last.nextCursor : undefined),
  });
}

// useThreadMessages loads one thread's replies (stage K), newest-first
// pages flattened oldest-first by the caller via flattenMessages.
export function useThreadMessages(rootId: string | null) {
  return useInfiniteQuery({
    queryKey: rootId ? messageKeys.thread(rootId) : ["thread", "none"],
    enabled: !!rootId,
    initialPageParam: undefined as string | undefined,
    queryFn: ({ pageParam }) =>
      messagesApi.listThread(rootId as string, pageParam).then(hydratePage),
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

/**
 * Send a message. In private chats with an active E2EE session the text is
 * encrypted client-side (the server stores only MLS ciphertext); the returned
 * message is re-hydrated with the plaintext so the UI never regresses to a
 * lock placeholder. Falls back to plaintext while the peer has no E2EE
 * devices or the session failed to initialize.
 */
export function useSendMessage(chatType: ChatType, chatId: string, peerUserId?: string) {
  return useMutation({
    mutationFn: async (args: { text: string; replyTo?: string; attachmentIds?: string[]; threadRootId?: string }) => {
      const s = e2eeSession();
      let body: SendMessageBody = {
        text: args.text,
        replyTo: args.replyTo,
        attachmentIds: args.attachmentIds,
        threadRootId: args.threadRootId,
      };
      let sentEncrypted = false;
      if (s && chatType === "private" && peerUserId && args.text) {
        const enc = await encryptForChat(s, chatId, peerUserId, args.text).catch(() => null);
        if (enc) {
          body = { ...enc, replyTo: args.replyTo, attachmentIds: args.attachmentIds, contentKind: 1 };
          sentEncrypted = true;
        }
      }
      const { message } = await messagesApi.send(chatType, chatId, body);
      if (sentEncrypted && s) {
        // Senders cannot decrypt their own MLS messages (keys are consumed
        // at encryption time) — cache the plaintext under the real id now,
        // tagged with the message's disappearing timer (stage J) so it
        // self-evicts even if this device misses the deletion event.
        await cachePlaintext(s, message.id, args.text, message.expiresAt);
        return { message: { ...message, text: args.text, encrypted: true } };
      }
      return { message };
    },
    // Optimistic insertion is handled by the caller via the cache writer so
    // the pending bubble can be reconciled with the server ack / WS echo.
  });
}

export function useDeleteMessage() {
  return useMutation({ mutationFn: (messageId: string) => messagesApi.remove(messageId) });
}

export interface ForwardTargetRef {
  chatType: ChatType;
  chatId: string;
  /** Peer user id for private targets — enables E2EE re-encryption. */
  peerUserId?: string;
}

/**
 * Forward messages into a target chat. Plaintext messages go server-side in
 * one batch (the server enforces the clearance hierarchy and stamps the
 * attribution). Encrypted messages are re-sent client-side: re-encrypted for
 * an E2EE private target, or — when forwarded out to a non-E2EE target — sent
 * as the locally decrypted text (an explicit user choice). Attribution is
 * preserved: an already-forwarded message keeps its original author.
 */
export function useForwardMessages() {
  return useMutation({
    mutationFn: async (args: {
      target: ForwardTargetRef;
      messages: Message[];
      resolveName: (senderId: string) => string;
    }) => {
      const { target, messages, resolveName } = args;
      const s = e2eeSession();

      const plaintextIds: string[] = [];
      const encrypted: Message[] = [];
      for (const m of messages) {
        // An undecryptable message (no local plaintext) cannot be forwarded.
        if (m.undecryptable || (m.encrypted && !m.text)) continue;
        if (m.encrypted) encrypted.push(m);
        else plaintextIds.push(m.id);
      }
      if (plaintextIds.length === 0 && encrypted.length === 0) {
        throw new Error("Нет сообщений, доступных для пересылки");
      }

      for (const m of encrypted) {
        const senderId = m.forwardedFrom?.senderId ?? m.senderId;
        const senderName = m.forwardedFrom?.senderName ?? resolveName(senderId);
        const text = m.text ?? "";
        let sent = false;
        if (s && target.chatType === "private" && target.peerUserId && text) {
          const enc = await encryptForChat(s, target.chatId, target.peerUserId, text).catch(() => null);
          if (enc) {
            const { message } = await messagesApi.send(target.chatType, target.chatId, {
              ...enc,
              contentKind: 1,
              forwardedFromSenderId: senderId,
              forwardedFromSenderName: senderName,
            });
            await cachePlaintext(s, message.id, text, message.expiresAt);
            sent = true;
          }
        }
        if (!sent) {
          // Non-E2EE target: forwarding necessarily reveals the text there.
          await messagesApi.send(target.chatType, target.chatId, {
            text,
            forwardedFromSenderId: senderId,
            forwardedFromSenderName: senderName,
          });
        }
      }

      if (plaintextIds.length > 0) {
        await messagesApi.forward(plaintextIds, target.chatType, target.chatId);
      }
    },
  });
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
    // insertThread adds a reply to an open thread panel's cache (stage K),
    // creating the cache if the panel was empty and deduping WS echoes.
    insertThread(rootId: string, msg: Message) {
      qc.setQueryData(messageKeys.thread(rootId), (old: any) => {
        if (!old) {
          return { pageParams: [undefined], pages: [{ items: [msg], hasMore: false }] };
        }
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

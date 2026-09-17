import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { messagesApi, type SendMessageBody } from "@shared/api/endpoints";
import type { ChatType, Message, MessagePage } from "@shared/api/types";
import {
  adoptOutgoingPlaintext,
  cacheOutgoingPlaintext,
  cachePlaintext,
  e2eeSession,
  hydrateMessages,
  type EncryptedBody,
} from "@entities/e2ee";
import { encryptPrivateText } from "./encryption";

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
 * Send a message. In a private chat the text is encrypted client-side and the
 * server receives only MLS ciphertext; the returned message is re-hydrated with
 * the plaintext so the UI never regresses to a lock placeholder. If the text
 * cannot be encrypted the message is NOT sent — a UserFacingError says why
 * (fail-closed, audit A-10). Attachment-only messages carry no text to encrypt.
 */
export interface SendArgs {
  text: string;
  replyTo?: string;
  attachmentIds?: string[];
  threadRootId?: string;
}

export async function sendMessage(chatType: ChatType, chatId: string, peerUserId: string | undefined, args: SendArgs) {
  if (chatType !== "private" || !args.text) {
    const body: SendMessageBody = {
      text: args.text,
      replyTo: args.replyTo,
      attachmentIds: args.attachmentIds,
      threadRootId: args.threadRootId,
    };
    return messagesApi.send(chatType, chatId, body);
  }

  const { session: s, body: enc } = await encryptPrivateText(chatId, peerUserId, args.text);
  const body: SendMessageBody = { ...enc, replyTo: args.replyTo, attachmentIds: args.attachmentIds, contentKind: 1 };
  const ciphertext = enc.ciphertext;
  // Before the request, not after: a sender cannot decrypt its own MLS
  // message, so until this write lands the text exists nowhere but in memory.
  // Keyed by the ciphertext digest because the server has not assigned an id.
  await cacheOutgoingPlaintext(s, ciphertext, args.text, null);
  const { message } = await messagesApi.send(chatType, chatId, body);
  // Re-key onto the real id and stamp it with the disappearing timer (stage J)
  // so it self-evicts even if this device misses the deletion event.
  await adoptOutgoingPlaintext(s, ciphertext, message.id, message.expiresAt);
  return { message: { ...message, text: args.text, encrypted: true } };
}

export function useSendMessage(chatType: ChatType, chatId: string, peerUserId?: string) {
  return useMutation({
    mutationFn: (args: SendArgs) => sendMessage(chatType, chatId, peerUserId, args),
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
 * one batch (the server enforces the clearance hierarchy, stamps the
 * attribution and copies attachments) — into a private target with their text
 * encrypted by this client, since a private chat stores only ciphertext.
 * Encrypted messages are re-sent client-side: re-encrypted for a private
 * target, or — forwarded out to a group — sent as the locally decrypted text
 * (an explicit user choice). If any text cannot be encrypted, nothing is
 * forwarded (fail-closed, audit A-10). Attribution is preserved: an
 * already-forwarded message keeps its original author.
 */
export interface ForwardArgs {
  target: ForwardTargetRef;
  messages: Message[];
  resolveName: (senderId: string) => string;
}

export async function forwardMessages(args: ForwardArgs): Promise<void> {
  const { target, messages, resolveName } = args;
  const intoPrivate = target.chatType === "private";

  const plaintext: Message[] = [];
  const encrypted: Message[] = [];
  for (const m of messages) {
    // An undecryptable message (no local plaintext) cannot be forwarded.
    if (m.undecryptable || (m.encrypted && !m.text)) continue;
    if (m.encrypted) encrypted.push(m);
    else plaintext.push(m);
  }
  if (plaintext.length === 0 && encrypted.length === 0) {
    throw new Error("Нет сообщений, доступных для пересылки");
  }

  // Into a private chat every text is encrypted FIRST, all of it, before
  // anything is sent: one failure forwards nothing, and nothing is ever sent
  // in the clear (fail-closed, audit A-10).
  const serverEncrypted: Record<string, EncryptedBody> = {};
  const clientSends: { m: Message; body: EncryptedBody | null }[] = [];
  if (intoPrivate) {
    for (const m of plaintext) {
      if (m.text) serverEncrypted[m.id] = (await encryptPrivateText(target.chatId, target.peerUserId, m.text)).body;
    }
  }
  for (const m of encrypted) {
    const body = intoPrivate ? (await encryptPrivateText(target.chatId, target.peerUserId, m.text ?? "")).body : null;
    clientSends.push({ m, body });
  }

  for (const { m, body } of clientSends) {
    const senderId = m.forwardedFrom?.senderId ?? m.senderId;
    const senderName = m.forwardedFrom?.senderName ?? resolveName(senderId);
    if (body) {
      const { message } = await messagesApi.send(target.chatType, target.chatId, {
        ...body,
        contentKind: 1,
        forwardedFromSenderId: senderId,
        forwardedFromSenderName: senderName,
      });
      const s = e2eeSession();
      if (s) await cachePlaintext(s, message.id, m.text ?? "", message.expiresAt);
    } else {
      // Out of an E2EE chat into a group: forwarding necessarily reveals the
      // text there — the user chose the target.
      await messagesApi.send(target.chatType, target.chatId, {
        text: m.text ?? "",
        forwardedFromSenderId: senderId,
        forwardedFromSenderName: senderName,
      });
    }
  }

  if (plaintext.length > 0) {
    await messagesApi.forward(
      plaintext.map((m) => m.id),
      target.chatType,
      target.chatId,
      intoPrivate ? serverEncrypted : undefined,
    );
  }
}

export function useForwardMessages() {
  return useMutation({ mutationFn: forwardMessages });
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

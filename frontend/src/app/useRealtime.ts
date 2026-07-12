import { useEffect, useRef } from "react";
import { useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { wsClient } from "@shared/ws/client";
import type { ServerEvent } from "@shared/ws/events";
import type { Chat, ChatType, Message, User } from "@shared/api/types";
import { chatKeys } from "@entities/chat/queries";
import { groupKeys } from "@entities/group/queries";
import { notificationKeys } from "@entities/notification/queries";
import { messageKeys } from "@entities/message/queries";
import { scheduledKey } from "@entities/message/scheduled";
import { usePresenceStore } from "@shared/store/presence";
import { useTypingStore } from "@shared/store/typing";
import { useReadReceiptStore } from "@shared/store/readReceipts";
import { useAuthStore } from "@shared/store/auth";
import {
  dropPlaintext,
  e2eeSession,
  hydrateMessage,
  initE2EE,
  processChatHandshake,
  processWelcomes,
  sweepExpiredPlaintext,
} from "@entities/e2ee";

// useRealtime connects the WebSocket for the session and routes server events
// into the query cache and the presence/typing stores. Mounted once by the
// authenticated app layout so it stays connected across every page.
export function useRealtime() {
  const qc = useQueryClient();
  const meId = useAuthStore((s) => s.user?.id);

  // Which chat/group is open right now — an incoming message for it must not
  // raise an unread badge (the user is already looking at it).
  const params = useParams();
  const activeChatId = params.chatId ?? params.groupId ?? null;
  const activeChatRef = useRef<string | null>(activeChatId);
  activeChatRef.current = activeChatId;

  // Clear the unread badge of the chat the user just opened.
  useEffect(() => {
    if (!activeChatId) return;
    qc.setQueryData<Chat[]>(chatKeys.list, (prev) =>
      prev?.map((c) => (c.id === activeChatId ? { ...c, unreadCount: 0 } : c)),
    );
  }, [activeChatId, qc]);

  useEffect(() => {
    if (!meId) return;

    // Bootstrap E2EE for this session (device identity, key packages) and
    // join any chats other devices/users invited us into while offline.
    void initE2EE(meId).then((s) => {
      if (s) return processWelcomes(s).catch(() => {});
    });

    // Disappearing messages (stage J): evict any cached E2EE plaintext whose
    // timer elapsed while this device was offline (the WS message.deleted
    // purge only reaches connected clients). Run now and hourly.
    const sweep = () => {
      const s = e2eeSession();
      if (s) void sweepExpiredPlaintext(s).catch(() => {});
    };
    sweep();
    const sweepTimer = window.setInterval(sweep, 60 * 60 * 1000);

    wsClient.connect();

    // Subscribe to presence of every known chat partner. Runs on every
    // (re)connect so subscriptions survive socket drops, and refetches chats
    // + open conversations to backfill anything missed while disconnected.
    const resubscribe = () => {
      const chats = qc.getQueryData<Chat[]>(chatKeys.list) ?? [];
      const partnerIds = chats.map((c) => c.otherUserId);
      if (partnerIds.length > 0) {
        wsClient.send({ type: "presence.subscribe", data: { userIds: partnerIds } });
      }
    };
    const unsubOpen = wsClient.onOpen(() => {
      resubscribe();
      // Backfill: pull fresh history for open conversations and the chat list.
      void qc.invalidateQueries({ queryKey: ["messages"] });
      void qc.invalidateQueries({ queryKey: chatKeys.list });
    });
    resubscribe();

    const unsub = wsClient.subscribe((ev: ServerEvent) => {
      switch (ev.event) {
        case "message.created":
          void handleMessageCreated(qc, ev.data, meId, activeChatRef.current);
          break;
        case "e2ee.welcome": {
          // A Welcome awaits this device — join, then refetch the chat so
          // freshly decryptable messages render.
          const s = e2eeSession();
          if (s) {
            void processWelcomes(s).then((joined) => {
              for (const chatId of joined) {
                qc.invalidateQueries({ queryKey: messageKeys.list("private", chatId) });
              }
            });
          }
          break;
        }
        case "e2ee.handshake": {
          // A commit/proposal advanced the chat's MLS epoch — apply it.
          const s = e2eeSession();
          if (s) {
            void processChatHandshake(s, ev.data.chatType as ChatType, ev.data.chatId);
          }
          break;
        }
        case "message.updated": {
          const upd = ev.data;
          patchMessage(qc, upd.chatType, upd.chatId, upd.id, (m) => ({
            ...m,
            text: upd.text,
            editedAt: upd.editedAt,
            pinnedAt: upd.pinnedAt,
          }));
          // Pin/unpin also changes the chat's pinned bar.
          qc.invalidateQueries({ queryKey: ["pinned", upd.chatType, upd.chatId] });
          break;
        }
        case "message.deleted": {
          // Security-critical (stage J): ANY deletion purges the locally
          // cached E2EE plaintext — a "disappeared" message must not
          // survive in IndexedDB.
          const s = e2eeSession();
          if (s) void dropPlaintext(s, ev.data.messageId).catch(() => {});

          if (ev.data.expired) {
            // Self-destructed: hard-deleted server-side, so the bubble
            // vanishes entirely (no tombstone).
            removeMessage(qc, ev.data.chatType as ChatType, ev.data.chatId, ev.data.messageId);
            qc.invalidateQueries({ queryKey: ["pinned", ev.data.chatType, ev.data.chatId] });
          } else {
            patchMessage(qc, ev.data.chatType as ChatType, ev.data.chatId, ev.data.messageId, (m) => ({
              ...m,
              isDeleted: true,
              text: null,
            }));
          }
          break;
        }
        case "message.read":
          // The counterpart advanced their read position; record it so our
          // own messages up to that point render as read.
          if (ev.data.userId !== meId) {
            handleMessageRead(qc, ev.data.chatType as ChatType, ev.data.chatId, ev.data.messageId);
          }
          break;
        case "reaction.added":
        case "reaction.removed":
          // Simplest correct approach: refetch the affected chat's messages.
          qc.invalidateQueries({ queryKey: messageKeys.list(ev.data.chatType as ChatType, ev.data.chatId) });
          break;
        case "typing.started":
          if (ev.data.userId !== meId) useTypingStore.getState().start(ev.data.chatId, ev.data.userId);
          break;
        case "typing.stopped":
          useTypingStore.getState().stop(ev.data.chatId, ev.data.userId);
          break;
        case "user.online":
          usePresenceStore.getState().setOnline(ev.data.userId, true);
          break;
        case "user.offline":
          usePresenceStore.getState().setOnline(ev.data.userId, false);
          break;
        case "user.updated":
          handleUserUpdated(qc, ev.data, meId);
          break;
        case "group.changed":
          qc.invalidateQueries({ queryKey: groupKeys.list });
          break;
        case "rating.changed":
          qc.invalidateQueries({ queryKey: ["rating"] });
          break;
        case "poll.changed":
          qc.invalidateQueries({ queryKey: ["polls"] });
          break;
        case "notification.created":
          qc.invalidateQueries({ queryKey: notificationKeys.list });
          break;
        case "board.changed":
          qc.invalidateQueries({ queryKey: ["board", ev.data.groupId] });
          break;
      }
    });

    return () => {
      unsub();
      unsubOpen();
      window.clearInterval(sweepTimer);
      wsClient.disconnect();
    };
  }, [qc, meId]);
}

// handleUserUpdated refreshes a user's cached name/avatar across the app when
// they edit their profile: the auth store (if it's us), every private chat
// where they are the counterpart, and any open group-member lists.
function handleUserUpdated(qc: ReturnType<typeof useQueryClient>, updated: User, meId: string) {
  if (updated.id === meId) {
    useAuthStore.getState().setUser(updated);
  }
  qc.setQueryData<Chat[]>(chatKeys.list, (prev) =>
    prev?.map((c) => (c.otherUserId === updated.id ? { ...c, otherUser: updated } : c)),
  );
  qc.invalidateQueries({ queryKey: ["group-members"] });
}

// handleMessageRead advances the counterpart's read position for a chat. The
// read event carries a message id; we resolve its timestamp from the message
// cache so read receipts compare like-for-like, falling back to "now" if the
// message is not loaded locally.
function handleMessageRead(
  qc: ReturnType<typeof useQueryClient>,
  chatType: ChatType,
  chatId: string,
  messageId: string,
) {
  const cached = qc.getQueryData<{ pages: { items: Message[] }[] }>(messageKeys.list(chatType, chatId));
  let iso = new Date().toISOString();
  const found = cached?.pages.flatMap((p) => p.items).find((m) => m.id === messageId);
  if (found) iso = found.createdAt;
  useReadReceiptStore.getState().advance(chatId, iso);
}

async function handleMessageCreated(
  qc: ReturnType<typeof useQueryClient>,
  raw: Message,
  meId: string,
  activeChatId: string | null,
) {
  // Decrypt E2EE messages at the boundary: everything past this point (query
  // cache, UI) sees a normal message with text filled in. Our own echoes
  // resolve from the plaintext cache written by the send path.
  let msg = raw;
  const s = e2eeSession();
  if (s && raw.ciphertext) {
    msg = await hydrateMessage(s, raw);
    if (msg.undecryptable && raw.senderId === meId) {
      // Own echo may beat the REST ack (which writes the cache) — skip the
      // insert; the ack's resolvePending supplies the readable message.
      return;
    }
  }

  // A scheduled message just went out — refresh the sender's scheduled list
  // (its row flipped from pending to sent).
  if (msg.scheduledId && msg.senderId === meId) {
    qc.invalidateQueries({ queryKey: scheduledKey });
  }

  // A thread reply (stage K) never enters the main feed: it lands in the
  // open thread panel's cache, and the root's plaque counters advance.
  if (msg.threadRootId) {
    const rootId = msg.threadRootId;
    qc.setQueryData(messageKeys.thread(rootId), (old: any) => {
      if (!old) return old; // panel not open — nothing cached to update
      const pages = old.pages as { items: Message[] }[];
      if (pages.some((p) => p.items.some((m) => m.id === msg.id))) return old;
      const first = { ...pages[0], items: [msg, ...pages[0].items] };
      return { ...old, pages: [first, ...pages.slice(1)] };
    });
    patchMessage(qc, msg.chatType, msg.chatId, rootId, (root) => ({
      ...root,
      threadReplyCount: (root.threadReplyCount ?? 0) + 1,
      threadLastReplyAt: msg.createdAt,
    }));
    return;
  }

  // Insert into the open conversation's cache if present.
  qc.setQueryData(messageKeys.list(msg.chatType, msg.chatId), (old: any) => {
    if (!old) return old;
    const pages = old.pages as { items: Message[] }[];
    if (pages.some((p) => p.items.some((m) => m.id === msg.id))) return old;
    const first = { ...pages[0], items: [msg, ...pages[0].items] };
    return { ...old, pages: [first, ...pages.slice(1)] };
  });

  // Reorder the chat list for messages from others; only raise the unread
  // badge when the chat is NOT the one currently open on screen.
  if (msg.senderId !== meId) {
    const isActive = msg.chatId === activeChatId;
    qc.setQueryData<Chat[]>(chatKeys.list, (prev) => {
      if (!prev) return prev;
      const idx = prev.findIndex((c) => c.id === msg.chatId);
      if (idx === -1) {
        // A new chat we did not know about yet — refetch the list.
        qc.invalidateQueries({ queryKey: chatKeys.list });
        return prev;
      }
      const updated = { ...prev[idx], unreadCount: isActive ? 0 : prev[idx].unreadCount + 1 };
      const rest = prev.filter((_, i) => i !== idx);
      return [updated, ...rest];
    });
  }
}

// removeMessage drops a message from the caches entirely (expired
// self-destructing messages leave no tombstone) — the main feed and any
// open thread panel (stage K).
function removeMessage(
  qc: ReturnType<typeof useQueryClient>,
  chatType: ChatType,
  chatId: string,
  messageId: string,
) {
  const drop = (old: any) => {
    if (!old) return old;
    const pages = (old.pages as { items: Message[] }[]).map((p) => ({
      ...p,
      items: p.items.filter((m) => m.id !== messageId),
    }));
    return { ...old, pages };
  };
  qc.setQueryData(messageKeys.list(chatType, chatId), drop);
  qc.setQueriesData({ queryKey: ["thread"] }, drop);
}

// patchMessage rewrites one message wherever it is cached: the main feed
// and any open thread panel (stage K).
function patchMessage(
  qc: ReturnType<typeof useQueryClient>,
  chatType: ChatType,
  chatId: string,
  messageId: string,
  fn: (m: Message) => Message,
) {
  const rewrite = (old: any) => {
    if (!old) return old;
    const pages = (old.pages as { items: Message[] }[]).map((p) => ({
      ...p,
      items: p.items.map((m) => (m.id === messageId ? fn(m) : m)),
    }));
    return { ...old, pages };
  };
  qc.setQueryData(messageKeys.list(chatType, chatId), rewrite);
  qc.setQueriesData({ queryKey: ["thread"] }, rewrite);
}

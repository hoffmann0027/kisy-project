import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { wsClient } from "@shared/ws/client";
import type { ServerEvent } from "@shared/ws/events";
import type { Chat, ChatType, Message, User } from "@shared/api/types";
import { chatKeys } from "@entities/chat/queries";
import { groupKeys } from "@entities/group/queries";
import { notificationKeys } from "@entities/notification/queries";
import { messageKeys } from "@entities/message/queries";
import { usePresenceStore } from "@shared/store/presence";
import { useTypingStore } from "@shared/store/typing";
import { useReadReceiptStore } from "@shared/store/readReceipts";
import { useAuthStore } from "@shared/store/auth";

// useRealtime connects the WebSocket for the session and routes server
// events into the query cache and the presence/typing stores. It is
// mounted once by the messenger shell.
export function useRealtime() {
  const qc = useQueryClient();
  const meId = useAuthStore((s) => s.user?.id);

  useEffect(() => {
    if (!meId) return;

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
          handleMessageCreated(qc, ev.data, meId);
          break;
        case "message.updated":
          patchMessage(qc, ev.data.chatType, ev.data.chatId, ev.data.id, (m) => ({
            ...m,
            text: ev.data.text,
            editedAt: ev.data.editedAt,
          }));
          break;
        case "message.deleted":
          patchMessage(qc, ev.data.chatType as ChatType, ev.data.chatId, ev.data.messageId, (m) => ({
            ...m,
            isDeleted: true,
            text: null,
          }));
          break;
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

function handleMessageCreated(qc: ReturnType<typeof useQueryClient>, msg: Message, meId: string) {
  // Insert into the open conversation's cache if present.
  qc.setQueryData(messageKeys.list(msg.chatType, msg.chatId), (old: any) => {
    if (!old) return old;
    const pages = old.pages as { items: Message[] }[];
    if (pages.some((p) => p.items.some((m) => m.id === msg.id))) return old;
    const first = { ...pages[0], items: [msg, ...pages[0].items] };
    return { ...old, pages: [first, ...pages.slice(1)] };
  });

  // Bump unread + reorder the chat list for messages from others.
  if (msg.senderId !== meId) {
    qc.setQueryData<Chat[]>(chatKeys.list, (prev) => {
      if (!prev) return prev;
      const idx = prev.findIndex((c) => c.id === msg.chatId);
      if (idx === -1) {
        // A new chat we did not know about yet — refetch the list.
        qc.invalidateQueries({ queryKey: chatKeys.list });
        return prev;
      }
      const updated = { ...prev[idx], unreadCount: prev[idx].unreadCount + 1 };
      const rest = prev.filter((_, i) => i !== idx);
      return [updated, ...rest];
    });
  }
}

function patchMessage(
  qc: ReturnType<typeof useQueryClient>,
  chatType: ChatType,
  chatId: string,
  messageId: string,
  fn: (m: Message) => Message,
) {
  qc.setQueryData(messageKeys.list(chatType, chatId), (old: any) => {
    if (!old) return old;
    const pages = (old.pages as { items: Message[] }[]).map((p) => ({
      ...p,
      items: p.items.map((m) => (m.id === messageId ? fn(m) : m)),
    }));
    return { ...old, pages };
  });
}

import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { useCallControls } from "@features/call/CallProvider";
import { formatDay } from "@shared/lib/format";
import { Avatar, Button, Spinner, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { Attachment, ChatType, Message } from "@shared/api/types";
import {
  flattenMessages,
  useDeleteMessage,
  useEditMessage,
  useMessageCacheWriter,
  useMessages,
  usePinMessage,
  usePinnedMessages,
  useReaction,
  useSendMessage,
} from "@entities/message/queries";
import { messagesApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { useTypingStore } from "@shared/store/typing";
import { useReadReceiptStore } from "@shared/store/readReceipts";
import { MessageBubble, type DeliveryStatus } from "./MessageBubble";
import { Composer } from "./Composer";

export interface ConversationTarget {
  chatType: ChatType;
  chatId: string;
  title: string;
  avatarName: string;
  avatarUrl?: string | null;
  online?: boolean;
  offlineLabel?: string;
  /** Counterpart's last-read position, seeds read receipts for private chats. */
  otherLastReadAt?: string | null;
  /** Direct-chat peer's user id; enables the audio-call button (private only). */
  peerUserId?: string;
}

interface Props {
  target: ConversationTarget;
  /** Extra controls rendered on the right of the header (e.g. board tab). */
  headerActions?: ReactNode;
}

export function Conversation({ target, headerActions }: Props) {
  const { chatType, chatId } = target;
  const navigate = useNavigate();
  const { startCall, busy: callBusy } = useCallControls();
  const me = useAuthStore((s) => s.user!);
  const typingByChat = useTypingStore((s) => s.byChat);
  const { data, isPending, fetchNextPage, hasNextPage, isFetchingNextPage } = useMessages(chatType, chatId);
  const send = useSendMessage(chatType, chatId);
  const del = useDeleteMessage();
  const edit = useEditMessage();
  const pin = usePinMessage(chatType, chatId);
  const { data: pinned } = usePinnedMessages(chatType, chatId);
  const react = useReaction();
  const cache = useMessageCacheWriter();

  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const messages = useMemo(() => flattenMessages(data?.pages), [data]);
  const typers = Array.from(typingByChat[chatId] ?? []).filter((id) => id !== me.id);

  // Read receipts: seed the counterpart's read position from the chat DTO,
  // then read the (possibly fresher) live value the WebSocket layer maintains.
  const seedRead = useReadReceiptStore((s) => s.seed);
  const otherReadAt = useReadReceiptStore((s) => s.otherReadAt[chatId]);
  useEffect(() => {
    seedRead(chatId, target.otherLastReadAt ?? null);
  }, [chatId, target.otherLastReadAt, seedRead]);

  const statusFor = (m: Message): DeliveryStatus | undefined => {
    if (m.senderId !== me.id) return undefined;
    if (m.pending) return "pending";
    if (m.failed) return "failed";
    if (chatType !== "private") return undefined; // group ticks are out of scope
    return otherReadAt && m.createdAt <= otherReadAt ? "read" : "sent";
  };

  const lastCount = useRef(0);
  useEffect(() => {
    if (messages.length > lastCount.current) {
      bottomRef.current?.scrollIntoView({ behavior: lastCount.current === 0 ? "auto" : "smooth" });
    }
    lastCount.current = messages.length;
  }, [messages.length]);

  useEffect(() => {
    // Mark the newest server-acked message as read (skip optimistic bubbles,
    // whose temp ids are not valid on the server).
    const newest = [...messages].reverse().find((m) => !m.pending);
    if (!newest) return;
    void messagesApi.markRead(chatType, chatId, newest.id).catch(() => {});
  }, [chatType, chatId, messages.length]);

  const handleSend = (text: string, replyToId?: string, attachments?: Attachment[]) => {
    // Optimistic bubble: show the message instantly, reconcile on server ack.
    const tempId = `temp-${crypto.randomUUID()}`;
    const optimistic: Message = {
      id: tempId,
      chatId,
      chatType,
      senderId: me.id,
      text: text || null,
      replyTo: replyToId ?? null,
      attachments: attachments ?? [],
      reactions: [],
      mentions: [],
      isDeleted: false,
      createdAt: new Date().toISOString(),
      deletedAt: null,
      editedAt: null,
      pinnedAt: null,
      readCount: null,
      readTotal: null,
      pending: true,
    };
    cache.insertPending(optimistic);

    send.mutate(
      { text, replyTo: replyToId, attachmentIds: attachments?.map((a) => a.id) },
      {
        onSuccess: ({ message }) => cache.resolvePending(chatType, chatId, tempId, message),
        onError: () => {
          cache.patch(chatType, chatId, tempId, (m) => ({ ...m, pending: false, failed: true }));
          toast.error("Не удалось отправить сообщение");
        },
      },
    );
  };

  const handleDelete = (m: Message) =>
    del.mutate(m.id, { onError: () => toast.error("Не удалось удалить сообщение") });

  const handleEdit = (m: Message, text: string) =>
    edit.mutate({ messageId: m.id, text }, { onError: () => toast.error("Не удалось изменить сообщение") });

  const handlePin = (m: Message, doPin: boolean) =>
    pin.mutate({ messageId: m.id, pin: doPin }, { onError: () => toast.error("Не удалось закрепить сообщение") });

  const handleReact = (m: Message, emoji: string) => {
    const existing = m.reactions.find((r) => r.emoji === emoji);
    react.mutate({ messageId: m.id, emoji, remove: !!existing?.reacted });
  };

  const previewFor = (id: string | null): string | undefined => {
    if (!id) return undefined;
    const parent = messages.find((m) => m.id === id);
    if (!parent) return undefined;
    return parent.isDeleted ? "удалённое сообщение" : (parent.text ?? "").slice(0, 80);
  };

  let lastDay = "";

  return (
    <section className="conv">
      <header className="conv__header">
        <button className="conv__back" title="Назад" onClick={() => navigate("/")}>
          <Icon.Back size={22} />
        </button>
        <Avatar name={target.avatarName} url={target.avatarUrl} presence={target.online ? "online" : undefined} />
        <div className="conv__header-body">
          <div className="conv__title">{target.title}</div>
          <div className={cn("conv__status", target.online && "conv__status--online")}>
            {typers.length > 0
              ? "печатает…"
              : target.online
                ? "в сети"
                : (target.offlineLabel ?? "не в сети")}
          </div>
        </div>
        {chatType === "private" && target.peerUserId && (
          <button
            className="conv__call"
            title="Позвонить"
            disabled={callBusy}
            onClick={() =>
              startCall(
                { id: target.peerUserId!, displayName: target.title, avatarUrl: target.avatarUrl ?? null },
                chatId,
              )
            }
          >
            <Icon.Phone size={20} />
          </button>
        )}
        {headerActions}
      </header>

      {pinned && pinned.length > 0 && (
        <div className="conv__pinned">
          <Icon.Pin size={15} />
          <div className="conv__pinned-list">
            {pinned.map((m) => (
              <div key={m.id} className="conv__pinned-item">
                <span className="conv__pinned-text">{m.text}</span>
                <button
                  className="conv__pinned-unpin"
                  title="Открепить"
                  onClick={() => handlePin(m, false)}
                >
                  ✕
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="conv__scroll">
        {isPending && (
          <div style={{ margin: "auto" }}>
            <Spinner size={28} />
          </div>
        )}
        {hasNextPage && (
          <div className="conv__load-more">
            <Button variant="ghost" loading={isFetchingNextPage} onClick={() => void fetchNextPage()}>
              Загрузить ещё
            </Button>
          </div>
        )}
        {messages.map((m) => {
          const day = formatDay(m.createdAt);
          const showDay = day !== lastDay;
          lastDay = day;
          return (
            <div key={m.id}>
              {showDay && <div className="conv__day">{day}</div>}
              <MessageBubble
                message={m}
                mine={m.senderId === me.id}
                canDelete={m.senderId === me.id || me.roleLevel === 1}
                canEdit={m.senderId === me.id && !m.pending && !m.failed}
                status={statusFor(m)}
                replyPreview={previewFor(m.replyTo)}
                onReply={setReplyTo}
                onEdit={handleEdit}
                onDelete={handleDelete}
                onReact={handleReact}
                onPin={handlePin}
              />
            </div>
          );
        })}
        {typers.length > 0 && (
          <div className="bubble-row bubble-row--in">
            <div className="bubble bubble--in">
              <span className="typing">
                <span />
                <span />
                <span />
              </span>
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <Composer
        chatType={chatType}
        chatId={chatId}
        replyTo={replyTo}
        replyPreview={previewFor(replyTo?.id ?? null)}
        onClearReply={() => setReplyTo(null)}
        onSend={handleSend}
      />
    </section>
  );
}

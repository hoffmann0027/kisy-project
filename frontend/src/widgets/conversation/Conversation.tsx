import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { useCallControls } from "@features/call/CallProvider";
import { formatDay } from "@shared/lib/format";
import { useVisualViewport } from "@shared/lib/useVisualViewport";
import { Avatar, Button, Spinner, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { MediaViewer, type MediaViewerItem } from "@shared/ui/MediaViewer";
import { ChatPanel } from "./ChatPanel";
import { ForwardModal, type ForwardTarget } from "@features/forward/ForwardModal";
import { MuteMenu } from "@features/notif-prefs/MuteMenu";
import { DisappearMenu } from "@features/disappear/DisappearMenu";
import { useSetMessageExpiry } from "@entities/chat/disappearing";
import type { Attachment, ChatMediaItem, ChatType, Message } from "@shared/api/types";
import {
  flattenMessages,
  useDeleteMessage,
  useEditMessage,
  useForwardMessages,
  useMessageCacheWriter,
  useMessages,
  usePinMessage,
  usePinnedMessages,
  useReaction,
  useSendMessage,
} from "@entities/message/queries";
import { pendingForChat, useScheduledMessages, useScheduleMessage } from "@entities/message/scheduled";
import { ScheduledPanel } from "@features/scheduled/ScheduledPanel";
import { ThreadPanel } from "@features/threads/ThreadPanel";
import { e2eeSession } from "@entities/e2ee";
import { messagesApi } from "@shared/api/endpoints";
import { ApiError } from "@shared/api/envelope";
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
  /** When set, the composer is replaced by this read-only notice (e.g. an
   *  editors-only group where the viewer is a plain member). */
  readOnly?: string;
}

export function Conversation({ target, headerActions, readOnly }: Props) {
  const { chatType, chatId } = target;
  const navigate = useNavigate();
  const { startCall, busy: callBusy } = useCallControls();
  const me = useAuthStore((s) => s.user!);
  const typingByChat = useTypingStore((s) => s.byChat);
  const { data, isPending, fetchNextPage, hasNextPage, isFetchingNextPage } = useMessages(chatType, chatId);
  const send = useSendMessage(chatType, chatId, target.peerUserId);
  const del = useDeleteMessage();
  const edit = useEditMessage();
  const forward = useForwardMessages();
  const pin = usePinMessage(chatType, chatId);
  const { data: pinned } = usePinnedMessages(chatType, chatId);
  const react = useReaction();
  const cache = useMessageCacheWriter();
  const schedule = useScheduleMessage(chatType, chatId, target.peerUserId);
  const setExpiry = useSetMessageExpiry();
  const { scheduled } = useScheduledMessages();
  const pendingScheduled = useMemo(() => pendingForChat(scheduled, chatType, chatId), [scheduled, chatType, chatId]);
  const [scheduledOpen, setScheduledOpen] = useState(false);
  // Threads (stage K, groups): the root whose discussion panel is open.
  const [threadRoot, setThreadRoot] = useState<Message | null>(null);

  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const [panelOpen, setPanelOpen] = useState(false);
  const [viewer, setViewer] = useState<{ items: MediaViewerItem[]; index: number } | null>(null);
  // Reply jump (stage F): the message briefly highlighted after a jump.
  const [highlightId, setHighlightId] = useState<string | null>(null);
  // Multi-select + forwarding (stage D).
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [forwardOpen, setForwardOpen] = useState(false);
  const [forwardSource, setForwardSource] = useState<Message[] | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const selectionMode = selected.size > 0;

  const messages = useMemo(() => flattenMessages(data?.pages), [data]);
  // The open thread's root, kept live: WS counter patches land in the main
  // list cache, so prefer that copy over the snapshot taken on open.
  const liveThreadRoot = useMemo(
    () => (threadRoot ? (messages.find((m) => m.id === threadRoot.id) ?? threadRoot) : null),
    [threadRoot, messages],
  );
  useEffect(() => setThreadRoot(null), [chatId]);
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

  // Opening the keyboard shrinks .conv by --kb-inset (see messenger.css), which
  // would slide the newest message up out of view. Re-pin to the bottom — but
  // only when that is where the reader already was, so someone scrolled back
  // through history is not yanked forward just for tapping the composer.
  const keyboardInset = useVisualViewport();
  const atBottomRef = useRef(true);
  const trackAtBottom = () => {
    const el = scrollRef.current;
    if (el) atBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 120;
  };
  useEffect(() => {
    if (!atBottomRef.current) return;
    bottomRef.current?.scrollIntoView({ behavior: "auto" });
  }, [keyboardInset]);

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

  const handleSchedule = (text: string, sendAt: Date, replyToId?: string, attachments?: Attachment[]) => {
    schedule.mutate(
      { text, sendAt, replyTo: replyToId, attachmentIds: attachments?.map((a) => a.id) },
      {
        onSuccess: () =>
          toast.success(
            `Сообщение будет отправлено ${sendAt.toLocaleString("ru-RU", { day: "numeric", month: "long", hour: "2-digit", minute: "2-digit" })}`,
          ),
        onError: () => toast.error("Не удалось запланировать сообщение"),
      },
    );
  };

  const handleDelete = (m: Message) =>
    del.mutate(m.id, { onError: () => toast.error("Не удалось удалить сообщение") });

  const handleEdit = (m: Message, text: string) =>
    edit.mutate({ messageId: m.id, text }, { onError: () => toast.error("Не удалось изменить сообщение") });

  const handlePin = (m: Message, doPin: boolean) =>
    pin.mutate({ messageId: m.id, pin: doPin }, { onError: () => toast.error("Не удалось закрепить сообщение") });

  const handleSetExpiry = (m: Message, ttlSeconds: number | null) =>
    setExpiry.mutate(
      { messageId: m.id, ttlSeconds },
      {
        onSuccess: () => toast.success(ttlSeconds ? "Таймер установлен" : "Таймер убран"),
        onError: () => toast.error("Не удалось изменить таймер"),
      },
    );

  const handleReact = (m: Message, emoji: string) => {
    const existing = m.reactions.find((r) => r.emoji === emoji);
    react.mutate({ messageId: m.id, emoji, remove: !!existing?.reacted });
  };

  const toggleSelect = (m: Message) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(m.id)) next.delete(m.id);
      else next.add(m.id);
      return next;
    });

  // Resolve the original author's display name for forward attribution: in a
  // private chat the sender is either us or the peer; groups fall back to the
  // server-side attribution for plaintext messages.
  const resolveSenderName = (senderId: string): string => {
    if (senderId === me.id) return me.displayName;
    if (chatType === "private") return target.title;
    return "";
  };

  const openForward = (msgs: Message[]) => {
    if (msgs.length === 0) return;
    setForwardSource(msgs);
    setForwardOpen(true);
  };

  const doForward = (t: ForwardTarget) => {
    const msgs = forwardSource ?? [];
    setForwardOpen(false);
    forward.mutate(
      { target: t, messages: msgs, resolveName: resolveSenderName },
      {
        onSuccess: () => {
          toast.success(`Переслано в «${t.title}»`);
          setSelected(new Set());
          setForwardSource(null);
        },
        onError: (e) => {
          toast.error(
            e instanceof ApiError && e.status === 403
              ? "Нельзя переслать в чат с более широкой аудиторией"
              : "Не удалось переслать сообщения",
          );
        },
      },
    );
  };

  const forwardSelected = () => openForward(messages.filter((m) => selected.has(m.id)));
  const copySelected = () => {
    const text = messages
      .filter((m) => selected.has(m.id) && m.text)
      .map((m) => m.text)
      .join("\n\n");
    if (text) void navigator.clipboard?.writeText(text).catch(() => {});
    setSelected(new Set());
  };
  const deleteSelected = () => {
    messages
      .filter((m) => selected.has(m.id) && (m.senderId === me.id || me.roleLevel === 1))
      .forEach((m) => del.mutate(m.id));
    setSelected(new Set());
  };

  // Open the viewer over every image currently loaded in the conversation,
  // oldest first, positioned at the clicked one. The context panel supplies
  // its own (server-aggregated) item list instead.
  const openImageFromBubble = (a: Attachment) => {
    const images: MediaViewerItem[] = messages
      .flatMap((m) => m.attachments)
      .filter((x) => x.isImage)
      .map((x) => ({ id: x.id, url: x.url, fileName: x.fileName }));
    const index = Math.max(
      0,
      images.findIndex((x) => x.id === a.id),
    );
    setViewer({ items: images.length > 0 ? images : [{ id: a.id, url: a.url, fileName: a.fileName }], index });
  };

  const openMediaFromPanel = (items: ChatMediaItem[], index: number) => {
    setViewer({
      items: items.map((it) => ({
        id: it.attachment.id,
        url: it.attachment.url,
        fileName: it.attachment.fileName,
      })),
      index,
    });
  };

  // Jump to a replied-to message: scroll it into view and flash a highlight.
  // If it is not loaded yet (older than the current window), page backwards a
  // few times until it appears, then retry.
  const jumpToMessage = (id: string | null) => {
    if (!id) return;
    let attempts = 0;
    const tryScroll = () => {
      const el = document.getElementById(`msg-${id}`);
      if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "center" });
        setHighlightId(id);
        window.setTimeout(() => setHighlightId((cur) => (cur === id ? null : cur)), 1600);
        return;
      }
      if (attempts < 8 && hasNextPage) {
        attempts++;
        void fetchNextPage().then(() => window.setTimeout(tryScroll, 120));
      } else {
        toast.info("Исходное сообщение недоступно");
      }
    };
    tryScroll();
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
        <button
          className="conv__back"
          title="Назад"
          onClick={() => navigate(target.chatType === "group" ? "/communities" : "/")}
        >
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
        <DisappearMenu chatType={chatType} chatId={chatId} />
        <MuteMenu chatType={chatType} chatId={chatId} />
        <button
          className={cn("conv__panel-toggle", panelOpen && "conv__panel-toggle--active")}
          title="Медиа, файлы и ссылки"
          onClick={() => setPanelOpen((v) => !v)}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <rect x="3" y="3" width="18" height="18" rx="2" />
            <circle cx="8.5" cy="8.5" r="1.5" />
            <path d="m21 15-5-5L5 21" />
          </svg>
        </button>
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

      <div className="conv__scroll" ref={scrollRef} onScroll={trackAtBottom}>
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
            <div key={m.id} id={`msg-${m.id}`} className={cn(highlightId === m.id && "msg-row--highlight")}>
              {showDay && <div className="conv__day">{day}</div>}
              <MessageBubble
                message={m}
                mine={m.senderId === me.id}
                canDelete={m.senderId === me.id || me.roleLevel === 1}
                canEdit={m.senderId === me.id && !m.pending && !m.failed && !m.encrypted}
                status={statusFor(m)}
                replyPreview={previewFor(m.replyTo)}
                replyTargetId={m.replyTo}
                onJumpToReply={jumpToMessage}
                onReply={setReplyTo}
                onEdit={handleEdit}
                onDelete={handleDelete}
                onReact={handleReact}
                onPin={handlePin}
                onOpenImage={openImageFromBubble}
                onForward={(msg) => openForward([msg])}
                selectionMode={selectionMode}
                selected={selected.has(m.id)}
                onToggleSelect={toggleSelect}
                onSetExpiry={handleSetExpiry}
                onOpenThread={chatType === "group" ? setThreadRoot : undefined}
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

      {selectionMode ? (
        <div className="select-bar">
          <button className="select-bar__cancel" onClick={() => setSelected(new Set())} title="Отменить выбор">
            ✕
          </button>
          <span className="select-bar__count">Выбрано: {selected.size}</span>
          <button className="select-bar__action" onClick={copySelected} title="Копировать">
            <Icon.Copy size={18} />
          </button>
          <button className="select-bar__action" onClick={forwardSelected} title="Переслать">
            <Icon.Forward size={18} />
          </button>
          <button className="select-bar__action select-bar__action--danger" onClick={deleteSelected} title="Удалить">
            <Icon.Trash size={18} />
          </button>
        </div>
      ) : (
        <>
          {pendingScheduled.length > 0 && (
            <button className="conv__scheduled-bar" onClick={() => setScheduledOpen(true)}>
              <Icon.Calendar size={15} />
              Запланированные сообщения: {pendingScheduled.length}
            </button>
          )}
          {readOnly ? (
            <div className="conv__readonly">{readOnly}</div>
          ) : (
            <Composer
              chatType={chatType}
              chatId={chatId}
              replyTo={replyTo}
              replyPreview={previewFor(replyTo?.id ?? null)}
              onClearReply={() => setReplyTo(null)}
              onSend={handleSend}
              onSchedule={handleSchedule}
              scheduleE2EEWarning={chatType === "private" && !!e2eeSession()}
            />
          )}
        </>
      )}

      <ScheduledPanel open={scheduledOpen} items={pendingScheduled} onClose={() => setScheduledOpen(false)} />

      <ForwardModal
        open={forwardOpen}
        count={forwardSource?.length ?? 0}
        onClose={() => setForwardOpen(false)}
        onPick={doForward}
      />

      {panelOpen && (
        <ChatPanel
          chatType={chatType}
          chatId={chatId}
          onClose={() => setPanelOpen(false)}
          onOpenMedia={openMediaFromPanel}
        />
      )}
      {liveThreadRoot && (
        <ThreadPanel
          root={liveThreadRoot}
          onClose={() => setThreadRoot(null)}
          onReact={handleReact}
          onDelete={handleDelete}
          onEdit={handleEdit}
          onOpenImage={openImageFromBubble}
        />
      )}
      {viewer && (
        <MediaViewer
          items={viewer.items}
          index={viewer.index}
          onClose={() => setViewer(null)}
          onIndexChange={(index) => setViewer((v) => (v ? { ...v, index } : v))}
        />
      )}
    </section>
  );
}

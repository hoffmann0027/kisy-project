import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { cn } from "@shared/lib/cn";
import { formatDay } from "@shared/lib/format";
import { Avatar, Button, Spinner, toast } from "@shared/ui";
import type { ChatType, Message } from "@shared/api/types";
import {
  flattenMessages,
  useDeleteMessage,
  useMessageCacheWriter,
  useMessages,
  useReaction,
  useSendMessage,
} from "@entities/message/queries";
import { messagesApi } from "@shared/api/endpoints";
import { useAuthStore } from "@shared/store/auth";
import { useTypingStore } from "@shared/store/typing";
import { MessageBubble } from "./MessageBubble";
import { Composer } from "./Composer";

export interface ConversationTarget {
  chatType: ChatType;
  chatId: string;
  title: string;
  avatarName: string;
  avatarUrl?: string | null;
  online?: boolean;
  offlineLabel?: string;
}

interface Props {
  target: ConversationTarget;
  /** Extra controls rendered on the right of the header (e.g. board tab). */
  headerActions?: ReactNode;
}

export function Conversation({ target, headerActions }: Props) {
  const { chatType, chatId } = target;
  const me = useAuthStore((s) => s.user!);
  const typingByChat = useTypingStore((s) => s.byChat);
  const { data, isPending, fetchNextPage, hasNextPage, isFetchingNextPage } = useMessages(chatType, chatId);
  const send = useSendMessage(chatType, chatId);
  const del = useDeleteMessage();
  const react = useReaction();
  const cache = useMessageCacheWriter();

  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const messages = useMemo(() => flattenMessages(data?.pages), [data]);
  const typers = Array.from(typingByChat[chatId] ?? []).filter((id) => id !== me.id);

  const lastCount = useRef(0);
  useEffect(() => {
    if (messages.length > lastCount.current) {
      bottomRef.current?.scrollIntoView({ behavior: lastCount.current === 0 ? "auto" : "smooth" });
    }
    lastCount.current = messages.length;
  }, [messages.length]);

  useEffect(() => {
    if (messages.length === 0) return;
    const newest = messages[messages.length - 1];
    void messagesApi.markRead(chatType, chatId, newest.id).catch(() => {});
  }, [chatType, chatId, messages.length]);

  const handleSend = (text: string, replyToId?: string) => {
    send.mutate(
      { text, replyTo: replyToId },
      {
        onSuccess: ({ message }) => cache.insert(message),
        onError: () => toast.error("Не удалось отправить сообщение"),
      },
    );
  };

  const handleDelete = (m: Message) =>
    del.mutate(m.id, { onError: () => toast.error("Не удалось удалить сообщение") });

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
        {headerActions}
      </header>

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
                replyPreview={previewFor(m.replyTo)}
                onReply={setReplyTo}
                onDelete={handleDelete}
                onReact={handleReact}
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

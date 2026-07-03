import { useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@shared/lib/cn";
import { formatDay } from "@shared/lib/format";
import { Avatar, Button, Spinner, toast } from "@shared/ui";
import type { Chat, Message } from "@shared/api/types";
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
import { usePresenceStore } from "@shared/store/presence";
import { useTypingStore } from "@shared/store/typing";
import { MessageBubble } from "./MessageBubble";
import { Composer } from "./Composer";

interface Props {
  chat: Chat;
}

export function Conversation({ chat }: Props) {
  const me = useAuthStore((s) => s.user!);
  const online = usePresenceStore((s) => s.online);
  const typingByChat = useTypingStore((s) => s.byChat);
  const { data, isPending, fetchNextPage, hasNextPage, isFetchingNextPage } = useMessages("private", chat.id);
  const send = useSendMessage("private", chat.id);
  const del = useDeleteMessage();
  const react = useReaction();
  const cache = useMessageCacheWriter();

  const [replyTo, setReplyTo] = useState<Message | null>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);

  const messages = useMemo(() => flattenMessages(data?.pages), [data]);
  const other = chat.otherUser;
  const isOnline = other ? online.has(other.id) || other.status === "online" : false;
  const typers = Array.from(typingByChat[chat.id] ?? []).filter((id) => id !== me.id);

  // Auto-scroll to the newest message when the count grows.
  const lastCount = useRef(0);
  useEffect(() => {
    if (messages.length > lastCount.current) {
      bottomRef.current?.scrollIntoView({ behavior: lastCount.current === 0 ? "auto" : "smooth" });
    }
    lastCount.current = messages.length;
  }, [messages.length]);

  // Mark the chat read when messages are shown / arrive.
  useEffect(() => {
    if (messages.length === 0) return;
    const newest = messages[messages.length - 1];
    void messagesApi.markRead("private", chat.id, newest.id).catch(() => {});
  }, [chat.id, messages.length]);

  const handleSend = (text: string, replyToId?: string) => {
    send.mutate(
      { text, replyTo: replyToId },
      {
        onSuccess: ({ message }) => cache.insert(message),
        onError: () => toast.error("Не удалось отправить сообщение"),
      },
    );
  };

  const handleDelete = (m: Message) => {
    del.mutate(m.id, {
      onError: () => toast.error("Не удалось удалить сообщение"),
    });
  };

  const handleReact = (m: Message, emoji: string) => {
    const existing = m.reactions.find((r) => r.emoji === emoji);
    const remove = !!existing?.reacted;
    react.mutate({ messageId: m.id, emoji, remove });
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
        <Avatar name={other?.displayName ?? "?"} url={other?.avatarUrl} presence={isOnline ? "online" : undefined} />
        <div className="conv__header-body">
          <div className="conv__title">{other?.displayName ?? "Пользователь"}</div>
          <div className={cn("conv__status", isOnline && "conv__status--online")}>
            {typers.length > 0 ? "печатает…" : isOnline ? "в сети" : "не в сети"}
          </div>
        </div>
      </header>

      <div className="conv__scroll" ref={scrollRef}>
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
        chatType="private"
        chatId={chat.id}
        replyTo={replyTo}
        replyPreview={previewFor(replyTo?.id ?? null)}
        onClearReply={() => setReplyTo(null)}
        onSend={handleSend}
      />
    </section>
  );
}

import { useMemo, useState } from "react";
import { cn } from "@shared/lib/cn";
import { formatRelative } from "@shared/lib/format";
import { Avatar, Badge, IconButton, Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { Chat } from "@shared/api/types";
import { useChats } from "@entities/chat/queries";
import { usePresenceStore } from "@shared/store/presence";

interface Props {
  activeChatId: string | null;
  onSelect: (chat: Chat) => void;
  onNewChat: () => void;
}

export function ChatList({ activeChatId, onSelect, onNewChat }: Props) {
  const { data: chats, isPending } = useChats();
  const [query, setQuery] = useState("");
  const online = usePresenceStore((s) => s.online);

  const filtered = useMemo(() => {
    const list = chats ?? [];
    if (!query.trim()) return list;
    const q = query.toLowerCase();
    return list.filter((c) => c.otherUser?.displayName.toLowerCase().includes(q) || c.otherUser?.username.toLowerCase().includes(q));
  }, [chats, query]);

  return (
    <aside className="chatlist">
      <div className="chatlist__header">
        <h1 className="chatlist__title">Чаты</h1>
        <IconButton label="Новый чат" onClick={onNewChat}>
          <Icon.Plus />
        </IconButton>
      </div>
      <div className="chatlist__search">
        <div style={{ position: "relative" }}>
          <span style={{ position: "absolute", left: 12, top: 11, color: "var(--color-text-tertiary)" }}>
            <Icon.Search size={18} />
          </span>
          <input
            className="ui-input"
            style={{ paddingLeft: 40 }}
            placeholder="Поиск"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
      </div>

      <div className="chatlist__scroll">
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
            <Spinner />
          </div>
        )}
        {!isPending && filtered.length === 0 && (
          <div style={{ padding: 24, textAlign: "center", color: "var(--color-text-secondary)", fontSize: 14 }}>
            {query ? "Ничего не найдено" : "Пока нет чатов. Начните новый диалог."}
          </div>
        )}
        {filtered.map((chat) => {
          const name = chat.otherUser?.displayName ?? "Пользователь";
          const isOnline = chat.otherUser ? online.has(chat.otherUser.id) || chat.otherUser.status === "online" : false;
          return (
            <button
              key={chat.id}
              className={cn("chat-item", chat.id === activeChatId && "chat-item--active")}
              onClick={() => onSelect(chat)}
            >
              <Avatar name={name} url={chat.otherUser?.avatarUrl} presence={isOnline ? "online" : undefined} />
              <div className="chat-item__body">
                <div className="chat-item__row">
                  <span className="chat-item__name">{name}</span>
                  <span className="chat-item__time">{formatRelative(chat.createdAt)}</span>
                </div>
                <div className="chat-item__preview">@{chat.otherUser?.username ?? "—"}</div>
              </div>
              {chat.unreadCount > 0 && (
                <div className="chat-item__meta">
                  <Badge>{chat.unreadCount > 99 ? "99+" : chat.unreadCount}</Badge>
                </div>
              )}
            </button>
          );
        })}
      </div>
    </aside>
  );
}

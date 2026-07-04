import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { formatRelative } from "@shared/lib/format";
import { Avatar, Badge, IconButton, Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { roleLabel, type Chat, type Group } from "@shared/api/types";
import { useChats } from "@entities/chat/queries";
import { useGroups } from "@entities/group/queries";
import { useMessageSearch } from "@entities/message/search";
import { usePresenceStore } from "@shared/store/presence";

interface Props {
  activeId: string | null;
  onSelect: (chat: Chat) => void;
  onSelectGroup: (group: Group) => void;
  onNewChat: () => void;
  onNewGroup: () => void;
}

export function ChatList({ activeId, onSelect, onSelectGroup, onNewChat, onNewGroup }: Props) {
  const { data: chats, isPending } = useChats();
  const { data: groups } = useGroups();
  const [query, setQuery] = useState("");
  const online = usePresenceStore((s) => s.online);
  const navigate = useNavigate();
  const { data: messageHits } = useMessageSearch(query);

  const openHit = (chatType: string, chatId: string) =>
    navigate(chatType === "group" ? `/group/${chatId}` : `/chat/${chatId}`);

  const q = query.trim().toLowerCase();
  const filtered = useMemo(() => {
    const list = chats ?? [];
    if (!q) return list;
    return list.filter((c) => c.otherUser?.displayName.toLowerCase().includes(q) || c.otherUser?.username.toLowerCase().includes(q));
  }, [chats, q]);
  const filteredGroups = useMemo(() => {
    const list = groups ?? [];
    if (!q) return list;
    return list.filter((g) => g.name.toLowerCase().includes(q));
  }, [groups, q]);

  return (
    <aside className="chatlist">
      <div className="chatlist__header">
        <h1 className="chatlist__title">Сообщения</h1>
        <div style={{ display: "flex", gap: 2 }}>
          <IconButton label="Новая группа" onClick={onNewGroup}>
            <Icon.Users />
          </IconButton>
          <IconButton label="Новый чат" onClick={onNewChat}>
            <Icon.Plus />
          </IconButton>
        </div>
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

        {filteredGroups.length > 0 && <div className="chatlist__section">Группы</div>}
        {filteredGroups.map((group) => (
          <button
            key={group.id}
            className={cn("chat-item", group.id === activeId && "chat-item--active")}
            onClick={() => onSelectGroup(group)}
          >
            <Avatar name={group.name} url={group.avatarUrl} size={44} />
            <div className="chat-item__body">
              <div className="chat-item__row">
                <span className="chat-item__name">{group.name}</span>
              </div>
              <div className="chat-item__preview">Группа · от {roleLabel(group.minRoleLevel)} и выше</div>
            </div>
          </button>
        ))}

        {filtered.length > 0 && <div className="chatlist__section">Личные чаты</div>}
        {!isPending && filtered.length === 0 && filteredGroups.length === 0 && (
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
              className={cn("chat-item", chat.id === activeId && "chat-item--active")}
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

        {q.length >= 2 && messageHits && messageHits.length > 0 && (
          <>
            <div className="chatlist__section">Сообщения</div>
            {messageHits.map((hit) => (
              <button
                key={hit.messageId}
                className="chat-item"
                onClick={() => openHit(hit.chatType, hit.chatId)}
              >
                <Avatar name={hit.senderName} size={44} />
                <div className="chat-item__body">
                  <div className="chat-item__row">
                    <span className="chat-item__name">{hit.senderName}</span>
                    <span className="chat-item__time">{formatRelative(hit.createdAt)}</span>
                  </div>
                  <div className="chat-item__preview">{hit.text}</div>
                </div>
              </button>
            ))}
          </>
        )}
      </div>
    </aside>
  );
}

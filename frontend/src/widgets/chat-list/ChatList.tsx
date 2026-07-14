import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { cn } from "@shared/lib/cn";
import { formatRelative } from "@shared/lib/format";
import { Avatar, Badge, IconButton, Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { roleLabel, type Chat, type ChatType, type Group } from "@shared/api/types";
import { useChats } from "@entities/chat/queries";
import { useGroups } from "@entities/group/queries";
import { useMessageSearch } from "@entities/message/search";
import { isMuted, useMutes } from "@entities/notif-prefs/queries";
import { chatKey, folderChatSet, isArchived, useArchived, useFolders } from "@entities/chat-folders/queries";
import { ChatContextMenu, type MenuTarget } from "@features/chat-folders/ChatContextMenu";
import { FolderManager } from "@features/chat-folders/FolderManager";
import { FindGroupModal } from "@features/new-chat/FindGroupModal";
import { usePresenceStore } from "@shared/store/presence";

interface Props {
  /** Which column to render: private chats ("chats") or groups ("communities"). */
  view: "chats" | "communities";
  activeId: string | null;
  onSelect: (chat: Chat) => void;
  onSelectGroup: (group: Group) => void;
  onNewChat: () => void;
  onNewGroup: () => void;
}

/** Chat-list tab: the fixed "all"/"unread" pseudo-tabs or a folder id. */
type Tab = "all" | "unread" | string;

export function ChatList({ view, activeId, onSelect, onSelectGroup, onNewChat, onNewGroup }: Props) {
  const communities = view === "communities";
  const { data: chats, isPending } = useChats();
  const { data: groups } = useGroups();
  const [query, setQuery] = useState("");
  const online = usePresenceStore((s) => s.online);
  const navigate = useNavigate();
  const { data: messageHits } = useMessageSearch(query);
  const { mutedSet } = useMutes();
  const { folders } = useFolders();
  const { archivedSet } = useArchived();

  const [tab, setTab] = useState<Tab>("all");
  const [showArchive, setShowArchive] = useState(false);
  const [menuTarget, setMenuTarget] = useState<MenuTarget | null>(null);
  const [managerOpen, setManagerOpen] = useState(false);
  const [findOpen, setFindOpen] = useState(false);

  const openHit = (chatType: string, chatId: string) =>
    navigate(chatType === "group" ? `/group/${chatId}` : `/chat/${chatId}`);

  const q = query.trim().toLowerCase();
  const searching = q.length > 0;

  // The active folder's chat set (empty when a fixed tab is selected).
  const activeFolder = folders.find((f) => f.id === tab);
  const folderSet = useMemo(() => (activeFolder ? folderChatSet(activeFolder) : null), [activeFolder]);

  // Tab + archive filters apply only while not searching: search is global.
  const visible = (chatType: ChatType, chatId: string, unread: number): boolean => {
    if (searching) return true;
    if (isArchived(archivedSet, chatType, chatId)) return false;
    if (tab === "unread") return unread > 0;
    if (folderSet) return folderSet.has(chatKey(chatType, chatId));
    return true;
  };

  const filtered = useMemo(() => {
    const list = chats ?? [];
    const byQuery = q
      ? list.filter(
          (c) => c.otherUser?.displayName.toLowerCase().includes(q) || c.otherUser?.username.toLowerCase().includes(q),
        )
      : list;
    return byQuery.filter((c) => visible("private", c.id, c.unreadCount));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chats, q, tab, archivedSet, folderSet, searching]);

  const filteredGroups = useMemo(() => {
    const list = groups ?? [];
    const byQuery = q ? list.filter((g) => g.name.toLowerCase().includes(q)) : list;
    // Groups have no unread counter yet — the "unread" tab is private-only.
    return byQuery.filter((g) => (tab === "unread" && !searching ? false : visible("group", g.id, 0)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groups, q, tab, archivedSet, folderSet, searching]);

  const archivedChats = useMemo(
    () => (chats ?? []).filter((c) => isArchived(archivedSet, "private", c.id)),
    [chats, archivedSet],
  );
  const archivedGroups = useMemo(
    () => (groups ?? []).filter((g) => isArchived(archivedSet, "group", g.id)),
    [groups, archivedSet],
  );
  // The archive folds under whichever column is active: groups in Сообщества,
  // private chats in Чаты.
  const viewArchivedCount = communities ? archivedGroups.length : archivedChats.length;

  const onRowMenu = (e: React.MouseEvent, chatType: ChatType, chatId: string) => {
    e.preventDefault();
    setMenuTarget({ chatType, chatId, x: e.clientX, y: e.clientY });
  };

  const groupRow = (group: Group) => (
    <button
      key={group.id}
      className={cn("chat-item", group.id === activeId && "chat-item--active")}
      onClick={() => onSelectGroup(group)}
      onContextMenu={(e) => onRowMenu(e, "group", group.id)}
    >
      <Avatar name={group.name} url={group.avatarUrl} size={44} />
      <div className="chat-item__body">
        <div className="chat-item__row">
          <span className="chat-item__name">{group.name}</span>
          {isMuted(mutedSet, "group", group.id) && (
            <span className="chat-item__muted" title="Уведомления отключены">
              <Icon.BellOff size={14} />
            </span>
          )}
        </div>
        <div className="chat-item__preview">Группа · от {roleLabel(group.minRoleLevel)} и выше</div>
      </div>
    </button>
  );

  const chatRow = (chat: Chat) => {
    const name = chat.otherUser?.displayName ?? "Пользователь";
    const isOnline = chat.otherUser ? online.has(chat.otherUser.id) || chat.otherUser.status === "online" : false;
    return (
      <button
        key={chat.id}
        className={cn("chat-item", chat.id === activeId && "chat-item--active")}
        onClick={() => onSelect(chat)}
        onContextMenu={(e) => onRowMenu(e, "private", chat.id)}
      >
        <Avatar name={name} url={chat.otherUser?.avatarUrl} presence={isOnline ? "online" : undefined} />
        <div className="chat-item__body">
          <div className="chat-item__row">
            <span className="chat-item__name">{name}</span>
            <span className="chat-item__time">{formatRelative(chat.createdAt)}</span>
          </div>
          <div className="chat-item__preview">@{chat.otherUser?.username ?? "—"}</div>
        </div>
        <div className="chat-item__meta">
          {isMuted(mutedSet, "private", chat.id) && (
            <span className="chat-item__muted" title="Уведомления отключены">
              <Icon.BellOff size={14} />
            </span>
          )}
          {chat.unreadCount > 0 && (
            <Badge muted={isMuted(mutedSet, "private", chat.id)}>
              {chat.unreadCount > 99 ? "99+" : chat.unreadCount}
            </Badge>
          )}
        </div>
      </button>
    );
  };

  const emptyLabel = query
    ? "Ничего не найдено"
    : communities
      ? "Пока нет сообществ. Создайте группу или найдите существующую."
      : tab === "unread"
        ? "Нет непрочитанных чатов"
        : activeFolder
          ? "В папке пока нет чатов. Добавьте чат через правый клик по нему."
          : "Пока нет чатов. Начните новый диалог.";

  return (
    <aside className="chatlist">
      <div className="chatlist__header">
        <h1 className="chatlist__title">{communities ? "Сообщества" : "Сообщения"}</h1>
        <div style={{ display: "flex", gap: 2 }}>
          {communities ? (
            <IconButton label="Новая группа" onClick={onNewGroup}>
              <Icon.Plus />
            </IconButton>
          ) : (
            <>
              <IconButton label="Папки чатов" onClick={() => setManagerOpen(true)}>
                <Icon.FolderPlus />
              </IconButton>
              <IconButton label="Новый чат" onClick={onNewChat}>
                <Icon.Plus />
              </IconButton>
            </>
          )}
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

      {!searching && !communities && (
        <div className="chatlist__tabs" role="tablist">
          <button
            className={cn("chatlist__tab", tab === "all" && "chatlist__tab--active")}
            role="tab"
            onClick={() => setTab("all")}
          >
            Все
          </button>
          <button
            className={cn("chatlist__tab", tab === "unread" && "chatlist__tab--active")}
            role="tab"
            onClick={() => setTab("unread")}
          >
            Непрочитанные
          </button>
          {folders.map((f) => (
            <button
              key={f.id}
              className={cn("chatlist__tab", tab === f.id && "chatlist__tab--active")}
              role="tab"
              onClick={() => setTab(f.id)}
            >
              {f.name}
            </button>
          ))}
        </div>
      )}

      <div className="chatlist__scroll">
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
            <Spinner />
          </div>
        )}

        {communities && (
          <>
            <div className="chatlist__section" style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
              <span>Группы</span>
              <button
                type="button"
                onClick={() => setFindOpen(true)}
                style={{ border: "none", background: "none", color: "var(--acc)", cursor: "pointer", fontSize: 12, fontWeight: 700, textTransform: "none", letterSpacing: 0 }}
              >
                Найти группу
              </button>
            </div>
            {filteredGroups.map(groupRow)}
            {!isPending && filteredGroups.length === 0 && (
              <div style={{ padding: 24, textAlign: "center", color: "var(--color-text-secondary)", fontSize: 14 }}>
                {emptyLabel}
              </div>
            )}
          </>
        )}

        {!communities && (
          <>
            {filtered.length > 0 && <div className="chatlist__section">Личные чаты</div>}
            {!isPending && filtered.length === 0 && (
              <div style={{ padding: 24, textAlign: "center", color: "var(--color-text-secondary)", fontSize: 14 }}>
                {emptyLabel}
              </div>
            )}
            {filtered.map(chatRow)}
          </>
        )}

        {q.length >= 2 && messageHits && messageHits.length > 0 && (
          <>
            <div className="chatlist__section">Сообщения</div>
            {messageHits.map((hit) => (
              <button key={hit.messageId} className="chat-item" onClick={() => openHit(hit.chatType, hit.chatId)}>
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

        {!searching && tab === "all" && viewArchivedCount > 0 && (
          <>
            <button className="chatlist__archive" onClick={() => setShowArchive((v) => !v)}>
              <Icon.Archive size={18} />
              <span>Архив</span>
              <span className="chatlist__archive-count">{viewArchivedCount}</span>
              <span className="chatlist__archive-chevron">{showArchive ? "▾" : "▸"}</span>
            </button>
            {showArchive && (communities ? archivedGroups.map(groupRow) : archivedChats.map(chatRow))}
          </>
        )}
      </div>

      {menuTarget && <ChatContextMenu target={menuTarget} onClose={() => setMenuTarget(null)} />}
      <FolderManager open={managerOpen} onClose={() => setManagerOpen(false)} />
      <FindGroupModal open={findOpen} onClose={() => setFindOpen(false)} />
    </aside>
  );
}

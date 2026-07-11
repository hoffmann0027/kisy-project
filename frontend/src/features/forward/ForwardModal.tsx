// Target picker for forwarding (stage D): pick one of the actor's existing
// chats or groups. Server enforces the hierarchy rule and returns a clear
// error if the target broadens the audience, surfaced here as a toast.
import { useMemo, useState } from "react";
import { Avatar, Modal } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { useChats } from "@entities/chat/queries";
import { useGroups } from "@entities/group/queries";

export interface ForwardTarget {
  chatType: "private" | "group";
  chatId: string;
  title: string;
  /** Peer user id for private chats — enables E2EE re-encryption. */
  peerUserId?: string;
}

interface Props {
  open: boolean;
  count: number;
  onClose: () => void;
  onPick: (target: ForwardTarget) => void;
}

export function ForwardModal({ open, count, onClose, onPick }: Props) {
  const { data: chats } = useChats();
  const { data: groups } = useGroups();
  const [query, setQuery] = useState("");

  const targets = useMemo<ForwardTarget[]>(() => {
    const q = query.trim().toLowerCase();
    const chatTargets: ForwardTarget[] = (chats ?? []).map((c) => ({
      chatType: "private" as const,
      chatId: c.id,
      title: c.otherUser?.displayName ?? "Диалог",
      peerUserId: c.otherUserId,
    }));
    const groupTargets: ForwardTarget[] = (groups ?? []).map((g) => ({
      chatType: "group" as const,
      chatId: g.id,
      title: g.name,
    }));
    return [...groupTargets, ...chatTargets].filter((t) => !q || t.title.toLowerCase().includes(q));
  }, [chats, groups, query]);

  return (
    <Modal open={open} title={`Переслать (${count})`} onClose={onClose}>
      <input
        className="ui-input"
        placeholder="Поиск чата или группы"
        autoFocus
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div style={{ maxHeight: 360, overflowY: "auto", display: "flex", flexDirection: "column", gap: 2 }}>
        {targets.length === 0 && (
          <div style={{ textAlign: "center", color: "var(--color-text-secondary)", padding: 20, fontSize: 14 }}>
            Ничего не найдено
          </div>
        )}
        {targets.map((t) => (
          <button
            key={`${t.chatType}-${t.chatId}`}
            className="user-row"
            onClick={() => onPick(t)}
          >
            {t.chatType === "group" ? (
              <span className="forward-target__icon">
                <Icon.Users size={20} />
              </span>
            ) : (
              <Avatar name={t.title} size={40} />
            )}
            <div>
              <div className="user-row__name">{t.title}</div>
              <div className="user-row__role">{t.chatType === "group" ? "Группа" : "Личный чат"}</div>
            </div>
          </button>
        ))}
      </div>
    </Modal>
  );
}

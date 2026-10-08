// Target picker for forwarding (stage D): pick one of the actor's existing
// chats or groups. Server enforces the hierarchy rule and returns a clear
// error if the target broadens the audience, surfaced here as a toast.
import { useMemo, useState } from "react";
import { t } from "@shared/i18n";
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
      title: c.otherUser?.displayName ?? t("chat.forward.untitledChat"),
      peerUserId: c.otherUserId,
    }));
    const groupTargets: ForwardTarget[] = (groups ?? []).map((g) => ({
      chatType: "group" as const,
      chatId: g.id,
      title: g.name,
    }));
    return [...groupTargets, ...chatTargets].filter((x) => !q || x.title.toLowerCase().includes(q));
  }, [chats, groups, query]);

  return (
    <Modal open={open} title={t("chat.forward.title", { count })} onClose={onClose}>
      <input
        className="ui-input"
        placeholder={t("chat.forward.searchPlaceholder")}
        autoFocus
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div style={{ maxHeight: 360, overflowY: "auto", display: "flex", flexDirection: "column", gap: 2 }}>
        {targets.length === 0 && (
          <div style={{ textAlign: "center", color: "var(--color-text-secondary)", padding: 20, fontSize: 14 }}>
            {t("chat.list.nothingFound")}
          </div>
        )}
        {targets.map((target) => (
          <button
            key={`${target.chatType}-${target.chatId}`}
            className="user-row"
            onClick={() => onPick(target)}
          >
            {target.chatType === "group" ? (
              <span className="forward-target__icon">
                <Icon.Users size={20} />
              </span>
            ) : (
              <Avatar name={target.title} size={40} />
            )}
            <div>
              <div className="user-row__name">{target.title}</div>
              <div className="user-row__role">
                {target.chatType === "group" ? t("chat.group.group") : t("chat.forward.privateChat")}
              </div>
            </div>
          </button>
        ))}
      </div>
    </Modal>
  );
}

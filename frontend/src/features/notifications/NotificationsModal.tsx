import { useState } from "react";
import { Button, Modal, Spinner } from "@shared/ui";
import { formatRelative } from "@shared/lib/format";
import { roleLabel } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useMarkNotificationsRead, useNotifications } from "@entities/notification/queries";
import { canAnnounce } from "@entities/announcement/queries";
import { AnnouncementsModal } from "@features/announcements/AnnouncementsModal";

interface Props {
  open: boolean;
  onClose: () => void;
}

function describe(type: string, payload: Record<string, unknown>): string {
  if (type === "mention") return "Вас упомянули в сообщении";
  // Moderation notices carry their full sentence, reason included, as the
  // server wrote it — the same text the push showed.
  if (type === "group_sanction" && typeof payload.text === "string") return payload.text;
  return type;
}

/** An announcement from levels 1-3: title, text and who wrote it. */
function AnnouncementView({ payload }: { payload: Record<string, unknown> }) {
  const author = (payload.author ?? {}) as { displayName?: string; roleLevel?: number };
  const role = roleLabel(author.roleLevel || null);
  return (
    <div className="announce-note">
      <div className="announce-note__title">{String(payload.title ?? "")}</div>
      <div className="announce-note__body">{String(payload.body ?? "")}</div>
      <div className="announce-note__author">
        {author.displayName ?? ""}
        {role && ` · ${role}`}
      </div>
    </div>
  );
}

export function NotificationsModal({ open, onClose }: Props) {
  const { data, isPending } = useNotifications();
  const markRead = useMarkNotificationsRead();
  const roleLevel = useAuthStore((s) => s.user?.roleLevel);
  const [composing, setComposing] = useState(false);

  return (
    <Modal open={open} title="Уведомления" onClose={onClose}>
      {canAnnounce(roleLevel) && (
        <>
          <Button variant="secondary" onClick={() => setComposing(true)}>
            Создать уведомление
          </Button>
          <AnnouncementsModal open={composing} onClose={() => setComposing(false)} authorLevel={roleLevel!} />
        </>
      )}
      {(data?.unreadCount ?? 0) > 0 && (
        <Button variant="secondary" onClick={() => markRead.mutate(undefined)} loading={markRead.isPending}>
          Отметить все как прочитанные
        </Button>
      )}
      <div style={{ maxHeight: 360, overflowY: "auto", display: "flex", flexDirection: "column", gap: 6 }}>
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
            <Spinner />
          </div>
        )}
        {!isPending && (data?.notifications.length ?? 0) === 0 && (
          <div style={{ textAlign: "center", color: "var(--color-text-secondary)", padding: 20, fontSize: 14 }}>
            Нет уведомлений
          </div>
        )}
        {data?.notifications.map((n) => (
          <div
            key={n.id}
            style={{
              padding: "10px 12px",
              borderRadius: "var(--radius-md)",
              background: n.isRead ? "transparent" : "var(--color-accent-soft)",
              border: "1px solid var(--color-border)",
            }}
          >
            {n.type === "announcement" ? (
              <AnnouncementView payload={n.payload} />
            ) : (
              <div style={{ fontSize: 14 }}>{describe(n.type, n.payload)}</div>
            )}
            <div style={{ fontSize: 12, color: "var(--color-text-tertiary)", marginTop: 2 }}>
              {formatRelative(n.createdAt)}
            </div>
          </div>
        ))}
      </div>
    </Modal>
  );
}

import { Button, Modal, Spinner } from "@shared/ui";
import { formatRelative } from "@shared/lib/format";
import { useMarkNotificationsRead, useNotifications } from "@entities/notification/queries";

interface Props {
  open: boolean;
  onClose: () => void;
}

function describe(type: string): string {
  if (type === "mention") return "Вас упомянули в сообщении";
  return type;
}

export function NotificationsModal({ open, onClose }: Props) {
  const { data, isPending } = useNotifications();
  const markRead = useMarkNotificationsRead();

  return (
    <Modal open={open} title="Уведомления" onClose={onClose}>
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
            <div style={{ fontSize: 14 }}>{describe(n.type)}</div>
            <div style={{ fontSize: 12, color: "var(--color-text-tertiary)", marginTop: 2 }}>
              {formatRelative(n.createdAt)}
            </div>
          </div>
        ))}
      </div>
    </Modal>
  );
}

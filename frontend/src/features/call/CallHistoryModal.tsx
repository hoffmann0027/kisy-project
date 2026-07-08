import "./call-history.css";
import { Avatar, Modal, Spinner } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { cn } from "@shared/lib/cn";
import { formatRelative } from "@shared/lib/format";
import type { CallLogItem } from "@shared/api/types";
import { useCallHistory } from "@entities/call/queries";

interface Props {
  open: boolean;
  onClose: () => void;
}

function duration(seconds: number): string {
  if (seconds <= 0) return "";
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return m > 0 ? `${m} мин ${s} с` : `${s} с`;
}

// statusLabel describes the outcome from the viewer's perspective.
function statusLabel(c: CallLogItem): string {
  switch (c.status) {
    case "completed":
      return duration(c.durationSeconds) || "Завершён";
    case "missed":
      return c.direction === "incoming" ? "Пропущенный" : "Нет ответа";
    case "rejected":
      return c.direction === "incoming" ? "Отклонён вами" : "Отклонён";
    case "canceled":
      return "Отменён";
    default:
      return "Сбой";
  }
}

export function CallHistoryModal({ open, onClose }: Props) {
  const { data: calls, isPending } = useCallHistory(open);

  return (
    <Modal open={open} title="История звонков" onClose={onClose}>
      <div className="callhist">
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 24 }}>
            <Spinner />
          </div>
        )}
        {!isPending && (calls?.length ?? 0) === 0 && (
          <div className="callhist__empty">Звонков пока нет.</div>
        )}
        {calls?.map((c) => {
          const missed = c.status === "missed" || c.status === "failed";
          return (
            <div key={c.id} className="callhist__item">
              <Avatar name={c.peer.displayName} url={c.peer.avatarUrl} size={40} />
              <div className="callhist__body">
                <div className={cn("callhist__name", missed && "callhist__name--missed")}>
                  {c.peer.displayName}
                </div>
                <div className="callhist__meta">
                  <span className={cn("callhist__dir", "callhist__dir--" + c.direction)}>
                    {c.direction === "incoming" ? <Icon.Reply size={13} /> : <Icon.Send size={13} />}
                  </span>
                  <span>{statusLabel(c)}</span>
                </div>
              </div>
              <div className="callhist__time">{formatRelative(c.startedAt)}</div>
            </div>
          );
        })}
      </div>
    </Modal>
  );
}

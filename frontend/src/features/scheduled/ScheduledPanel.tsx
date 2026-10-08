// The chat's pending scheduled messages (UPD3 stage I): a modal listing
// each with its text (E2EE rows resolve from the local sched cache), send
// time, reschedule and cancel controls.
import { useEffect, useState } from "react";
import { intlLocale, t } from "@shared/i18n";
import { hourStyle } from "@shared/lib/format";
import { Modal, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { ScheduledMessage } from "@shared/api/types";
import {
  scheduledDisplayText,
  useCancelScheduled,
  useRescheduleMessage,
} from "@entities/message/scheduled";

interface Props {
  open: boolean;
  items: ScheduledMessage[];
  onClose: () => void;
}

function toLocalInput(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function formatSendAt(iso: string): string {
  return new Date(iso).toLocaleString(intlLocale(), {
    day: "numeric",
    month: "long",
    hour: hourStyle(),
    minute: "2-digit",
  });
}

function Row({ item }: { item: ScheduledMessage }) {
  const cancel = useCancelScheduled();
  const reschedule = useRescheduleMessage();
  const [text, setText] = useState<string | null>(item.text);
  const [editingTime, setEditingTime] = useState(false);
  const [newTime, setNewTime] = useState(() => toLocalInput(new Date(item.sendAt)));

  // E2EE rows: resolve the locally cached plaintext.
  useEffect(() => {
    if (item.text != null) return;
    let alive = true;
    void scheduledDisplayText(item).then((resolved) => {
      if (alive) setText(resolved);
    });
    return () => {
      alive = false;
    };
  }, [item]);

  const saveTime = () => {
    const d = new Date(newTime);
    if (Number.isNaN(d.getTime()) || d.getTime() < Date.now() + 10_000) {
      toast.error(t("chat.scheduled.timeInPast"));
      return;
    }
    reschedule.mutate(
      { id: item.id, sendAt: d },
      {
        onSuccess: () => setEditingTime(false),
        onError: () => toast.error(t("chat.scheduled.rescheduleFailed")),
      },
    );
  };

  return (
    <li className="schedlist__row">
      <div className="schedlist__body">
        <div className="schedlist__text">
          {text ?? (item.ciphertext ? t("chat.bubble.encrypted") : t("chat.scheduled.attachment"))}
        </div>
        {editingTime ? (
          <div className="schedlist__edit-time">
            <input
              className="ui-input"
              type="datetime-local"
              value={newTime}
              min={toLocalInput(new Date())}
              onChange={(e) => setNewTime(e.target.value)}
            />
            <button className="schedlist__btn" title={t("chat.action.save")} onClick={saveTime}>
              <Icon.Check size={16} />
            </button>
          </div>
        ) : (
          <div className="schedlist__time">
            <Icon.Calendar size={14} />
            {formatSendAt(item.sendAt)}
          </div>
        )}
      </div>
      <div className="schedlist__actions">
        <button className="schedlist__btn" title={t("chat.scheduled.reschedule")} onClick={() => setEditingTime((v) => !v)}>
          <Icon.Edit size={16} />
        </button>
        <button
          className="schedlist__btn schedlist__btn--danger"
          title={t("chat.scheduled.cancel")}
          onClick={() =>
            cancel.mutate(item.id, { onError: () => toast.error(t("chat.scheduled.cancelFailed")) })
          }
        >
          <Icon.Trash size={16} />
        </button>
      </div>
    </li>
  );
}

export function ScheduledPanel({ open, items, onClose }: Props) {
  return (
    <Modal open={open} title={t("chat.scheduled.title")} onClose={onClose}>
      {items.length === 0 ? (
        <p className="schedlist__empty">
          {t("chat.scheduled.empty")}
        </p>
      ) : (
        <ul className="schedlist">
          {items.map((m) => (
            <Row key={m.id} item={m} />
          ))}
        </ul>
      )}
    </Modal>
  );
}

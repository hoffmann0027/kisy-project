// The chat's pending scheduled messages (UPD3 stage I): a modal listing
// each with its text (E2EE rows resolve from the local sched cache), send
// time, reschedule and cancel controls.
import { useEffect, useState } from "react";
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
  return new Date(iso).toLocaleString("ru-RU", {
    day: "numeric",
    month: "long",
    hour: "2-digit",
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
    void scheduledDisplayText(item).then((t) => {
      if (alive) setText(t);
    });
    return () => {
      alive = false;
    };
  }, [item]);

  const saveTime = () => {
    const d = new Date(newTime);
    if (Number.isNaN(d.getTime()) || d.getTime() < Date.now() + 10_000) {
      toast.error("Время должно быть в будущем");
      return;
    }
    reschedule.mutate(
      { id: item.id, sendAt: d },
      {
        onSuccess: () => setEditingTime(false),
        onError: () => toast.error("Не удалось перенести отправку"),
      },
    );
  };

  return (
    <li className="schedlist__row">
      <div className="schedlist__body">
        <div className="schedlist__text">
          {text ?? (item.ciphertext ? "🔒 Зашифрованное сообщение" : "Вложение")}
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
            <button className="schedlist__btn" title="Сохранить" onClick={saveTime}>
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
        <button className="schedlist__btn" title="Перенести" onClick={() => setEditingTime((v) => !v)}>
          <Icon.Edit size={16} />
        </button>
        <button
          className="schedlist__btn schedlist__btn--danger"
          title="Отменить отправку"
          onClick={() =>
            cancel.mutate(item.id, { onError: () => toast.error("Не удалось отменить отправку") })
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
    <Modal open={open} title="Запланированные сообщения" onClose={onClose}>
      {items.length === 0 ? (
        <p className="schedlist__empty">
          Нет запланированных сообщений. Кнопка «часы» рядом с отправкой планирует сообщение на будущее.
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

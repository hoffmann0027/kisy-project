// "Send later" dropdown (UPD3 stage I): presets + a datetime-local field.
// For E2EE-capable private chats a warning explains the epoch-drift caveat
// (path A, docs/security.md): a scheduled encrypted message can become
// unreadable if the chat's keys rotate before send time.
import { useEffect, useRef, useState } from "react";
import { Button } from "@shared/ui";

interface Props {
  /** Show the E2EE epoch-drift warning (private chat with encryption). */
  e2eeWarning: boolean;
  onPick: (sendAt: Date) => void;
  onClose: () => void;
}

// datetime-local wants "YYYY-MM-DDTHH:MM" in local time.
function toLocalInput(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function presetTonight(): Date {
  const d = new Date();
  d.setHours(20, 0, 0, 0);
  if (d.getTime() <= Date.now()) d.setDate(d.getDate() + 1);
  return d;
}

function presetTomorrowMorning(): Date {
  const d = new Date();
  d.setDate(d.getDate() + 1);
  d.setHours(9, 0, 0, 0);
  return d;
}

export function SchedulePicker({ e2eeWarning, onPick, onClose }: Props) {
  const [custom, setCustom] = useState(() => toLocalInput(new Date(Date.now() + 60 * 60 * 1000)));
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) onClose();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  const pick = (d: Date) => {
    if (d.getTime() < Date.now() + 10_000) return;
    onPick(d);
  };

  const customDate = new Date(custom);
  const customValid = !Number.isNaN(customDate.getTime()) && customDate.getTime() > Date.now() + 10_000;

  return (
    <div className="schedpick" ref={rootRef} role="menu">
      <div className="schedpick__title">Отправить позже</div>
      <button className="schedpick__item" onClick={() => pick(new Date(Date.now() + 60 * 60 * 1000))}>
        Через час
      </button>
      <button className="schedpick__item" onClick={() => pick(presetTonight())}>
        Сегодня в 20:00
      </button>
      <button className="schedpick__item" onClick={() => pick(presetTomorrowMorning())}>
        Завтра в 9:00
      </button>
      <div className="schedpick__custom">
        <input
          className="ui-input"
          type="datetime-local"
          value={custom}
          min={toLocalInput(new Date())}
          onChange={(e) => setCustom(e.target.value)}
        />
        <Button disabled={!customValid} onClick={() => pick(customDate)}>
          ОК
        </Button>
      </div>
      {e2eeWarning && (
        <p className="schedpick__warn">
          Чат зашифрован: текст шифруется сейчас. Если ключи чата обновятся до отправки, сообщение может
          прийти нечитаемым.
        </p>
      )}
    </div>
  );
}

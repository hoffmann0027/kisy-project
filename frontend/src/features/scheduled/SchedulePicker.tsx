// "Send later" dropdown (UPD3 stage I): presets + a datetime-local field.
// For E2EE-capable private chats a warning explains the epoch-drift caveat
// (path A, docs/security.md): a scheduled encrypted message can become
// unreadable if the chat's keys rotate before send time.
import { useEffect, useRef, useState } from "react";
import { intlLocale, t } from "@shared/i18n";
import { Button } from "@shared/ui";
import { useBackHandler } from "@shared/lib/backStack";

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

/** A preset's clock time ("20:00", "8:00 PM") in the language on screen. */
function clock(hours: number): string {
  return new Date(2000, 0, 1, hours).toLocaleTimeString(intlLocale(), { hour: "numeric", minute: "2-digit" });
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
  useBackHandler(true, onClose); // mounted only while the picker is open
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
      <div className="schedpick__title">{t("chat.composer.sendLater")}</div>
      <button className="schedpick__item" onClick={() => pick(new Date(Date.now() + 60 * 60 * 1000))}>
        {t("chat.schedule.inAnHour")}
      </button>
      <button className="schedpick__item" onClick={() => pick(presetTonight())}>
        {t("chat.schedule.tonight", { time: clock(20) })}
      </button>
      <button className="schedpick__item" onClick={() => pick(presetTomorrowMorning())}>
        {t("chat.schedule.tomorrowMorning", { time: clock(9) })}
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
          {t("chat.schedule.ok")}
        </Button>
      </div>
      {e2eeWarning && (
        <p className="schedpick__warn">
          {t("chat.schedule.e2eeWarning")}
        </p>
      )}
    </div>
  );
}

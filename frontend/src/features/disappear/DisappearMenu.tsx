// Chat-header control for disappearing messages (stage J): an hourglass
// that opens a dropdown — off / 1h / 24h / 7d. The timer is a chat-wide
// mode (every participant sees the change); expired messages are
// hard-deleted server-side and purged from local caches.
import { useEffect, useRef, useState } from "react";
import { Icon } from "@shared/ui/icons";
import { toast } from "@shared/ui";
import type { ChatType } from "@shared/api/types";
import { ttlLabel, useDisappearSetting, useSetDisappearing } from "@entities/chat/disappearing";

const OPTIONS: { ttl: number; label: string }[] = [
  { ttl: 3600, label: "1 час" },
  { ttl: 86400, label: "24 часа" },
  { ttl: 7 * 86400, label: "7 дней" },
];

interface Props {
  chatType: ChatType;
  chatId: string;
}

export function DisappearMenu({ chatType, chatId }: Props) {
  const { data: setting } = useDisappearSetting(chatType, chatId);
  const setDisappearing = useSetDisappearing(chatType, chatId);
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const active = setting?.ttlSeconds != null && setting.ttlSeconds > 0;

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  const apply = (ttl: number | null) => {
    setDisappearing.mutate(ttl, {
      onSuccess: () =>
        toast.success(ttl ? `Новые сообщения исчезают через ${ttlLabel(ttl)}` : "Исчезающие сообщения выключены"),
      onError: () => toast.error("Не удалось изменить таймер"),
    });
    setOpen(false);
  };

  return (
    <div className="mutemenu" ref={rootRef}>
      <button
        className={`conv__call mutemenu__toggle${active ? " disappear-toggle--active" : ""}`}
        title={
          active ? `Исчезающие сообщения: ${ttlLabel(setting!.ttlSeconds!)}` : "Исчезающие сообщения"
        }
        onClick={() => setOpen((v) => !v)}
      >
        <Icon.Timer size={20} />
      </button>
      {open && (
        <div className="mutemenu__dropdown" role="menu">
          <div className="schedpick__title">Исчезающие сообщения</div>
          {OPTIONS.map((o) => (
            <button key={o.ttl} className="mutemenu__item" onClick={() => apply(o.ttl)}>
              {o.label}
              {setting?.ttlSeconds === o.ttl && " ✓"}
            </button>
          ))}
          <button className="mutemenu__item" onClick={() => apply(null)}>
            Выключить{!active && " ✓"}
          </button>
          <p className="schedpick__warn" style={{ color: "var(--color-text-tertiary)" }}>
            Таймер общий для чата: новые сообщения будут удалены безвозвратно по истечении срока.
          </p>
        </div>
      )}
    </div>
  );
}

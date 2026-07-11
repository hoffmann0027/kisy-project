// Mute control for a chat header (stage G): a bell button that opens a
// dropdown — mute for 1h / 8h / forever, or unmute. Reflects the current
// mute state via the icon.
import { useEffect, useRef, useState } from "react";
import { Icon } from "@shared/ui/icons";
import { toast } from "@shared/ui";
import type { ChatType } from "@shared/api/types";
import { isMuted, useMuteChat, useMutes } from "@entities/notif-prefs/queries";

const HOUR = 3600;

interface Props {
  chatType: ChatType;
  chatId: string;
}

export function MuteMenu({ chatType, chatId }: Props) {
  const { mutedSet } = useMutes();
  const muteChat = useMuteChat();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const muted = isMuted(mutedSet, chatType, chatId);

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  const doMute = (untilSeconds?: number, label?: string) => {
    muteChat.mutate(
      { chatType, chatId, untilSeconds, mute: true },
      {
        onSuccess: () => toast.success(label ? `Уведомления отключены (${label})` : "Уведомления отключены"),
        onError: () => toast.error("Не удалось отключить уведомления"),
      },
    );
    setOpen(false);
  };
  const doUnmute = () => {
    muteChat.mutate(
      { chatType, chatId, mute: false },
      {
        onSuccess: () => toast.success("Уведомления включены"),
        onError: () => toast.error("Не удалось включить уведомления"),
      },
    );
    setOpen(false);
  };

  return (
    <div className="mutemenu" ref={rootRef}>
      <button
        className="conv__call mutemenu__toggle"
        title={muted ? "Уведомления отключены" : "Уведомления"}
        onClick={() => setOpen((v) => !v)}
      >
        {muted ? <Icon.BellOff size={20} /> : <Icon.Bell size={20} />}
      </button>
      {open && (
        <div className="mutemenu__dropdown" role="menu">
          {muted ? (
            <button className="mutemenu__item" onClick={doUnmute}>
              Включить уведомления
            </button>
          ) : (
            <>
              <button className="mutemenu__item" onClick={() => doMute(HOUR, "1 час")}>
                Отключить на 1 час
              </button>
              <button className="mutemenu__item" onClick={() => doMute(8 * HOUR, "8 часов")}>
                Отключить на 8 часов
              </button>
              <button className="mutemenu__item" onClick={() => doMute(undefined, "навсегда")}>
                Отключить навсегда
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}

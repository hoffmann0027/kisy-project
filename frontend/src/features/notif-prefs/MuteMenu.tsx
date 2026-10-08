// Mute control for a chat header (stage G): a bell button that opens a
// dropdown — mute for 1h / 8h / forever, or unmute. Reflects the current
// mute state via the icon.
import { useEffect, useRef, useState } from "react";
import { Icon } from "@shared/ui/icons";
import { useBackHandler } from "@shared/lib/backStack";
import { toast } from "@shared/ui";
import type { ChatType } from "@shared/api/types";
import { isMuted, useMuteChat, useMutes } from "@entities/notif-prefs/queries";
import { t } from "@shared/i18n";

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

  useBackHandler(open, () => setOpen(false));

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [open]);

  // `done` is the whole confirmation, for how long included.
  const doMute = (untilSeconds: number | undefined, done: string) => {
    muteChat.mutate(
      { chatType, chatId, untilSeconds, mute: true },
      {
        onSuccess: () => toast.success(done),
        onError: () => toast.error(t("account.mute.muteFailed")),
      },
    );
    setOpen(false);
  };
  const doUnmute = () => {
    muteChat.mutate(
      { chatType, chatId, mute: false },
      {
        onSuccess: () => toast.success(t("account.mute.unmuted")),
        onError: () => toast.error(t("account.mute.unmuteFailed")),
      },
    );
    setOpen(false);
  };

  return (
    <div className="mutemenu" ref={rootRef}>
      <button
        className="conv__call mutemenu__toggle"
        title={muted ? t("account.mute.muted") : t("account.mute.title")}
        onClick={() => setOpen((v) => !v)}
      >
        {muted ? <Icon.BellOff size={20} /> : <Icon.Bell size={20} />}
      </button>
      {open && (
        <div className="mutemenu__dropdown" role="menu">
          {muted ? (
            <button className="mutemenu__item" onClick={doUnmute}>
              {t("account.mute.unmute")}
            </button>
          ) : (
            <>
              <button className="mutemenu__item" onClick={() => doMute(HOUR, t("account.mute.mutedFor1h"))}>
                {t("account.mute.mute1h")}
              </button>
              <button className="mutemenu__item" onClick={() => doMute(8 * HOUR, t("account.mute.mutedFor8h"))}>
                {t("account.mute.mute8h")}
              </button>
              <button className="mutemenu__item" onClick={() => doMute(undefined, t("account.mute.mutedForever"))}>
                {t("account.mute.muteForever")}
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}

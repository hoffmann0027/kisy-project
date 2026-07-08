import { useEffect, useState } from "react";
import { Avatar } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { CallView } from "./useCall";

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

function connLabel(view: CallView): { text: string; warn: boolean } {
  if (view.phase === "connecting") return { text: "Соединение…", warn: false };
  switch (view.conn) {
    case "connected":
      return { text: "", warn: false };
    case "disconnected":
      return { text: "Восстановление связи…", warn: true };
    case "failed":
      return { text: "Соединение потеряно", warn: true };
    default:
      return { text: "Соединение…", warn: false };
  }
}

export function OngoingCall({
  view,
  onHangup,
  onToggleMute,
}: {
  view: CallView;
  onHangup: () => void;
  onToggleMute: () => void;
}) {
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    if (view.phase !== "active" || !view.startedAt) {
      setElapsed(0);
      return;
    }
    const tick = () => setElapsed(Math.floor((Date.now() - view.startedAt!) / 1000));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [view.phase, view.startedAt]);

  const conn = connLabel(view);

  return (
    <div className="call-overlay">
      <div className="call-card">
        <div className="call-card__avatar">
          <Avatar name={view.peer?.displayName ?? "?"} url={view.peer?.avatarUrl} size={96} />
        </div>
        <div className="call-card__name">{view.peer?.displayName}</div>
        {view.phase === "active" ? (
          <div className="call-card__timer">{formatDuration(elapsed)}</div>
        ) : (
          <div className="call-card__status">Соединение…</div>
        )}
        <div className={"call-card__conn" + (conn.warn ? " call-card__conn--warn" : "")}>{conn.text}</div>

        <div className="call-actions">
          <div className="call-btn-group">
            <button
              className={"call-btn call-btn--mute" + (view.muted ? " call-btn--on" : "")}
              onClick={onToggleMute}
              aria-label={view.muted ? "Включить микрофон" : "Выключить микрофон"}
            >
              {view.muted ? <Icon.MicOff size={22} /> : <Icon.Mic size={22} />}
            </button>
            <span className="call-btn__label">{view.muted ? "Вкл. микр." : "Микрофон"}</span>
          </div>
          <div className="call-btn-group">
            <button className="call-btn call-btn--decline" onClick={onHangup} aria-label="Завершить">
              <Icon.PhoneOff size={26} />
            </button>
            <span className="call-btn__label">Завершить</span>
          </div>
        </div>
      </div>
    </div>
  );
}

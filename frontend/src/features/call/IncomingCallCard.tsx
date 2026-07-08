import { Avatar } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { CallPeer } from "./useCall";

export function IncomingCallCard({
  peer,
  onAccept,
  onReject,
}: {
  peer: CallPeer;
  onAccept: () => void;
  onReject: () => void;
}) {
  return (
    <div className="call-overlay">
      <div className="call-card">
        <div className="call-card__avatar call-card__avatar--pulse">
          <Avatar name={peer.displayName} url={peer.avatarUrl} size={96} />
        </div>
        <div className="call-card__name">{peer.displayName}</div>
        <div className="call-card__status">Входящий аудиозвонок…</div>
        <div className="call-actions">
          <div className="call-btn-group">
            <button className="call-btn call-btn--decline" onClick={onReject} aria-label="Отклонить">
              <Icon.PhoneOff size={26} />
            </button>
            <span className="call-btn__label">Отклонить</span>
          </div>
          <div className="call-btn-group">
            <button className="call-btn call-btn--accept" onClick={onAccept} aria-label="Принять">
              <Icon.Phone size={26} />
            </button>
            <span className="call-btn__label">Принять</span>
          </div>
        </div>
      </div>
    </div>
  );
}

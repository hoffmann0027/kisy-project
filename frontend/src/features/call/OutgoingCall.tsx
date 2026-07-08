import { Avatar } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { CallPeer } from "./useCall";

export function OutgoingCall({ peer, onCancel }: { peer: CallPeer; onCancel: () => void }) {
  return (
    <div className="call-overlay">
      <div className="call-card">
        <div className="call-card__avatar call-card__avatar--pulse">
          <Avatar name={peer.displayName} url={peer.avatarUrl} size={96} />
        </div>
        <div className="call-card__name">{peer.displayName}</div>
        <div className="call-card__status">Вызов…</div>
        <div className="call-actions">
          <div className="call-btn-group">
            <button className="call-btn call-btn--decline" onClick={onCancel} aria-label="Отменить">
              <Icon.PhoneOff size={26} />
            </button>
            <span className="call-btn__label">Отменить</span>
          </div>
        </div>
      </div>
    </div>
  );
}

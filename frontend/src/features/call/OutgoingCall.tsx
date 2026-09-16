import { Avatar } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import type { CallPeer, CallView } from "./useCall";
import { SpeakerControl } from "./SpeakerControl";

export function OutgoingCall({
  peer,
  view,
  onCancel,
  onToggleSpeaker,
}: {
  peer: CallPeer;
  view: CallView;
  onCancel: () => void;
  /** Absent where sound cannot be routed (a browser). */
  onToggleSpeaker?: () => void;
}) {
  return (
    <div className="call-overlay">
      <div className="call-card">
        <div className="call-card__avatar call-card__avatar--pulse">
          <Avatar name={peer.displayName} url={peer.avatarUrl} size={96} />
        </div>
        <div className="call-card__name">{peer.displayName}</div>
        <div className="call-card__status">Вызов…</div>
        <div className="call-actions">
          {/* The ringback plays at the ear, like a phone's; the loudspeaker is one tap away. */}
          {onToggleSpeaker && <SpeakerControl view={view} onToggle={onToggleSpeaker} />}
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

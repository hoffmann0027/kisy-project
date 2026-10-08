import { Icon } from "@shared/ui/icons";
import { t } from "@shared/i18n";
import type { CallView } from "./useCall";

// The loudspeaker switch, or — while a headset has the sound — an indicator of
// that headset in its place. A loudspeaker button that does nothing because
// the sound is in someone's ears would be a switch that lies.

export function SpeakerControl({ view, onToggle }: { view: CallView; onToggle: () => void }) {
  if (view.audioRoute === "wired" || view.audioRoute === "bluetooth") {
    const bluetooth = view.audioRoute === "bluetooth";
    const label = bluetooth ? "Bluetooth" : t("hub.call.headset");
    return (
      <div className="call-btn-group">
        <div className="call-btn call-btn--toggle call-btn--route" role="img" aria-label={t("hub.call.audioRoute", { device: label })}>
          {bluetooth ? <Icon.Bluetooth size={22} /> : <Icon.Headphones size={22} />}
        </div>
        <span className="call-btn__label">{label}</span>
      </div>
    );
  }
  return (
    <div className="call-btn-group">
      <button
        className={"call-btn call-btn--toggle" + (view.speaker ? " call-btn--on" : "")}
        onClick={onToggle}
        aria-pressed={view.speaker}
        aria-label={view.speaker ? t("hub.call.speakerTurnOff") : t("hub.call.speakerTurnOn")}
      >
        <Icon.Speaker size={22} />
      </button>
      <span className="call-btn__label">{view.speaker ? t("hub.call.speakerOn") : t("hub.call.speaker")}</span>
    </div>
  );
}

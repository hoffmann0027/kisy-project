// Voice note player bubble: play/pause, waveform progress (click to seek),
// duration, playback speed and the "unlistened" dot.
import { useMemo } from "react";
import { cn } from "@shared/lib/cn";
import type { Attachment } from "@shared/api/types";
import { useVoicePlayer } from "./player";
import { formatDuration, waveformFromBase64, WAVEFORM_BARS } from "./waveform";

interface Props {
  attachment: Attachment;
  mine: boolean;
}

export function VoiceBubble({ attachment, mine }: Props) {
  const { activeId, playing, progress, rate, listened, toggle, seek, cycleRate } = useVoicePlayer();
  const isActive = activeId === attachment.id;
  const isPlaying = isActive && playing;
  const unlistened = !mine && !listened.includes(attachment.id);

  const bars = useMemo(() => {
    const peaks = waveformFromBase64(attachment.waveform ?? "");
    if (peaks.length > 0) return Array.from(peaks);
    // No envelope (decode failed on the sender) — flat placeholder bars.
    return Array.from({ length: WAVEFORM_BARS }, () => 96);
  }, [attachment.waveform]);

  const durationMs = attachment.durationMs ?? 0;
  const shownMs = isActive ? Math.round(progress * durationMs) : durationMs;

  return (
    <div className={cn("voice", mine && "voice--mine")}>
      <button
        className="voice__play"
        onClick={() => toggle(attachment.id, attachment.url)}
        aria-label={isPlaying ? "Пауза" : "Слушать голосовое сообщение"}
      >
        {isPlaying ? (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
            <rect x="6" y="5" width="4" height="14" rx="1" />
            <rect x="14" y="5" width="4" height="14" rx="1" />
          </svg>
        ) : (
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
            <path d="M8 5.5v13l11-6.5z" />
          </svg>
        )}
      </button>

      <div
        className="voice__wave"
        role="slider"
        aria-label="Позиция воспроизведения"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round((isActive ? progress : 0) * 100)}
        tabIndex={0}
        onClick={(e) => {
          if (!isActive) return;
          const rect = e.currentTarget.getBoundingClientRect();
          seek((e.clientX - rect.left) / rect.width);
        }}
        onKeyDown={(e) => {
          if (!isActive) return;
          if (e.key === "ArrowRight") seek(Math.min(1, progress + 0.05));
          if (e.key === "ArrowLeft") seek(Math.max(0, progress - 0.05));
        }}
      >
        {bars.map((v, i) => (
          <span
            key={i}
            className={cn("voice__bar", isActive && i / bars.length <= progress && "voice__bar--played")}
            style={{ height: `${18 + Math.round((v / 255) * 82)}%` }}
          />
        ))}
      </div>

      <div className="voice__meta">
        <span className="voice__time">{formatDuration(shownMs)}</span>
        {unlistened && <span className="voice__dot" title="Не прослушано" />}
      </div>

      <button className="voice__rate" onClick={cycleRate} title="Скорость воспроизведения">
        ×{rate}
      </button>
    </div>
  );
}

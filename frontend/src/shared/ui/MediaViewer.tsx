// Lightbox for chat media (stage C): arrows/swipe-like navigation, zoom
// toggle, download, Esc to close. Keyboard-driven and focus-trapped —
// images no longer open in a new tab.
import { useCallback, useEffect, useRef, useState } from "react";
import { cn } from "@shared/lib/cn";

export interface MediaViewerItem {
  id: string;
  url: string;
  fileName: string;
}

interface Props {
  items: MediaViewerItem[];
  index: number;
  onClose: () => void;
  onIndexChange: (index: number) => void;
}

export function MediaViewer({ items, index, onClose, onIndexChange }: Props) {
  const [zoomed, setZoomed] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const item = items[index];

  const prev = useCallback(() => {
    if (index > 0) {
      setZoomed(false);
      onIndexChange(index - 1);
    }
  }, [index, onIndexChange]);
  const next = useCallback(() => {
    if (index < items.length - 1) {
      setZoomed(false);
      onIndexChange(index + 1);
    }
  }, [index, items.length, onIndexChange]);

  useEffect(() => {
    // Focus the dialog so keys work immediately; restore focus on close.
    const previous = document.activeElement as HTMLElement | null;
    rootRef.current?.focus();
    return () => previous?.focus();
  }, []);

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "ArrowLeft") prev();
    if (e.key === "ArrowRight") next();
    if (e.key === "Tab") {
      // Minimal focus trap: keep focus inside the dialog.
      const focusables = rootRef.current?.querySelectorAll<HTMLElement>("button, a[href]");
      if (!focusables || focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  };

  if (!item) return null;

  return (
    <div
      ref={rootRef}
      className="mviewer"
      role="dialog"
      aria-modal="true"
      aria-label={item.fileName}
      tabIndex={-1}
      onKeyDown={onKeyDown}
      onClick={onClose}
    >
      <div className="mviewer__bar" onClick={(e) => e.stopPropagation()}>
        <span className="mviewer__name">{item.fileName}</span>
        <span className="mviewer__count">
          {index + 1} / {items.length}
        </span>
        <a className="mviewer__btn" href={item.url} download={item.fileName} title="Скачать">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3" strokeLinecap="round" />
          </svg>
        </a>
        <button className="mviewer__btn" onClick={onClose} title="Закрыть (Esc)">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6 6 18M6 6l12 12" strokeLinecap="round" />
          </svg>
        </button>
      </div>

      {index > 0 && (
        <button
          className="mviewer__nav mviewer__nav--prev"
          onClick={(e) => {
            e.stopPropagation();
            prev();
          }}
          aria-label="Предыдущее"
        >
          ‹
        </button>
      )}
      <img
        className={cn("mviewer__img", zoomed && "mviewer__img--zoomed")}
        src={item.url}
        alt={item.fileName}
        onClick={(e) => {
          e.stopPropagation();
          setZoomed((z) => !z);
        }}
      />
      {index < items.length - 1 && (
        <button
          className="mviewer__nav mviewer__nav--next"
          onClick={(e) => {
            e.stopPropagation();
            next();
          }}
          aria-label="Следующее"
        >
          ›
        </button>
      )}
    </div>
  );
}

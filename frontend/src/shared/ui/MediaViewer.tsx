// Lightbox for chat media (stage C): arrows and swipes between pictures,
// download, Esc to close. Keyboard-driven and focus-trapped — images no
// longer open in a new tab.
//
// Zoom follows what people expect from a phone's gallery (a user's
// suggestion — a tap used to double the size around the centre and nothing
// more, so the edges of a picture could not be reached): pinch with two
// fingers, double-tap (or double-click) a point, drag a zoomed picture up to
// its edges, the mouse wheel on a computer, and + / − / 0 on a keyboard. The
// arithmetic lives in shared/lib/zoomPan.
import { useCallback, useEffect, useRef, useState } from "react";
import { cn } from "@shared/lib/cn";
import { useBackHandler } from "@shared/lib/backStack";
import { handleDownloadClick } from "@shared/lib/mediaSrc";
import { t } from "@shared/i18n";
import {
  IDENTITY,
  clampPan,
  swipeDirection,
  toggleZoom,
  zoomTo,
  type Point,
  type View,
} from "@shared/lib/zoomPan";
import { ApiImage } from "./ApiImage";

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

const DOUBLE_TAP_MS = 300;
const TAP_SLOP = 10;

interface Gesture {
  pointers: Map<number, Point>;
  /** Where the current one-finger gesture began, and the view then. */
  start?: { at: Point; view: View; time: number };
  /**
   * Whether the gesture began on the picture. Read at pointerdown: once the
   * stage captures the pointer, later events report the stage as their target.
   */
  onImage: boolean;
  pinch?: { dist: number; mid: Point; view: View };
  moved: boolean;
  multi: boolean;
  lastTap?: { at: Point; time: number };
}

export function MediaViewer({ items, index, onClose, onIndexChange }: Props) {
  const [view, setView] = useState<View>(IDENTITY);
  const [live, setLive] = useState(false);
  const viewRef = useRef(view);
  viewRef.current = view;
  // Escape here is a React handler on the focused dialog, which the Android
  // back gesture never triggers; it claims the gesture directly instead.
  useBackHandler(true, onClose);
  const rootRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<HTMLDivElement>(null);
  const gesture = useRef<Gesture>({ pointers: new Map(), moved: false, multi: false, onImage: false });
  const item = items[index];

  const prev = useCallback(() => {
    if (index > 0) onIndexChange(index - 1);
  }, [index, onIndexChange]);
  const next = useCallback(() => {
    if (index < items.length - 1) onIndexChange(index + 1);
  }, [index, items.length, onIndexChange]);

  // A new picture starts unzoomed.
  useEffect(() => setView(IDENTITY), [index]);

  useEffect(() => {
    // Focus the dialog so keys work immediately; restore focus on close.
    const previous = document.activeElement as HTMLElement | null;
    rootRef.current?.focus();
    return () => previous?.focus();
  }, []);

  /** A client point, measured from the stage centre. */
  const local = useCallback((clientX: number, clientY: number): Point => {
    const r = stageRef.current?.getBoundingClientRect();
    if (!r) return { x: 0, y: 0 };
    return { x: clientX - (r.left + r.width / 2), y: clientY - (r.top + r.height / 2) };
  }, []);

  /** Keep the picture within its own edges at the given view. */
  const fit = useCallback((v: View): View => {
    const stage = stageRef.current;
    const img = stage?.querySelector("img");
    if (!stage || !img) return v;
    // offsetWidth/Height are the laid-out size, before the transform.
    return clampPan(v, { w: img.offsetWidth, h: img.offsetHeight }, { w: stage.clientWidth, h: stage.clientHeight });
  }, []);

  // The wheel needs a non-passive listener to keep the page from scrolling.
  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) return;
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      const v = viewRef.current;
      const factor = Math.exp(-e.deltaY * 0.0015);
      setView(fit(zoomTo(v, v.scale * factor, local(e.clientX, e.clientY))));
    };
    stage.addEventListener("wheel", onWheel, { passive: false });
    return () => stage.removeEventListener("wheel", onWheel);
  }, [fit, local]);

  const onPointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    const g = gesture.current;
    g.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
    if (g.pointers.size === 1) g.onImage = (e.target as HTMLElement).tagName === "IMG";
    e.currentTarget.setPointerCapture?.(e.pointerId);
    if (g.pointers.size === 1) {
      g.start = { at: { x: e.clientX, y: e.clientY }, view: viewRef.current, time: Date.now() };
      g.moved = false;
      g.multi = false;
    } else if (g.pointers.size === 2) {
      const [a, b] = [...g.pointers.values()];
      g.pinch = {
        dist: Math.hypot(a.x - b.x, a.y - b.y) || 1,
        mid: local((a.x + b.x) / 2, (a.y + b.y) / 2),
        view: viewRef.current,
      };
      g.multi = true;
      g.moved = true;
    }
    setLive(true);
  };

  const onPointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    const g = gesture.current;
    if (!g.pointers.has(e.pointerId)) return;
    g.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (g.pointers.size >= 2 && g.pinch) {
      const [a, b] = [...g.pointers.values()];
      const dist = Math.hypot(a.x - b.x, a.y - b.y);
      const mid = local((a.x + b.x) / 2, (a.y + b.y) / 2);
      const zoomed = zoomTo(g.pinch.view, (g.pinch.view.scale * dist) / g.pinch.dist, g.pinch.mid);
      // Two fingers also drag: the picture follows their midpoint.
      setView(fit({ ...zoomed, x: zoomed.x + mid.x - g.pinch.mid.x, y: zoomed.y + mid.y - g.pinch.mid.y }));
      return;
    }
    if (!g.start) return;
    const dx = e.clientX - g.start.at.x;
    const dy = e.clientY - g.start.at.y;
    if (Math.hypot(dx, dy) > TAP_SLOP) g.moved = true;
    if (g.start.view.scale > 1) {
      setView(fit({ ...g.start.view, x: g.start.view.x + dx, y: g.start.view.y + dy }));
    }
  };

  const onPointerEnd = (e: React.PointerEvent<HTMLDivElement>) => {
    const g = gesture.current;
    if (!g.pointers.has(e.pointerId)) return;
    g.pointers.delete(e.pointerId);

    if (g.pointers.size === 1) {
      // One finger left after a pinch: carry on dragging from here, no jump.
      const [rest] = [...g.pointers.values()];
      g.start = { at: rest, view: viewRef.current, time: Date.now() };
      g.pinch = undefined;
      return;
    }
    if (g.pointers.size > 0) return;
    setLive(false);
    g.pinch = undefined;
    const start = g.start;
    g.start = undefined;
    if (!start || e.type === "pointercancel") return;

    const dx = e.clientX - start.at.x;
    const dy = e.clientY - start.at.y;
    if (g.moved) {
      // A swipe on an unzoomed picture turns to the next or previous one.
      if (!g.multi && start.view.scale === 1) {
        const dir = swipeDirection(dx, dy);
        if (dir === 1) next();
        if (dir === -1) prev();
      }
      return;
    }

    // A tap.
    const at = { x: e.clientX, y: e.clientY };
    const last = g.lastTap;
    if (last && Date.now() - last.time < DOUBLE_TAP_MS && Math.hypot(at.x - last.at.x, at.y - last.at.y) < 40) {
      g.lastTap = undefined;
      setView(fit(toggleZoom(viewRef.current, local(at.x, at.y))));
      return;
    }
    g.lastTap = { at, time: Date.now() };
    // A single tap beside an unzoomed picture closes the viewer, as before.
    if (!g.onImage && viewRef.current.scale === 1) onClose();
  };

  const zoomBy = (factor: number) => setView((v) => fit(zoomTo(v, v.scale * factor, { x: 0, y: 0 })));

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") onClose();
    if (e.key === "ArrowLeft") prev();
    if (e.key === "ArrowRight") next();
    if (e.key === "+" || e.key === "=") zoomBy(1.5);
    if (e.key === "-") zoomBy(1 / 1.5);
    if (e.key === "0") setView(IDENTITY);
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
  const zoomed = view.scale > 1;

  return (
    <div
      ref={rootRef}
      className="mviewer"
      role="dialog"
      aria-modal="true"
      aria-label={item.fileName}
      tabIndex={-1}
      onKeyDown={onKeyDown}
    >
      <div
        ref={stageRef}
        className={cn("mviewer__stage", zoomed && "mviewer__stage--zoomed")}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerEnd}
        onPointerCancel={onPointerEnd}
        data-testid="mviewer-stage"
      >
        <ApiImage
          className={cn("mviewer__img", live && "mviewer__img--live")}
          src={item.url}
          alt={item.fileName}
          draggable={false}
          style={{ transform: `translate(${view.x}px, ${view.y}px) scale(${view.scale})` }}
        />
      </div>

      <div className="mviewer__bar">
        <span className="mviewer__name">{item.fileName}</span>
        <span className="mviewer__count">
          {index + 1} / {items.length}
        </span>
        {zoomed && (
          <button className="mviewer__btn" onClick={() => setView(IDENTITY)} title={t("common.media.resetZoom")}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="11" cy="11" r="7" />
              <path d="M8 11h6M20 20l-4.35-4.35" strokeLinecap="round" />
            </svg>
          </button>
        )}
        <a
          className="mviewer__btn"
          href={item.url}
          download={item.fileName}
          title={t("common.media.download")}
          onClick={(e) => handleDownloadClick(e, item.url, item.fileName)}
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3" strokeLinecap="round" />
          </svg>
        </a>
        <button className="mviewer__btn" onClick={onClose} title={t("common.media.close")}>
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6 6 18M6 6l12 12" strokeLinecap="round" />
          </svg>
        </button>
      </div>

      {index > 0 && !zoomed && (
        <button className="mviewer__nav mviewer__nav--prev" onClick={prev} aria-label={t("common.media.previous")}>
          ‹
        </button>
      )}
      {index < items.length - 1 && !zoomed && (
        <button className="mviewer__nav mviewer__nav--next" onClick={next} aria-label={t("common.media.next")}>
          ›
        </button>
      )}
    </div>
  );
}

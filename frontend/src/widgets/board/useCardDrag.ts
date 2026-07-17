import { useCallback, useEffect, useRef, useState } from "react";

/** Travel before a press counts as a drag rather than a tap. */
const START_THRESHOLD_PX = 6;
/** How close to a scroller's edge the pointer must get to auto-scroll it. */
const EDGE_ZONE_PX = 56;
const EDGE_STEP_PX = 14;

export interface CardDrag {
  cardId: string;
  fromColumnId: string;
  /** Ghost top-left in viewport coordinates. */
  x: number;
  y: number;
  width: number;
  /** Where a drop right now would land. */
  overColumnId: string | null;
  overIndex: number;
}

interface Press {
  pointerId: number;
  cardId: string;
  fromColumnId: string;
  startX: number;
  startY: number;
  /** Where inside the card the pointer grabbed it. */
  grabDX: number;
  grabDY: number;
  width: number;
}

/** Which column, and which slot in it, is under this point. */
function hitTest(x: number, y: number): { columnId: string; index: number } | null {
  const col = document.elementFromPoint(x, y)?.closest<HTMLElement>("[data-column-id]");
  const columnId = col?.dataset.columnId;
  if (!col || !columnId) return null;

  const cards = Array.from(col.querySelectorAll<HTMLElement>("[data-card-id]"));
  let index = cards.length;
  for (let i = 0; i < cards.length; i++) {
    const r = cards[i].getBoundingClientRect();
    if (y < r.top + r.height / 2) {
      index = i;
      break;
    }
  }
  return { columnId, index };
}

/**
 * Card dragging built on Pointer Events, so it works with a mouse *and* a
 * finger — native HTML5 drag-and-drop simply never fires on mobile browsers.
 *
 * Touch gestures are split by `touch-action: pan-y` on the card (board.css):
 * a vertical swipe is left to the browser and scrolls the column, while a
 * horizontal one is delivered here and becomes a drag. Scroll containers keep
 * their own touch-action untouched.
 */
export function useCardDrag(onDrop: (cardId: string, toColumnId: string, index: number) => void) {
  const [drag, setDrag] = useState<CardDrag | null>(null);
  const press = useRef<Press | null>(null);
  const active = useRef<CardDrag | null>(null);
  const point = useRef({ x: 0, y: 0 });
  const raf = useRef(0);
  // A drag ends with a click on the card; swallow it so the card modal does
  // not pop open the moment the user drops.
  const suppressClick = useRef(false);

  // Keep the listeners below stable across renders.
  const onDropRef = useRef(onDrop);
  useEffect(() => {
    onDropRef.current = onDrop;
  }, [onDrop]);

  const stopAutoScroll = () => {
    if (raf.current) cancelAnimationFrame(raf.current);
    raf.current = 0;
  };

  const autoScroll = useCallback(() => {
    raf.current = 0;
    if (!active.current) return;
    const { x, y } = point.current;

    const scroller = document.querySelector<HTMLElement>("[data-board-scroll]");
    if (scroller) {
      const r = scroller.getBoundingClientRect();
      if (x < r.left + EDGE_ZONE_PX) scroller.scrollLeft -= EDGE_STEP_PX;
      else if (x > r.right - EDGE_ZONE_PX) scroller.scrollLeft += EDGE_STEP_PX;
    }

    const body = document.elementFromPoint(x, y)?.closest<HTMLElement>("[data-column-body]");
    if (body) {
      const r = body.getBoundingClientRect();
      if (y < r.top + EDGE_ZONE_PX) body.scrollTop -= EDGE_STEP_PX;
      else if (y > r.bottom - EDGE_ZONE_PX) body.scrollTop += EDGE_STEP_PX;
    }

    raf.current = requestAnimationFrame(autoScroll);
  }, []);

  /** Arm a press on a card; it only becomes a drag once it travels far enough. */
  const begin = useCallback((e: React.PointerEvent<HTMLElement>, cardId: string, fromColumnId: string) => {
    if (e.pointerType === "mouse" && e.button !== 0) return;
    // Clear any leftover suppression from a previous drag whose trailing click
    // never arrived (touch drags that end over another column often fire none),
    // so it cannot swallow this fresh tap.
    suppressClick.current = false;
    const rect = e.currentTarget.getBoundingClientRect();
    press.current = {
      pointerId: e.pointerId,
      cardId,
      fromColumnId,
      startX: e.clientX,
      startY: e.clientY,
      grabDX: e.clientX - rect.left,
      grabDY: e.clientY - rect.top,
      width: rect.width,
    };
  }, []);

  /** True once per completed drag: the caller should ignore that click. */
  const consumeClick = useCallback(() => {
    if (!suppressClick.current) return false;
    suppressClick.current = false;
    return true;
  }, []);

  useEffect(() => {
    const move = (e: PointerEvent) => {
      point.current = { x: e.clientX, y: e.clientY };
      const p = press.current;
      if (!p || p.pointerId !== e.pointerId) return;

      if (!active.current) {
        if (Math.hypot(e.clientX - p.startX, e.clientY - p.startY) < START_THRESHOLD_PX) return;
        const hit = hitTest(e.clientX, e.clientY);
        active.current = {
          cardId: p.cardId,
          fromColumnId: p.fromColumnId,
          x: e.clientX - p.grabDX,
          y: e.clientY - p.grabDY,
          width: p.width,
          overColumnId: hit?.columnId ?? p.fromColumnId,
          overIndex: hit?.index ?? 0,
        };
        setDrag(active.current);
        raf.current = requestAnimationFrame(autoScroll);
        return;
      }

      const hit = hitTest(e.clientX, e.clientY);
      active.current = {
        ...active.current,
        x: e.clientX - p.grabDX,
        y: e.clientY - p.grabDY,
        overColumnId: hit?.columnId ?? active.current.overColumnId,
        overIndex: hit?.index ?? active.current.overIndex,
      };
      setDrag(active.current);
      e.preventDefault(); // no text selection while dragging with a mouse
    };

    const finish = (commit: boolean) => {
      stopAutoScroll();
      const a = active.current;
      press.current = null;
      active.current = null;
      if (a) {
        suppressClick.current = true;
        setDrag(null);
        if (commit && a.overColumnId) onDropRef.current(a.cardId, a.overColumnId, a.overIndex);
      }
    };

    const up = () => finish(true);
    // Fires when the browser takes the gesture over (e.g. the column starts
    // scrolling vertically) — that was a scroll, not a drag.
    const cancel = () => finish(false);

    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
    window.addEventListener("pointercancel", cancel);
    return () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      window.removeEventListener("pointercancel", cancel);
      stopAutoScroll();
    };
  }, [autoScroll]);

  return { drag, begin, consumeClick };
}

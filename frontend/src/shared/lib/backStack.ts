import { useEffect, useRef } from "react";

// What the Android back gesture should close, in the order a person expects.
//
// The app had no answer for back at all: from any screen it closed the app,
// losing the session, the socket and the history — and the way back in was a
// cold start. The reason it had no answer is that nothing knew what was open.
// Dismissal was written seven separate times (Modal, drawer, media viewer,
// three menus, the picker), each with its own Escape listener, and half the
// overlays — side panels, selection mode, the actions row — had none at all.
// A synthetic Escape would therefore have closed some things and silently
// missed the rest.
//
// So overlays announce themselves here instead, and back closes the most
// recently opened one: last in, first out, which is the order they are stacked
// on screen. A modal inside a modal registers second and closes first.

type BackHandler = () => void;

interface Entry {
  id: number;
  run: BackHandler;
}

const stack: Entry[] = [];
let nextId = 1;

/** Registers a handler and returns its id. Exported for tests. */
export function pushBackHandler(run: BackHandler): number {
  const id = nextId++;
  stack.push({ id, run });
  return id;
}

/** Removes a handler by id, wherever it sits in the stack. */
export function removeBackHandler(id: number): void {
  const at = stack.findIndex((e) => e.id === id);
  if (at !== -1) stack.splice(at, 1);
}

/** Runs the topmost handler. Returns false when nothing is open. */
export function closeTopOverlay(): boolean {
  const top = stack[stack.length - 1];
  if (!top) return false;
  top.run();
  return true;
}

/** How many overlays currently claim the back gesture. Exported for tests. */
export function openOverlayCount(): number {
  return stack.length;
}

/**
 * Claims the back gesture while `open` is true.
 *
 * `onBack` is read at the moment back is pressed, so a handler that closes over
 * changing state does not need to re-register — re-registering would move it to
 * the top of the stack and quietly steal the gesture from whatever opened after
 * it.
 */
export function useBackHandler(open: boolean, onBack: BackHandler): void {
  const latest = useRef(onBack);
  latest.current = onBack;

  useEffect(() => {
    if (!open) return;
    const id = pushBackHandler(() => latest.current());
    return () => removeBackHandler(id);
  }, [open]);
}

export interface BackDecisionDeps {
  /** History index within the app; 0 means this is where the app opened. */
  historyIndex: number;
  goBack: () => void;
  /** Send the app to the background — never close it. */
  minimize: () => void;
}

/**
 * Decides what one press of back does.
 *
 * Minimising rather than closing at the end is deliberate: a backgrounded app
 * keeps its session, its socket and its place; a closed one has to cold-start,
 * and on this app that used to mean a password screen on a phone that has just
 * woken up.
 */
export function handleBackPress({ historyIndex, goBack, minimize }: BackDecisionDeps): void {
  if (closeTopOverlay()) return;
  if (historyIndex > 0) {
    goBack();
    return;
  }
  minimize();
}
